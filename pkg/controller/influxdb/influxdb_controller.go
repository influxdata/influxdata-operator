package influxdb

import (
	"context"
	"log"
	"reflect"

	influxdatav1alpha1 "github.com/influxdata-operator/pkg/apis/influxdata/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

	log.Printf("Reconciling Influxdb Service\n")
	// Reconcile the cluster service
	err = r.reconcileService(influxdb)
	if err != nil {
		return reconcile.Result{}, err
	}

	log.Printf("Reconciling Influxdb Persistent Volume Claim\n")
	// Reconcile the cluster service
	err = r.reconcilePVC(influxdb)
	if err != nil {
		return reconcile.Result{}, err
	}

	log.Printf("Reconciling Influxdb StatefulSet\n")
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
	log.Printf("Matching size in spec")
	size := influxdb.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			log.Printf("Failed to update Deployment: %v\n", err)
			return reconcile.Result{}, err
		}

		log.Printf("Spec was updated, so request is getting re-queued")
		// Spec updated - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	log.Printf("Matching size in spec\n")
	// Update the Influxdb status with the pod names
	// List the pods for this influxdb's statefulset
	podList := &corev1.PodTemplateList{}
	labelSelector := labels.SelectorFromSet(labelsForInfluxdb(influxdb.Name))
	listOps := &client.ListOptions{Namespace: influxdb.Namespace, LabelSelector: labelSelector}
	err = r.client.List(context.TODO(), listOps, podList)
	if err != nil {
		log.Printf("Failed to list pods: %v\n", err)
		return reconcile.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	log.Printf("Updating status nodes\n")
	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, influxdb.Status.Nodes) {
		influxdb.Status.Nodes = podNames
		err := r.client.Update(context.TODO(), influxdb)
		if err != nil {
			log.Printf("failed to update influxdb status: %v", err)
			return reconcile.Result{}, err
		}
	}

	log.Printf("Reconcile completed successfully")
	return reconcile.Result{}, nil
}

// reconcileService ensures the Service is created.
func (r *ReconcileInfluxdb) reconcileService(cr *influxdatav1alpha1.Influxdb) error {
	// Check if this Service already exists
	found := &corev1.Service{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cr.Name + "-svc", Namespace: cr.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating a new Service %s/%s\n", cr.Namespace, cr.Name+"-svc")
		svc := newService(cr)

		// Set InfluxDB instance as the owner and controller
		if err = controllerutil.SetControllerReference(cr, svc, r.scheme); err != nil {
			log.Printf("Error setting controller reference: %v\n", err)
			return err
		}

		err = r.client.Create(context.TODO(), svc)
		if err != nil {
			log.Printf("Error creating client: %v\n", err)
			return err
		}

		// Service created successfully
		cr.Status.ServiceName = svc.Name
		err = r.client.Update(context.TODO(), cr)
		if err != nil {
			return err
		}

		log.Printf("Service %s/%s created successfully \n", cr.Namespace, cr.Name+"-svc")
		return nil
	} else if err != nil {
		return err
	}

	log.Printf("Skip reconcile: Service %s/%s already exists", found.Namespace, found.Name)
	return nil
}

// reconcilePVC ensures the Persistent Volume Claim is created.
func (r *ReconcileInfluxdb) reconcilePVC(cr *influxdatav1alpha1.Influxdb) error {
	// Check if this Persitent Volume Claim already exists
	found := &corev1.PersistentVolumeClaim{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cr.Spec.Pod.PersistentVolumeClaim.Name, Namespace: cr.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating a new Persistent Volume Claim %s/%s", cr.Namespace, cr.Spec.Pod.PersistentVolumeClaim.Name)
		pvc := newPVCs(cr)

		// Set InfluxDB instance as the owner and controller
		if err = controllerutil.SetControllerReference(cr, pvc, r.scheme); err != nil {
			log.Printf("%v", err)
			return err
		}

		err = r.client.Create(context.TODO(), pvc)
		if err != nil {
			log.Printf("%v", err)
			return err
		}

		// Persitent Volume Claim created successfully
		cr.Status.PersistentVolumeClaimName = pvc.Name
		err = r.client.Update(context.TODO(), cr)
		if err != nil {
			log.Printf("%v", err)
			return err
		}

		return nil
	} else if err != nil {
		log.Printf("%v", err)
		return err
	}

	log.Printf("Skip reconcile: Persistent Volume Claim %s/%s already exists", found.Namespace, found.Name)
	return nil
}

// statefulsetForInfluxdb returns a influxdb Deployment object
func (r *ReconcileInfluxdb) statefulsetForInfluxdb(m *influxdatav1alpha1.Influxdb) *appsv1.StatefulSet {
	ls := labelsForInfluxdb(m.Name)
	replicas := m.Spec.Size

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
			ServiceName: m.Name + "-svc",
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:    ls,
					Namespace: m.Namespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           m.Spec.BaseImage,
						Name:            "influxdb",
						ImagePullPolicy: m.Spec.ImagePullPolicy,
						Resources:       newContainerResources(m),
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
									ClaimName: m.Spec.Pod.PersistentVolumeClaim.Name,
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

// newContainerResources will create the container Resources for the InfluxDB Pod.
func newContainerResources(m *influxdatav1alpha1.Influxdb) corev1.ResourceRequirements {
	resources := corev1.ResourceRequirements{}
	if m.Spec.Pod != nil {
		resources = m.Spec.Pod.Resources
	}
	return resources
}

// newServicePorts constructs the ServicePort objects for the Service.
func newServicePorts(m *influxdatav1alpha1.Influxdb) []corev1.ServicePort {
	var ports []corev1.ServicePort

	ports = append(ports, corev1.ServicePort{Port: 8086, Name: "api"},
		corev1.ServicePort{Port: 2003, Name: "graphite"},
		corev1.ServicePort{Port: 25826, Name: "collectd"},
		corev1.ServicePort{Port: 8089, Name: "udp"},
		corev1.ServicePort{Port: 4242, Name: "opentsdb"},
		corev1.ServicePort{Port: 8088, Name: "backup-restore"},
	)
	return ports
}

// newService constructs a new Service object.
func newService(m *influxdatav1alpha1.Influxdb) *corev1.Service {
	ls := labelsForInfluxdb(m.Name)

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-svc",
			Namespace: m.Namespace,
			Labels:    ls,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Type:     "ClusterIP",
			Ports:    newServicePorts(m),
		},
	}
}

// newPVCs creates the PVCs used by the application.
func newPVCs(cr *influxdatav1alpha1.Influxdb) *corev1.PersistentVolumeClaim {
	ls := labelsForInfluxdb(cr.Name)

	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Spec.Pod.PersistentVolumeClaim.Name,
			Namespace: cr.Namespace,
			Labels:    ls,
		},
		Spec: cr.Spec.Pod.PersistentVolumeClaim.Spec,
	}
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
