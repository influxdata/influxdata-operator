package storage

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/influxdata-operator/pkg/apis/influxdata/v1alpha1"
	influxdatav1alpha1 "github.com/influxdata-operator/pkg/apis/influxdata/v1alpha1"
	myremote "github.com/influxdata-operator/pkg/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// a mock factory for GCS
type MockStorageClientFactory struct {
	mock *storage.Client
}

// implement GCS factory
func (m *MockStorageClientFactory) NewStorageClient() (*storage.Client, error) {
	return m.mock, nil
}

// Test NewStorageClient
func TestNilNewStorageClient(t *testing.T) {
	factory := StorageClientFactoryGCS{CredentialsFile: CredentialsFile}
	client, err := factory.NewStorageClient()
	assert.Nil(t, client)
	assert.NotNil(t, err)
	assert.Error(t, err)
}

// Test NewGcsStorageProvider
func TestNewGcsStorageProvider(t *testing.T) {
	mt := &mockTransport{}
	body := "{}"
	mt.addResult(&http.Response{StatusCode: 200, Body: bodyReader(body)}, nil)
	mockGCSFactory := &MockStorageClientFactory{mock: mockClient(t, mt)}

	tests := []struct {
		name            string
		secretNamespace string
		k8snamespace    string
		secretName      string
		keyName         string
		factory         StorageClientFactory
		expectedJSON    string
		expectedError   error
	}{
		{
			name:            "valid factory",
			secretNamespace: "default",
			k8snamespace:    "default",
			secretName:      "influxdb-backup-gcs",
			keyName:         "sa",
			expectedJSON:    "{gcs-service-account-json-base64}",
			factory:         mockGCSFactory,
			expectedError:   nil,
		},
		{
			name:            "invalid sa",
			secretNamespace: "",
			k8snamespace:    "default",
			secretName:      "influxdb-backup-gcs",
			keyName:         "sa",
			expectedJSON:    "{gcs-service-account-json-base64}",
			factory:         mockGCSFactory,
			expectedError:   nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := testclient.NewFakeClient(NewSecret(test.secretNamespace, test.secretName, test.keyName, test.expectedJSON))
			spec := NewGCSSpecWithSecret(test.secretNamespace, test.secretName, test.keyName)
			provider, err := NewGcsStorageProvider(test.k8snamespace, client, spec, test.factory)
			if err != nil {
				require.Error(t, err)
				//t.Logf("TestNewGcsStorageProvider err: %s, %v", err, provider)
				assert.Nil(t, provider)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, provider)
		})
	}
}

func TestStoreSuccess(t *testing.T) {
	key := "backup/20190122062557/20190122T062557Z.manifest"
	body := fmt.Sprintf("{\"name\": \"%s\"}", key)
	provider := NewMockGcsStorageProvider(t, body, 1)
	uploadData := "testdata/upload.manifest"
	f, err := os.Open(uploadData)
	if err != nil {
		t.Fatalf("TestStore open loal file: %+v", err)
		return
	}
	err2 := provider.Store(key, f)
	if err2 != nil {
		t.Fatalf("TestStore store error: %v", err2)
		return
	}
	assert.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestRetrieveSuccess(t *testing.T) {
	body := string(helperLoadBytes(t, "download.manifest"))
	key := "backup/download.manifest"
	provider := NewMockGcsStorageProvider(t, body, 1)
	rc, _, err := provider.Retrieve(key)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("Retrieve fails %v", err))
		return
	}
	f, err := os.Create("/tmp/download.manifest")
	if err != nil {
		assert.Fail(t, fmt.Sprintf("Retrieve fails %v", err))
		return
	}
	_, copyErr := io.Copy(f, rc)

	if copyErr != nil {
		assert.Fail(t, fmt.Sprintf("copyErr fails %v", copyErr))
		return
	}
	assert.NoError(t, err)
}

func TestListDirectory(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		testDatafile string
	}{
		{
			name:         "list-objects",
			key:          "backup/20190122062557",
			testDatafile: "list_objects.json",
		},
		{
			name:         "list-empty",
			key:          "backup/2019012345555",
			testDatafile: "empty.json",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := string(helperLoadBytes(t, test.testDatafile))
			provider := NewMockGcsStorageProvider(t, body, 1)
			objects, err := provider.ListDirectory(test.key)
			if err != nil {
				assert.Error(t, err)
				assert.Nil(t, objects)
				t.Log(err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, provider)
		})
	}
}
func TestGetSaJson(t *testing.T) {
	tests := []struct {
		name            string
		secretNamespace string
		k8snamespace    string
		secretName      string
		keyName         string
		wantJSON        string
		wantError       string
	}{
		{
			name:            "found-servie-account-key",
			secretNamespace: "default",
			k8snamespace:    "default",
			secretName:      "influxdb-backup-gcs",
			keyName:         "sa",
			wantJSON:        "{gcs-service-account-json-base64}",
		},
		{
			name:            "missing-secret",
			secretNamespace: "",
			k8snamespace:    "default",
			secretName:      "",
			keyName:         "",
			wantJSON:        "",
			wantError:       "secrets \"\" not found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// create a client with secret or a client without secret
			var client client.Client
			if test.secretName != "" {
				client = testclient.NewFakeClient(NewSecret(test.secretNamespace, test.secretName, test.keyName, test.wantJSON))
			} else {
				client = testclient.NewFakeClient()
			}
			spec := NewGCSSpecWithSecret(test.secretNamespace, test.secretName, test.keyName)
			saJSON, err := getSaJSON(test.k8snamespace, client, spec)
			if err != nil {
				require.Error(t, err)
				assert.Empty(t, saJSON)
				assert.Equal(t, test.wantError, err.Error())
				return
			}
			assert.Nil(t, err)
			assert.Equal(t, test.wantJSON, saJSON)
		})
	}
}

