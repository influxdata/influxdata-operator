package influxdb

import (
	"context"
	"log"
	"reflect"

	influxdatav1alpha1 "github.com/dev9/prod/influxdata-operator/pkg/apis/influxdata/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Influxdb Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileInfluxdb{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("influxdb-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Influxdb
	err = c.Watch(&source.Kind{Type: &influxdatav1alpha1.Influxdb{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Influxdb
	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &influxdatav1alpha1.Influxdb{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileInfluxdb{}

// ReconcileInfluxdb reconciles a Influxdb object
type ReconcileInfluxdb struct {
	// TODO: Clarify the split client
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Influxdb object and makes changes based on the state read
// and what is in the Influxdb.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Influxdb Deployment for each Influxdb CR
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileInfluxdb) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling Influxdb %s/%s\n", request.Namespace, request.Name)

	// Fetch the Influxdb instance
	influxdb := &influxdatav1alpha1.Influxdb{}
	err := r.client.Get(context.TODO(), request.NamespacedName, influxdb)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Printf("Influxdb %s/%s not found. Ignoring since object must be deleted\n", request.Namespace, request.Name)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Printf("Failed to get Influxdb: %v", err)
		return reconcile.Result{}, err
	}

	// Check if the statefulset already exists, if not create a new one
	found := &appsv1.StatefulSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: influxdb.Name, Namespace: influxdb.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new statefulset
		dep := r.statefulsetForInfluxdb(influxdb)
		log.Printf("Creating a new Deployment %s/%s\n", dep.Namespace, dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			log.Printf("Failed to create new Deployment: %v\n", err)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		log.Printf("Failed to get Deployment: %v\n", err)
		return reconcile.Result{}, err
	}

	// Ensure the statefulset size is the same as the spec
	size := influxdb.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			log.Printf("Failed to update Deployment: %v\n", err)
			return reconcile.Result{}, err
		}
		// Spec updated - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// Update the Influxdb status with the pod names
	// List the pods for this influxdb's statefulset
	podList := &corev1.PodTemplateList{}
	labelSelector := labels.SelectorFromSet(labelsForInfluxdb(influxdb.Name))
	listOps := &client.ListOptions{Namespace: influxdb.Namespace, LabelSelector: labelSelector}
	err = r.client.List(context.TODO(), listOps, podList)
	if err != nil {
		log.Printf("Failed to list pods: %v", err)
		return reconcile.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, influxdb.Status.Nodes) {
		influxdb.Status.Nodes = podNames
		err := r.client.Update(context.TODO(), influxdb)
		if err != nil {
			log.Printf("failed to update influxdb status: %v", err)
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// statefulsetForInfluxdb returns a influxdb Deployment object
func (r *ReconcileInfluxdb) statefulsetForInfluxdb(m *influxdatav1alpha1.Influxdb) *appsv1.StatefulSet {
	ls := labelsForInfluxdb(m.Name)
	replicas := m.Spec.Size
	Image := m.Spec.Image

	dep := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "influxdb-svc",
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           Image,
						Name:            "influxdb",
						ImagePullPolicy: "Always",
						Ports: []corev1.ContainerPort{
							{
								Name:          "api",
								HostPort:      8086,
								ContainerPort: 8086,
								Protocol:      corev1.ProtocolTCP,
							},
							{
								Name:          "graphite",
								HostPort:      2003,
								ContainerPort: 2003,
								Protocol:      corev1.ProtocolTCP,
							},
							{
								Name:          "collectd",
								HostPort:      25826,
								ContainerPort: 25826,
								Protocol:      corev1.ProtocolTCP,
							},
							{
								Name:          "udp",
								HostPort:      8089,
								ContainerPort: 8089,
								Protocol:      corev1.ProtocolTCP,
							},
							{
								Name:          "opentsdb",
								HostPort:      4242,
								ContainerPort: 4242,
								Protocol:      corev1.ProtocolTCP,
							},
							{
								Name:          "backup-restore",
								HostPort:      8088,
								ContainerPort: 8088,
								Protocol:      corev1.ProtocolTCP,
							},
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("8"),
								corev1.ResourceMemory: resource.MustParse("16Gi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.1"),
								corev1.ResourceMemory: resource.MustParse("256Mi"),
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "influxdb-config",
								SubPath:   "influxdb.conf",
								MountPath: "/etc/influxdb/influxdb.conf",
							},
							{
								Name:      "influxdb-data",
								MountPath: "/var/lib/influxdb/",
							},
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "influxdb-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "influxdb-config",
									},
								},
							},
						},
						{
							Name: "influxdb-data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "influxdb-data-pvc",
								},
							},
						},
					},
				},
			},
		},
	}
	// Set Influxdb instance as the owner and controller
	controllerutil.SetControllerReference(m, dep, r.scheme)
	return dep
}

// labelsForInfluxdb returns the labels for selecting the resources
// belonging to the given influxdb CR name.
func labelsForInfluxdb(name string) map[string]string {
	return map[string]string{"app": "influxdb", "influxdb_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.PodTemplate) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
