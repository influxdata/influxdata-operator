package stub

import (
	"context"
	"fmt"
	"reflect"

	v1alpha1 "github.com/dev9-labs/influxdata-operator/pkg/apis/influxdata/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	appsv1 "k8s.io/api/apps/v1"
	apps_v1beta2 "k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func NewHandler() sdk.Handler {
	return &Handler{}
}

type Handler struct {
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.Influxdb:
		influxdb := o

		// Ignore the delete event since the garbage collector will clean up all secondary resources for the CR
		// All secondary resources must have the CR set as their OwnerReference for this to be the case
		if event.Deleted {
			return nil
		}

		// Create the deployment if it doesn't exist
		dep := deploymentForInfluxdb(influxdb)
		err := sdk.Create(dep)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create deployment: %v", err)
		}

		// Ensure the deployment size is the same as the spec
		err = sdk.Get(dep)
		if err != nil {
			return fmt.Errorf("failed to get deployment: %v", err)
		}
		size := influxdb.Spec.Size
		if *dep.Spec.Replicas != size {
			dep.Spec.Replicas = &size
			err = sdk.Update(dep)
			if err != nil {
				return fmt.Errorf("failed to update deployment: %v", err)
			}
		}

		// Update the Influxdb status with the pod names
		podList := podList()
		labelSelector := labels.SelectorFromSet(labelsForInfluxdb(influxdb.Name)).String()
		listOps := &metav1.ListOptions{LabelSelector: labelSelector}
		err = sdk.List(influxdb.Namespace, podList, sdk.WithListOptions(listOps))
		if err != nil {
			return fmt.Errorf("failed to list pods: %v", err)
		}
		podNames := getPodNames(podList.Items)
		if !reflect.DeepEqual(podNames, influxdb.Status.Nodes) {
			influxdb.Status.Nodes = podNames
			err := sdk.Update(influxdb)
			if err != nil {
				return fmt.Errorf("failed to update influxdb status: %v", err)
			}
		}
	}
	return nil
}

// deploymentForInfluxdb returns a influxdb StatefulSet object
func deploymentForInfluxdb(m *v1alpha1.Influxdb) *appsv1.StatefulSet {
	ls := labelsForInfluxdb(m.Name)
	replicas := m.Spec.Size
	baseImage := m.Spec.Image

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
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Image:           baseImage,
						Name:            "influxdb",
						ImagePullPolicy: "Always",
						Ports: []v1.ContainerPort{
							v1.ContainerPort{
								Name:          "http",
								ContainerPort: 8086,
								Protocol:      v1.ProtocolTCP,
							},
						},
						Resources: v1.ResourceRequirements{
							Limits: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("8"),
								v1.ResourceMemory: resource.MustParse("16Gi"),
							},
							Requests: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("0.1"),
								v1.ResourceMemory: resource.MustParse("256Mi"),
							},
						},
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "influxdb-config",
								SubPath:   "config.toml",
								MountPath: "/etc/influxdb/influxdb.conf",
							},
						},
					}},
					Volumes: []v1.Volume{{
						Name: "influxdb-config",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "influxdb",
								},
							},
						},
					}},
				},
			},
			VolumeClaimTemplates: v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "influxdb-data",
					Namespace: m.Namespace,
					Labels:    ls,
				},
				Spec: v1.PersistentVolumeClaimSpec{
					accessModes:      "ReadWriteOnce",
					storageClassName: standard,
					storage:          "8Gi",
				},
			},
		},
	},
		addOwnerRefToObject(dep, asOwner(m))
	return dep
}

func makeInfluxDBService(m *v1alpha1.Influxdb) *appsv1.Service {
	ls := labelsForInfluxdb(m.Name)

	dep := &appsv1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   m.Name,
			Labels: ls,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "influx",
					Port:       8086,
					TargetPort: intstr.FromInt(8086),
					Protocol:   v1.ProtocolTCP,
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
		},
	}
	addOwnerRefToObject(dep, asOwner(m))
	return dep
}

// labelsForInfluxdb returns the labels for selecting the resources
// belonging to the given influxdb CR name.
func labelsForInfluxdb(name string) map[string]string {
	return map[string]string{"app": "influxdb", "influxdb_cr": name}
}

// addOwnerRefToObject appends the desired OwnerReference to the object
func addOwnerRefToObject(obj metav1.Object, ownerRef metav1.OwnerReference) {
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), ownerRef))
}

// asOwner returns an OwnerReference set as the influxdb CR
func asOwner(m *v1alpha1.Influxdb) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: m.APIVersion,
		Kind:       m.Kind,
		Name:       m.Name,
		UID:        m.UID,
		Controller: &trueVar,
	}
}

// podList returns a v1.PodList object
func podList() *v1.PodList {
	return &v1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []v1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
