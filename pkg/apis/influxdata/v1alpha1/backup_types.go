package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BackupSpec defines the specification for a backup.
type BackupSpec struct {
	Databases []string `json:"databases"`
	Storage BackupStorage `json:"storage"`
        PodName string `json:"podname"`
        ContainerName string `json:"containername"` 
        Retention string `json:"retention"`
        Shard string `json:"shard"`
        Start string `json:"start"`
        End string `json:"end"`
        Since string `json:"since"`
}

type BackupStorage struct {
	S3 S3BackupStorage `json:"s3,omitempty"`
}

type S3BackupStorage struct {
	AwsKeyId SecretRef `json:"aws_key_id"`
	AwsSecretKey SecretRef `json:"aws_secret_key"`
	Bucket string `json:"bucket"`
	Folder string `json:"folder"`
	Region string `json:"region"`
}

type SecretRef struct {
	ValueFrom ValueFrom `json:"valueFrom"`
}

type ValueFrom struct {
	SecretKeyRef KubernetesSecret `json:"secretKeyRef"`
}

type KubernetesSecret struct {
	Name      string `json:"name"`
	Key       string `json:"key"`
	Namespace string `json:"namespace"`
}

// BackupStatus defines the observed state of Backup
type BackupStatus struct {
	Location string `json:"location"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Backup is the Schema for the backups API
// +k8s:openapi-gen=true
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupSpec   `json:"spec,omitempty"`
	Status BackupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// BackupList contains a list of Backup
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Backup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Backup{}, &BackupList{})
}
