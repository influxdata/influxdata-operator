package backup

import (
	"context"
	"github.com/influxdata-operator/pkg/storage"
	"io/ioutil"
	"log"
	"os"
	"time"

	influxdatav1alpha1 "github.com/influxdata-operator/pkg/apis/influxdata/v1alpha1"
	myremote "github.com/influxdata-operator/pkg/remote"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
)

const (
	backupDir = "/var/lib/influxdb/backup"
)

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

type ReconcileInfluxdbBackup struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileInfluxdbBackup{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("backup-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Backup
	err = c.Watch(&source.Kind{Type: &influxdatav1alpha1.Backup{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner Backup
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &influxdatav1alpha1.Backup{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileInfluxdbBackup{}

// This gets called when a Backup resource is created... I think.
func (r *ReconcileInfluxdbBackup) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Starting Influxdb Backup\n")

	// Fetch the Influxdb Backup instance
	backup := &influxdatav1alpha1.Backup{}
	err := r.client.Get(context.TODO(), request.NamespacedName, backup)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Printf("Influxdb Backup %s/%s not found. Ignoring since object must be deleted\n", request.Namespace, request.Name)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Printf("Failed to get Influxdb Backup: %v", err)
		return reconcile.Result{}, err
	}

	backupTime := time.Now().UTC().Format("20060102150405")

	output, err := r.execInPod(request.Namespace, []string{
		"influxd",
		"backup",
		"-portable",
		"-database " + backup.Spec.Databases[0],
		backupDir + "/" + backupTime,
	})
	if err != nil {
		log.Printf("Error occured while running backup command: %v", err)
		return reconcile.Result{}, err
	} else {
		log.Printf("Backup Output: %v", output)
	}

	sourceFile := request.NamespacedName.String() + ":" + backupDir + "/"+ backupTime
	destFile   := "/tmp/influxdb-backup/" + backupTime

	if err := os.MkdirAll(destFile, os.ModeType); err != nil {
		return reconcile.Result{}, err
	}

	if &backup.Spec.Storage.S3 != nil {
		if err := myremote.CopyFromPod(sourceFile, destFile); err != nil {
			log.Printf("error during copy from [%s] to [%s]: %v", sourceFile, destFile, err)
			return reconcile.Result{}, err
		}

		s3location, err := storeInS3(r.client, backup.Spec.Storage.S3, backupTime)

		if err != nil {
			log.Printf("Error during S3 storage: %v", err)
			return reconcile.Result{}, err
		}

		log.Printf("Backups stored to %s", s3location)
	}

	return reconcile.Result{}, nil
}

func storeInS3(client client.Client, backupStorage influxdatav1alpha1.S3BackupStorage, srcFolder string) (string, error) {
	provider, err := storage.NewS3StorageProvider(client, &backupStorage)
	if err != nil {
		return "", err
	}

	storageKey := backupStorage.Folder + "/" + srcFolder

	localFolder := "/tmp/influxdb-backup/" + srcFolder
	files, err := ioutil.ReadDir(localFolder)
	if err != nil {
		return "", err
	}

	// Loop through files in the source directory, send to s3
	for _, file := range files {
		localFile := localFolder + "/" + file.Name()
		f, err := os.Open(localFile)
		if err != nil {
			return "", err
		}

		err = provider.Store(storageKey + "/" + file.Name(), f)
		if err != nil {
			return "", err
		}
	}

	return "s3://" + backupStorage.Bucket + "/" + storageKey, nil
}

func (r *ReconcileInfluxdbBackup) execInPod(ns string, cmdOpts []string) (string, error) {
	cmd := strings.Join(cmdOpts, " ")

	// TODO:
	podName := "influxdb-0"
	containerName := "influxdb"

	output, stderr, err := ExecToPodThroughAPI(cmd, containerName, podName, ns, nil)

	if len(stderr) != 0 {
		log.Println("STDERR:", stderr)
	}

	if err != nil {
		log.Printf("Error occured while `exec`ing to the Pod %q, namespace %q, command %q. Error: %+v\n", podName, ns, cmd, err)
		return "", err
	} else {
		return output, nil
	}
}
