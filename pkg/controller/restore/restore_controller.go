package restore

import (
	"context"
	"fmt"
	"log"
	"strings"

	influxdatav1alpha1 "github.com/influxdata-operator/pkg/apis/influxdata/v1alpha1"
	backup "github.com/influxdata-operator/pkg/controller/backup"
	myremote "github.com/influxdata-operator/pkg/remote"
	storage2 "github.com/influxdata-operator/pkg/storage"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var RestoreDir = "/var/lib/influxdb/restore"

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

	k8s, err := myremote.NewK8sClient()
	if err != nil {
		log.Printf("Error occurred while getting K8s Client: %+v", err)
		return reconcile.Result{}, err
	}

	switch providerType := instance.Spec.Storage.Provider; providerType {
	case "s3":
		storage, err := storage2.NewS3StorageProvider(request.Namespace, r.client, &instance.Spec.Storage.S3)
		if err != nil {
			log.Printf("Error creating s3 storage provider: %v", err)
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
			log.Printf("localFileName is %s, size is %d\n", localFileName, *size)
			if err != nil {
				log.Printf("Unable to fetch key %s: %v", *key, err)
				return reconcile.Result{}, err
			}

			// Send directly to k8s pod
			destination := fmt.Sprintf("%s/%s:%s/%s%s", request.Namespace, instance.Spec.PodName,
				RestoreDir, instance.Spec.BackupId, localFileName)

			err = k8s.CopyToK8s(destination, size, &body)
			if err != nil {
				log.Printf("Error while copying to k8s: %+v\n", err)
				body.Close()
				return reconcile.Result{}, err
			}

			body.Close()
		}
		log.Println("Copy from S3 succeeded")

	case "gcs":
		// create a new storage client
		factory := &storage2.StorageClientFactoryGCS{CredentialsFile: storage2.CredentialsFile}

		provider, err := storage2.NewGcsStorageProvider(request.Namespace, r.client, &instance.Spec.Storage.Gcs, factory)
		if err != nil {
			log.Printf("error creating GCS storage provider: %v", err)
			return reconcile.Result{}, err
		}

		destinationBase := fmt.Sprintf("%s/%s:%s/%s", request.Namespace, instance.Spec.PodName,
			RestoreDir, instance.Spec.BackupId)
		remoteFolder := instance.Spec.Storage.Gcs.Folder + "/" + instance.Spec.BackupId
		err = provider.CopyFromGCS(remoteFolder, destinationBase, k8s)
		if err != nil {
			log.Printf("Error while CopyFromGCS: %v\n", err)
			return reconcile.Result{}, err
		}

		log.Println("Copy from GCS succeeded")
	case "pv":
		// uses the PV's backup data
		RestoreDir = backup.BackupDir
	default:
	}

	// Finally, we run the restore.
	pod, err := k8s.ClientSet.CoreV1().Pods(request.Namespace).Get(instance.Spec.PodName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return reconcile.Result{}, fmt.Errorf("cannot exec into a container in a completed pod; current phase is %s", pod.Status.Phase)
	}

	log.Printf("pod=%v\n", pod)

	restoreToDb := instance.Spec.RestoreToDatabase

	log.Printf("Restore DB: %s, To DB: %s, Backup key: %s", instance.Spec.Database, restoreToDb, instance.Spec.BackupId)

	command := []string{
		"influxd",
		"restore",
		"-portable",
	}

	if instance.Spec.Database != "" {
		command = append(command, "-db")
		command = append(command, instance.Spec.Database)
	}
	if restoreToDb != "" {
		command = append(command, "-newdb")
		command = append(command, restoreToDb)
	}
	if instance.Spec.Rp != "" && instance.Spec.Database != "" {
		command = append(command, "-rp")
		command = append(command, instance.Spec.Rp)
	}
	if instance.Spec.NewRp != "" && instance.Spec.Rp != "" {
		command = append(command, "-newrp")
		command = append(command, instance.Spec.NewRp)
	}
	if instance.Spec.Shard != "" && instance.Spec.Database != "" && instance.Spec.Rp != "" {
		command = append(command, "-shard")
		command = append(command, instance.Spec.Shard)
	}

	command = append(command, RestoreDir+"/"+instance.Spec.BackupId)

	log.Printf(strings.Join(command, " "))

	stdout, stderr, err := k8s.Exec(request.Namespace, instance.Spec.PodName, instance.Spec.ContainerName, command, nil)

	if stderr != nil && stderr.Len() != 0 {
		fmt.Printf("STDERR: %s\n", stderr.String())
	}

	if err != nil {
		fmt.Printf("Restore error: %v", err)
		return reconcile.Result{}, err
	}

	log.Printf("Restore output: %v\n", stdout)
	return reconcile.Result{}, nil
}
