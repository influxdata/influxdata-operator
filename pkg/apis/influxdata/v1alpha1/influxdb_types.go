package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// InfluxdbSpec defines the desired state of Influxdb
type InfluxdbSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Size  int32  `json:"size"`
	Image string `json:"image"`
}

// InfluxdbStatus defines the observed state of Influxdb
type InfluxdbStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Nodes []string `json:"nodes"`
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


// Below borrowed from mysql operator.

type InfluxdbBackupExecutor struct {
	Database string `json:"database"`
	Hostname string `json:"hostname"`
}

// BackupSpec defines the specification for a backup. This includes what should be backed up,
// what tool should perform the backup, and, where the backup should be stored.
type BackupSpec struct {
	// Executor is the configuration of the tool that will produce the backup, and a definition of
	// what databases and tables to backup.
	Backup InfluxdbBackupExecutor `json:"backup"`
}

// BackupOutcome describes the location of a Backup
type BackupOutcome struct {
	// Location is the Object Storage network location of the Backup.
	Location string `json:"location"`
}

// BackupStatus captures the current status of a Backup.
type BackupStatus struct {
	// Outcome holds the results of a successful backup.
	// +optional
	Outcome BackupOutcome `json:"outcome"`
	// TimeStarted is the time at which the backup was started.
	// +optional
	TimeStarted metav1.Time `json:"timeStarted"`
	// TimeCompleted is the time at which the backup completed.
	// +optional
	TimeCompleted metav1.Time `json:"timeCompleted"`
}

// +genclient
// +genclient:noStatus
// +resourceName=influxdbbackups
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Backup is a backup of a Cluster.
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   BackupSpec   `json:"spec"`
	Status BackupStatus `json:"status"`
}

// RestoreConditionType represents a valid condition of a Restore.
//type RestoreConditionType string
//
//const (
//	// RestoreScheduled means the Restore has been assigned to a Cluster
//	// member for execution.
//	RestoreScheduled RestoreConditionType = "Scheduled"
//	// RestoreRunning means the Restore is currently being executed by a
//	// Cluster member's mysql-agent side-car.
//	RestoreRunning RestoreConditionType = "Running"
//	// RestoreComplete means the Restore has successfully executed and the
//	// resulting artifact has been stored in object storage.
//	RestoreComplete RestoreConditionType = "Complete"
//	// RestoreFailed means the Restore has failed.
//	RestoreFailed RestoreConditionType = "Failed"
//)
//
//// +genclient
//// +genclient:noStatus
//// +resourceName=influxdbrestore
//// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//
//// Restores the influxdb
//type Restore struct {
//	metav1.TypeMeta   `json:",inline"`
//	metav1.ObjectMeta `json:"metadata"`
//
//	Spec   RestoreSpec   `json:"spec"`
//	Status RestoreStatus `json:"status"`
//}
//
//// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//
//// RestoreList is a list of Restores.
//type RestoreList struct {
//	metav1.TypeMeta `json:",inline"`
//	metav1.ListMeta `json:"metadata"`
//
//	Items []Restore `json:"items"`
//}



func init() {
	SchemeBuilder.Register(&Influxdb{}, &InfluxdbList{},
		&Backup{})
}
