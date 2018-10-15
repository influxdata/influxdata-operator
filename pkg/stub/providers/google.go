package providers

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"

	"github.com/influxdata-operator2/pkg/apis/influxdata/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

// GoogleProvider holds info about Google provider and allows us to implement the common interface
type GoogleProvider struct {
	projectId string
}

// CreateStorageClass creates a StorageClass based on specs described on PVC
func (gke *GoogleProvider) CreateStorageClass(pvc *v1.PersistentVolumeClaim) error {
	logrus.Info("Creating new storage class")
	provisioner, err := gke.determineProvisioner(pvc)
	if err != nil {
		return err
	}
	logrus.Info("Determining provisioner succeeded")
	parameter, err := gke.determineParameters(pvc)
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
func (gke *GoogleProvider) GenerateMetadata() error {
	return nil
}

// determineParameters determines the access mode from PVC
func (gke *GoogleProvider) determineParameters(pvc *v1.PersistentVolumeClaim) (map[string]string, error) {
	//var parameter = map[string]string{}
	for _, mode := range pvc.Spec.AccessModes {
		switch mode {
		case "ReadWriteOnce", "ReadOnlyMany":
			return nil, nil
		}
	}
	return nil, errors.New("could not determine parameters")
}

// determineProvisioner determines what kind of provisioner should the storage class use
func (gke *GoogleProvider) determineProvisioner(pvc *v1.PersistentVolumeClaim) (string, error) {
	for _, mode := range pvc.Spec.AccessModes {
		switch mode {
		case "ReadWriteOnce", "ReadOnlyMany":
			return "kubernetes.io/gce-pd", nil
		case "ReadWriteMany":
			return "", errors.New("Not supported yet")
		}
	}
	return "", errors.New("AccessMode is missing from the PVC")
}

// determineProjectId determines the project ID from the metadata server
func (gke *GoogleProvider) determineProjectId() error {
	logrus.Info("Getting ProjectID from Metadata service")
	req, err := http.NewRequest("GET", "http://169.254.169.254/0.1/meta-data/project-id", nil)
	if err != nil {
		logrus.Errorf("Error during getting project-id, %s", err.Error())
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logrus.Errorf("Error during getting project-id, %s", err.Error())
		return err
	}
	readProjectId, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Error during reading response %s", err.Error())
		return err
	}
	gke.projectId = string(readProjectId)
	return nil
}

// CreateObjectStoreBucket creates a bucket in a cloud specific object store
func (gke *GoogleProvider) CreateObjectStoreBucket(app *v1alpha1.Influxdb) error {
	ctx := context.Background()
	logrus.Info("Creating new storage client")
	client, err := storage.NewClient(ctx)
	if err != nil {
		logrus.Fatalf("Failed to create client: %v", err)
		return err
	}
	logrus.Info("Storage client created successfully")

	bucket := client.Bucket(app.Spec.Name)
	gke.determineProjectId()
	if err := bucket.Create(ctx, gke.projectId, nil); err != nil {
		logrus.Fatalf("Failed to create bucket: %v", err)
		return err
	}
	logrus.Infof("%s bucket created", app.Spec.Name)
	return nil
}

// CheckBucketExistence checks if the bucket already exists
func (gke *GoogleProvider) CheckBucketExistence(app *v1alpha1.Influxdb) (bool, error) {
	return false, nil
}
