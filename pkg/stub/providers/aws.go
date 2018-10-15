package providers

import (
	"errors"

	v1alpha1 "github.com/dev9-labs/influxdata-operator/pkg/apis/influxdata/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AwsProvider holds info about Aws provider and allows us to implement the common interface
type AwsProvider struct {
}

// CreateStorageClass creates a StorageClass based on specs described on PVC
func (aws *AwsProvider) CreateStorageClass(pvc *v1.PersistentVolumeClaim) error {
	logrus.Info("Creating new storage class")
	provisioner, err := aws.determineProvisioner(pvc)
	if err != nil {
		return err
	}
	logrus.Info("Determining provisioner succeeded")
	parameter, err := aws.determineParameters(pvc)
	if err != nil {
		return err
	}
	logrus.Info("Determining parameter succeeded")
	return sdk.Create(&storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            *pvc.Spec.StorageClassName,
			Annotations:     nil,
			OwnerReferences: nil,
		},
		Provisioner:  provisioner,
		MountOptions: nil,
		Parameters:   parameter,
	})
}

// GenerateMetadata generates metadata which are needed to create a StorageClass
func (aws *AwsProvider) GenerateMetadata() error {
	return nil
}

// determineParameters determines the access mode from PVC
func (aws *AwsProvider) determineParameters(pvc *v1.PersistentVolumeClaim) (map[string]string, error) {
	//var parameter = map[string]string{}
	for _, mode := range pvc.Spec.AccessModes {
		switch mode {
		case "ReadWriteOnce":
			return nil, nil
		}
	}
	return nil, errors.New("could not determine parameters")
}

// determineProvisioner determines what kind of provisioner should the storage class use
func (aws *AwsProvider) determineProvisioner(pvc *v1.PersistentVolumeClaim) (string, error) {
	for _, mode := range pvc.Spec.AccessModes {
		switch mode {
		case "ReadWriteOnce":
			return "kubernetes.io/aws-ebs", nil
		case "ReadWriteMany":
			return "", errors.New("Not supported yet")
		case "ReadOnlyMany":
			return "", errors.New("Not supported yet")
		}
	}
	return "", errors.New("AccessMode is missing from the PVC")
}

// CheckBucketExistence checks if the bucket already exists
func (aws *AwsProvider) CheckBucketExistence(store *v1alpha1.Influxdb) (bool, error) {
	return false, nil
}

// CreateObjectStoreBucket creates a bucket in a cloud specific object store
func (aws *AwsProvider) CreateObjectStoreBucket(store *v1alpha1.Influxdb) error {
	return nil
}
