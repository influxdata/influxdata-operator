package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
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
	Database string
	Hostname string
}

// BackupExecutor represents the configuration of the tool performing the backup
type BackupExecutor struct {
	Config *InfluxdbBackupExecutor `json:"backup"`
}

// BackupSpec defines the specification for a backup. This includes what should be backed up,
// what tool should perform the backup, and, where the backup should be stored.
type BackupSpec struct {
	// Executor is the configuration of the tool that will produce the backup, and a definition of
	// what databases and tables to backup.
	Executor BackupExecutor `json:"executor"`

	// ScheduledMember is the Pod name of the Cluster member on which the
	// Backup will be executed.
	ScheduledMember string `json:"scheduledMember"`
}

// BackupConditionType represents a valid condition of a Backup.
type BackupConditionType string

const (
	// BackupScheduled means the Backup has been assigned to a Cluster
	// member for execution.
	BackupScheduled BackupConditionType = "Scheduled"
	// BackupRunning means the Backup is currently being executed by a
	// Cluster member's sidecar
	BackupRunning BackupConditionType = "Running"
	// BackupComplete means the Backup has successfully executed
	BackupComplete BackupConditionType = "Complete"
	// BackupFailed means the Backup has failed.
	BackupFailed BackupConditionType = "Failed"
)

// BackupCondition describes the observed state of a Backup at a certain point.
type BackupCondition struct {
	Type   BackupConditionType
	Status corev1.ConditionStatus
	// +optional
	LastTransitionTime metav1.Time
	// +optional
	Reason string
	// +optional
	Message string
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
	// +optional
	Conditions []BackupCondition
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// BackupList is a list of Backups.
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Backup `json:"items"`
}

// RestoreConditionType represents a valid condition of a Restore.
type RestoreConditionType string

const (
	// RestoreScheduled means the Restore has been assigned to a Cluster
	// member for execution.
	RestoreScheduled RestoreConditionType = "Scheduled"
	// RestoreRunning means the Restore is currently being executed by a
	// Cluster member's mysql-agent side-car.
	RestoreRunning RestoreConditionType = "Running"
	// RestoreComplete means the Restore has successfully executed and the
	// resulting artifact has been stored in object storage.
	RestoreComplete RestoreConditionType = "Complete"
	// RestoreFailed means the Restore has failed.
	RestoreFailed RestoreConditionType = "Failed"
)

// RestoreCondition describes the observed state of a Restore at a certain point.
type RestoreCondition struct {
	Type   RestoreConditionType
	Status corev1.ConditionStatus
	// +optional
	LastTransitionTime metav1.Time
	// +optional
	Reason string
	// +optional
	Message string
}

// RestoreSpec defines the specification for a restore of a MySQL backup.
type RestoreSpec struct {
	// Cluster is a refeference to the Cluster to which the Restore
	// belongs.
	Cluster *corev1.LocalObjectReference `json:"cluster"`
	// Backup is a reference to the Backup object to be restored.
	Backup *corev1.LocalObjectReference `json:"backup"`
	// ScheduledMember is the Pod name of the Cluster member on which the
	// Restore will be executed.
	ScheduledMember string `json:"scheduledMember"`
}

// RestoreStatus captures the current status of a MySQL restore.
type RestoreStatus struct {
	// TimeStarted is the time at which the restore was started.
	// +optional
	TimeStarted metav1.Time `json:"timeStarted"`
	// TimeCompleted is the time at which the restore completed.
	// +optional
	TimeCompleted metav1.Time `json:"timeCompleted"`
	// +optional
	Conditions []RestoreCondition
}

// +genclient
// +genclient:noStatus
// +resourceName=influxdbrestore
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Restores the influxdb
type Restore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   RestoreSpec   `json:"spec"`
	Status RestoreStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RestoreList is a list of Restores.
type RestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Restore `json:"items"`
}



func init() {
	SchemeBuilder.Register(&Influxdb{}, &InfluxdbList{},
		&BackupList{}, &RestoreList{}, &Backup{}, &Restore{})
}
