package providers

import (
	"fmt"

	v1alpha1 "github.com/dev9-labs/influxdata-operator/pkg/apis/influxdata/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os"
)

// CommonProvider bonds together the required methods
type CommonProvider interface {
	CreateStorageClass(*v1.PersistentVolumeClaim) error
	GenerateMetadata() error
	determineParameters(*v1.PersistentVolumeClaim) (map[string]string, error)
	determineProvisioner(*v1.PersistentVolumeClaim) (string, error)
	CreateObjectStoreBucket(*v1alpha1.Influxdb) error
	CheckBucketExistence(*v1alpha1.Influxdb) (bool, error)
}

// DetermineProvider determines the cloud provider type based on metadata server
func DetermineProvider() (CommonProvider, error) {
	var providers = map[string]string{
		"azure":  "http://169.254.169.254/metadata/instance?api-version=2017-12-01",
		"aws":    "http://169.254.169.254/latest/meta-data/",
		"google": "http://169.254.169.254/0.1/meta-data/",
	}
	for key, value := range providers {
		req, err := http.NewRequest("GET", value, nil)
		if err != nil {
			logrus.Errorf("Could not create a proper http request %s", err.Error())
			return nil, err
		}
		req.Header.Set("Metadata", "true")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Something happened during the request %s", err.Error())
		}
		if resp.StatusCode == 404 || resp.StatusCode == 405 {
			continue
		}
		switch key {
		case "azure":
			return &AzureProvider{}, nil
		case "aws":
			return &AwsProvider{}, nil
		case "google":
			return &GoogleProvider{}, nil
		}

	}
	return nil, fmt.Errorf("could not determine cloud provider")
}

// CheckPersistentVolumeClaimExistence checks if the PVC already exists
func CheckPersistentVolumeClaimExistence(name, namespace string) bool {
	persistentVolumeClaim := &v1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := sdk.Get(persistentVolumeClaim); err != nil {
		logrus.Infof("PersistentVolumeClaim does not exists %s", err.Error())
		return false
	}
	logrus.Infof("PersistentVolumeClaim %s exist!", name)
	return true
}

// CheckStorageClassExistence checks if the storage class already exists
func CheckStorageClassExistence(name string) bool {
	storageClass := &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if err := sdk.Get(storageClass); err != nil {
		logrus.Infof("Storageclass does not exist %s", err.Error())
		return false
	}
	logrus.Infof("Storageclass %s exists!", name)
	return true
}

// asOwner returns an OwnerReference set as the memcached CR
func asOwner(m *v1beta1.Deployment) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: m.APIVersion,
		Kind:       m.Kind,
		Name:       m.Name,
		UID:        m.UID,
		Controller: &trueVar,
	}
}

//getOwner returns the Deployment created for the Operator
func getOwner() *v1beta1.Deployment {
	const operatorNamespace = "OPERATOR_NAMESPACE"

	deployment := &v1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      os.Getenv("OWNER_REFERENCE_NAME"),
			Namespace: os.Getenv(operatorNamespace),
		},
	}
	if err := sdk.Get(deployment); err != nil {
		logrus.Infof("PVC-handler does not exists! %s", err.Error())
		return nil
	}
	return deployment
}
