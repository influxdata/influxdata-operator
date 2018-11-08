package restore

import (
	"context"
	"log"
	"strings"

	influxdatav1alpha1 "github.com/influxdata-operator/pkg/apis/influxdata/v1alpha1"
	"github.com/influxdata-operator/pkg/controller/backup"
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

// ReconcileRestore reconciles a Restore object
type ReconcileRestore struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Restore object and makes changes based on the state read
// and what is in the Restore.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
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

	log.Printf("Restore DB: %s, To DB: %s_restore, Backup key: %s", instance.Spec.Database, instance.Spec.Database, instance.Spec.Location)

	return reconcile.Result{}, nil
	output, err := r.execInPod(request.Namespace, []string{
		"influxd",
		"restore",
		"-portable",
		"-db " + instance.Spec.Database,
		"-newdb " + instance.Spec.Database + "_restore",
		instance.Spec.Location,
	})
	if err != nil {
		log.Printf("Error occured while restoring backup: %v", err)
		return reconcile.Result{}, err
	} else {
		log.Printf("Restore Output: %v", output)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileRestore) execInPod(ns string, cmdOpts []string) (string, error) {
	cmd := strings.Join(cmdOpts, " ")

	// TODO: Parameterize?
	podName := "influxdb-0"
	containerName := "influxdb"

	output, stderr, err := backup.ExecToPodThroughAPI(cmd, containerName, podName,	ns, nil)

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

