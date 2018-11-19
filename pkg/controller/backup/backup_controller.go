package backup

import (
	"context"
	"fmt"
	"github.com/influxdata-operator/pkg/storage"
	"io/ioutil"
	"log"
	"os"
	"strings"
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
)

const (
	BackupDir = "/var/lib/influxdb/backup"
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

// This gets called when a Backup resource is created...
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

	k8s, err := myremote.NewK8sClient()
	if err != nil {
		log.Printf("Error occurred while getting K8s Client: %+v", err)
		return reconcile.Result{}, err
	}

	backupTime, err := r.runBackup(k8s, backup)
	if err != nil {
		log.Printf("Error running backup: %+v", err)
		return reconcile.Result{}, err
	}

	sourceFile := fmt.Sprintf("%s/%s:%s/%s", request.Namespace, backup.Spec.PodName, BackupDir, backupTime)
	destFile := os.TempDir() + "/influxdb-backup/" + backupTime

	if err := os.MkdirAll(destFile, os.ModePerm); err != nil {
		log.Printf("Unable to make directories: %v\n", err)
		return reconcile.Result{}, err
	}

	if &backup.Spec.Storage.S3 != nil {
		provider, err := storage.NewS3StorageProvider(request.Namespace, r.client, &backup.Spec.Storage.S3)

		if err != nil {
			log.Printf("error creating s3 storage provider: %v", err)
			return reconcile.Result{}, err
		}

		log.Println("Shipping influx backup to S3...")
		if err := k8s.CopyFromK8s(sourceFile, destFile); err != nil {
			log.Printf("error during copy from [%s] to [%s]: %v\n", sourceFile, destFile, err)
			return reconcile.Result{}, err
		}

		s3location, err := storeInS3(provider, backup.Spec.Storage.S3, backupTime, destFile)

		if err != nil {
			log.Printf("Error during S3 storage: %v\n", err)
			return reconcile.Result{}, err
		}

		log.Printf("Backups stored to %s\n", s3location)
	}

	log.Println("Done with reconcile!")
	return reconcile.Result{}, nil
}

// returns (backuptime, error)
func (r *ReconcileInfluxdbBackup) runBackup(k8s *myremote.K8sClient, backup *influxdatav1alpha1.Backup) (string, error) {
	backupTime := time.Now().UTC().Format("20060102150405")

	cmdOpts := []string{
		"influxd",
		"backup",
		"-portable",
	}

	if backup.Spec.Databases != nil && backup.Spec.Databases[0] != "" {
		cmdOpts = append(cmdOpts, "-database")
		cmdOpts = append(cmdOpts, backup.Spec.Databases[0])
	}
	if backup.Spec.Databases != nil && backup.Spec.Retention != "" {
		cmdOpts = append(cmdOpts, "--retention")
		cmdOpts = append(cmdOpts, backup.Spec.Retention)
	}
	if backup.Spec.Retention != "" && backup.Spec.Shard != "" {
		cmdOpts = append(cmdOpts, "-shard")
		cmdOpts = append(cmdOpts, backup.Spec.Shard)
	}
	if backup.Spec.Start != "" {
		cmdOpts = append(cmdOpts, "-start")
		cmdOpts = append(cmdOpts, backup.Spec.Start)
	}
	if backup.Spec.End != "" {
		cmdOpts = append(cmdOpts, "-end")
		cmdOpts = append(cmdOpts, backup.Spec.End)
	}
	if backup.Spec.Start == "" && backup.Spec.End == "" && backup.Spec.Since != "" {
		cmdOpts = append(cmdOpts, "-since")
		cmdOpts = append(cmdOpts, backup.Spec.Since)
	}
	cmdOpts = append(cmdOpts, BackupDir+"/"+backupTime)

	log.Printf("Command being run: %s", strings.Join(cmdOpts, " "))

	// TODO: Parameterize? Get from config?
	podName := backup.Spec.PodName
	container := backup.Spec.ContainerName
	outputBytes, stderrBytes, err := k8s.Exec(backup.Namespace, podName, container, cmdOpts, nil)

	stderr := stderrBytes.String()
	output := outputBytes.String()

	if stderrBytes != nil {
		log.Println("STDERR: ", stderr)
	}

	if err != nil {
		log.Printf("Error occured while running backup command: %v", err)
		log.Printf("Stdout: %v", output)
		return "", err
	} else {
		log.Printf("Backup Output: %v", output)
	}

	return backupTime, nil
}

func storeInS3(provider *storage.S3StorageProvider, backupStorage influxdatav1alpha1.S3BackupStorage, name, srcFolder string) (string, error) {
	storageKey := backupStorage.Folder + "/" + name

	localFolder := srcFolder
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

		log.Printf("Storing To S3: [%s] to [%s]\n", localFile, storageKey+"/"+file.Name())
		err = provider.Store(storageKey+"/"+file.Name(), f)
		if err != nil {
			return "", err
		}
	}

	return "s3://" + backupStorage.Bucket + "/" + storageKey, nil
}
