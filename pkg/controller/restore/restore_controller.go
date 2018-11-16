package restore

import (
	"context"
	"fmt"
	influxdatav1alpha1 "github.com/influxdata-operator/pkg/apis/influxdata/v1alpha1"
	"github.com/influxdata-operator/pkg/controller/backup"
	myremote "github.com/influxdata-operator/pkg/remote"
	storage2 "github.com/influxdata-operator/pkg/storage"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
)

const (
	RestoreDir = "/var/lib/influxdb/restore"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Restore Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRestore{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("restore-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Restore
	err = c.Watch(&source.Kind{Type: &influxdatav1alpha1.Restore{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Restore
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &influxdatav1alpha1.Restore{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileRestore{}

type ReconcileRestore struct {
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileRestore) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling Restore %s/%s\n", request.Namespace, request.Name)

	// Fetch the Restore instance
	instance := &influxdatav1alpha1.Restore{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	restoreToDb := instance.Spec.RestoreToDatabase
	if len(restoreToDb) <= 0 {
		restoreToDb = instance.Spec.Database
	}

	log.Printf("Restore DB: %s, To DB: %s, Backup key: %s", instance.Spec.Database, restoreToDb, instance.Spec.BackupId)

	storage, err := storage2.NewS3StorageProvider(request.Namespace, r.client, &instance.Spec.Storage.S3)
	if err != nil {
		log.Printf("Error creating s3 storage provider: %v", err)
		return reconcile.Result{}, err
	}

	k8s, err := myremote.NewK8sClient()
	if err != nil {
		log.Printf("Error occurred while getting K8s Client: %+v", err)
		return reconcile.Result{}, err
	}

	remoteFolder := instance.Spec.Storage.S3.Folder + "/" + instance.Spec.BackupId

	keys, err := storage.ListDirectory(remoteFolder)
	if err != nil {
		log.Printf("Error while listing directory: %v\n", err)
		return reconcile.Result{}, err
	}

	for _, key := range keys {
		// need to strip off prefix, just want name.
		localFileName := strings.TrimPrefix(*key, remoteFolder)

		body, size, err := storage.Retrieve(*key)
		if err != nil {
			log.Printf("Unable to fetch key %s: %v", *key, err)
			return reconcile.Result{}, err
		}

		// Send directly to k8s pod
		destination := fmt.Sprintf("%s/%s:%s/%s%s", request.Namespace, backup.FixedPodName,
			RestoreDir, instance.Spec.BackupId, localFileName)

		err = k8s.CopyToK8s(destination, size, &body)
		if err != nil {
			fmt.Printf("Error while copying to k8s: %+v\n", err)
			body.Close()
			return reconcile.Result{}, err
		}

		body.Close()
	}
	fmt.Println("Copy from S3 succeeded")

	// Finally, we run the restore.
	pod, err := k8s.ClientSet.CoreV1().Pods(request.Namespace).Get(backup.FixedPodName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return reconcile.Result{}, fmt.Errorf("cannot exec into a container in a completed pod; current phase is %s", pod.Status.Phase)
	}

	command := []string{
		"influxd",
		"restore",
		"-portable",
		"-db",
		instance.Spec.Database,
		"-newdb",
		restoreToDb,
		fmt.Sprintf("%s/%s", RestoreDir, instance.Spec.BackupId),
	}

	stdout, stderr, err := k8s.Exec(request.Namespace, backup.FixedPodName, pod.Spec.Containers[0].Name, command, nil)

	if stderr != nil && stderr.Len() != 0 {
		fmt.Printf("STDERR: %s\n", stderr.String())
	}

	if err != nil {
		fmt.Printf("Restore error: %v", err)
		return reconcile.Result{}, err
	}

	fmt.Printf("Restore output: %v\n", stdout)
	return reconcile.Result{}, nil
}
