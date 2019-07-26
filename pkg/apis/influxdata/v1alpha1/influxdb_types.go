package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PodPolicy defines the policy for pods owned by InfluxDB operator.
type PodPolicy struct {
	// Resources is the resource requirements for the containers.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// PersistentVolumeClaim is the claim to describe PVC for the influxdb container
	PersistentVolumeClaim *corev1.PersistentVolumeClaim `json:"persistentVolumeClaim,omitempty"`
}

// InfluxdbSpec defines the desired state of Influxdb
type InfluxdbSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Size is the number of Pods to create for the Influxdb cluster. Default: 1
	Size int32 `json:"size"`
	// BaseImage is the Influxdb container image to use for the Pods.
	BaseImage string `json:"baseImage"`
	// ImagePullPolicy defines how the image is pulled.
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy"`
	//  Affinitu defines which nodes your pod is eligible to be scheduled on, based on labels on the node.
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Tolerations are applied to pods, and allow the pods to schedule onto nodes with matching taints.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// Pod defines the policy for pods owned by InfluxDB operator.
	// This field cannot be updated once the CR is created.
	Pod *PodPolicy `json:"pod,omitempty"`
}

// InfluxdbStatus defines the observed state of Influxdb
type InfluxdbStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Nodes []string `json:"nodes"`
	// ServiceName is the name of the Service for accessing the pods in the cluster.
	ServiceName string `json:"serviceName"`
	// PersistentVolumeClaimName is the name of the PVC backing the pods in the cluster.
	PersistentVolumeClaimName string `json:"persistentVolumeClaimName"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Influxdb is the Schema for the influxdbs API
// +k8s:openapi-gen=true
type Influxdb struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InfluxdbSpec   `json:"spec,omitempty"`
	Status InfluxdbStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfluxdbList contains a list of Influxdb
type InfluxdbList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Influxdb `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Influxdb{}, &InfluxdbList{})
}
