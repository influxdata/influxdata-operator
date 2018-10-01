package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type InfluxdbList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Influxdb `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Influxdb struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              InfluxdbSpec   `json:"spec"`
	Status            InfluxdbStatus `json:"status,omitempty"`
}

type InfluxdbSpec struct {
	// Size is the size of the memcached deployment
	Size int32 `json:"size"`
}
type InfluxdbStatus struct {
	// Nodes are the names of the memcached pods
	Nodes []string `json:"nodes"`
}