func TestCopyToGCS(t *testing.T) {
	spec := NewGCSSpec()
	backupTime := "20190124234601"
	srcFolder := "testdata/backup"
	want := "gs://influxdb-backup-restore/backup/20190124234601"
	files, err := ioutil.ReadDir(srcFolder)
	if err != nil {
		t.Fatalf("can't read %s", srcFolder)
	}
	provider := NewMockGcsStorageProvider(t, "{}", len(files))
	url, err := provider.CopyToGCS(spec, backupTime, srcFolder)
	if err != nil {
		assert.Empty(t, url)
		assert.Error(t, err)
		t.Log(err)
		return
	}
	assert.NoError(t, err)
	assert.Equal(t, want, url)
}

func TestCopyFromGCS(t *testing.T) {

	remoteFolder := "backup/20190122062557"
	destinationBase := "default/influxdb-0:testdata/restore/20190122062557"

	// add fake provider.ListDirectory response
	mt := &mockTransport{}
	testDatafile := "list_objects_CopyFromGCS.json"
	dataListDirectory := string(helperLoadBytes(t, testDatafile))
	var bodys []string
	bodys = append(bodys, dataListDirectory)
	bodys = append(bodys, "{}")
	bodys = append(bodys, "{}")
	bodys = append(bodys, "{}")

	for _, body := range bodys {
		mt.addResult(&http.Response{StatusCode: 200, Body: bodyReader(body)}, nil)
	}
	client := mockClient(t, mt)
	spec := NewGCSSpecWithSecret("default", "influxdb-backup-gcs", "sa")
	bh := &bucket{bh: client.Bucket(spec.Bucket)}
	provider := &GcsStorageProvider{bucketHandle: bh, spec: spec, k8snamespace: "default"}

	// create a new mock of client-go with the required Runtime object by the test
	objs := []runtime.Object{pod(corev1.NamespaceDefault, "influxdb-0")}
	fakek8sClient, err := myremote.NewMockK8sClient(objs)
	if err != nil {
		t.Fatalf("can't create new fakek8sClient  %v", err)
	}
	err = provider.CopyFromGCS(remoteFolder, destinationBase, fakek8sClient)
	if err != nil {
		t.Fatalf("CopyFrom GCS failed %v", err)
	}
	assert.NoError(t, err)
	t.Log("TestCopyFromGCS works")
}

// helper func to create a pod
func pod(namespace, name string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
}

//helper func to create a k8s secret
func NewSecret(secretNamespace, secretName, keyName, keyValue string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: secretNamespace,
			Name:      secretName,
		},
		Data: map[string][]byte{
			keyName: []byte(keyValue),
		},
	}
}

//helper func to create a spec with secret
func NewGCSSpecWithSecret(secretNamespace, secretName, keyName string) *v1alpha1.GcsBackupStorage {
	secretKeyRef := influxdatav1alpha1.KubernetesSecret{Name: secretName, Key: keyName, Namespace: secretNamespace}
	vauleFrom := influxdatav1alpha1.ValueFrom{SecretKeyRef: secretKeyRef}
	spec := &influxdatav1alpha1.GcsBackupStorage{
		SaJson: influxdatav1alpha1.SecretRef{ValueFrom: vauleFrom},
		Bucket: "influxdb-backup-restore",
		Folder: "backup",
	}
	return spec
}

//helper func to create a spec with empty secret
func NewGCSSpec() *v1alpha1.GcsBackupStorage {
	spec := &influxdatav1alpha1.GcsBackupStorage{
		SaJson: influxdatav1alpha1.SecretRef{},
		Bucket: "influxdb-backup-restore",
		Folder: "backup",
	}
	return spec
}

//helper to load testing data
func helperLoadBytes(t *testing.T, name string) []byte {
	path := filepath.Join("testdata", name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}

//helper func for creating a mock GCS provider with noResults of body
func NewMockGcsStorageProvider(t *testing.T, body string, noResults int) *GcsStorageProvider {
	mt := &mockTransport{}
	for i := 0; i < noResults; i++ {
		mt.addResult(&http.Response{StatusCode: 200, Body: bodyReader(body)}, nil)
	}
	client := mockClient(t, mt)
	spec := NewGCSSpecWithSecret("default", "influxdb-backup-gcs", "sa")
	bh := &bucket{bh: client.Bucket(spec.Bucket)}
	provider := &GcsStorageProvider{bucketHandle: bh, spec: spec, k8snamespace: "default"}
	return provider
}
