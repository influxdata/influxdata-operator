package storage

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/influxdata-operator/pkg/apis/influxdata/v1alpha1"
	"github.com/influxdata-operator/pkg/remote"
	"github.com/pkg/errors"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"cloud.google.com/go/storage"
)

// type ObjectAttrs interface {
// 	GetAttrs(key string) (*storage.ObjectAttrs, error)
// }

type bucket struct {
	bh *storage.BucketHandle
}

func (b *bucket) GetWriteCloser(key string) io.WriteCloser {
	return b.bh.Object(key).NewWriter(context.Background())
}

func (b *bucket) GetReadCloser(key string) (io.ReadCloser, error) {
	return b.bh.Object(key).NewReader(context.Background())
}

func (b *bucket) GetAttrs(key string) (*storage.ObjectAttrs, error) {
	attrs, err := b.bh.Object(key).Attrs(context.Background())
	// this helps for unit testing using mockClient
	if attrs == nil {
		return &storage.ObjectAttrs{
			Name:        "nil-attrs",
			ContentType: "application/json",
			Size:        int64(0),
		}, nil
	}
	return attrs, err
}

func (b *bucket) List(q *storage.Query) *storage.ObjectIterator {
	return b.bh.Objects(context.Background(), q)
}

// GCS storage provider
type GcsStorageProvider struct {
	spec         *v1alpha1.GcsBackupStorage
	bucketHandle *bucket
	k8snamespace string
}

// a factory for GCS or mock
type StorageClientFactory interface {
	NewStorageClient() (*storage.Client, error)
}

// a factory for GCS
type StorageClientFactoryGCS struct {
	CredentialsFile string
}

// implement GCS factory
func (sc *StorageClientFactoryGCS) NewStorageClient() (*storage.Client, error) {

	// create a StorageClient
	client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(sc.CredentialsFile))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return client, nil
}

const CredentialsFile = "/tmp/sa"

// NewProvider creates a new GCS (compatible) storage provider.
func NewGcsStorageProvider(k8snamespace string, k8sClient client.Client, gcsSpec *v1alpha1.GcsBackupStorage, factory StorageClientFactory) (*GcsStorageProvider, error) {
	saJSON, err := getSaJSON(k8snamespace, k8sClient, gcsSpec)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// write out the gcp service account json string to a file
	f, err := os.Create(CredentialsFile)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer f.Close()
	bytesWritten, err := fmt.Fprintln(f, saJSON)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Printf("%d is written to %s", bytesWritten, CredentialsFile)

	// create a StorageClient
	client, err := factory.NewStorageClient()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &GcsStorageProvider{
		spec:         gcsSpec,
		bucketHandle: &bucket{bh: client.Bucket(gcsSpec.Bucket)},
		k8snamespace: k8snamespace,
	}, nil
}

// Store the given data at the given key.
func (p *GcsStorageProvider) Store(key string, body io.ReadCloser) error {
	log.Printf("Storing file (provider=\"GCS\", bucket=%q, key=%q)\n", p.spec.Bucket, key)
	defer body.Close()

	w := p.bucketHandle.GetWriteCloser(key)

	// The writer returned by NewWriter is asynchronous, so errors aren't guaranteed
	// until Close() is called
	_, copyErr := io.Copy(w, body)

	// Ensure we close w and report errors properly
	closeErr := w.Close()
	if copyErr != nil {
		return copyErr
	}
	return errors.Wrapf(closeErr, "Error storing backup (provider=\"GCS\", bucket=%q, key=%q)", p.spec.Bucket, key)
}

// Retrieve the given key from GCS storage service.
func (p *GcsStorageProvider) Retrieve(key string) (io.ReadCloser, *int64, error) {
	log.Printf("Retrieving backup (provider=\"Gcs\", bucket=%q, key=%q)", p.spec.Bucket, key)
	r, err := p.bucketHandle.GetReadCloser(key)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error1 retrieving backup (provider='GCS', bucket='%s', key='%s')", p.spec.Bucket, key)
	}

	attrs, err := p.bucketHandle.GetAttrs(key)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error2 retrieving backup (provider='GCS', bucket='%s', key='%s')", p.spec.Bucket, key)
	}

	return r, &attrs.Size, nil
}

// Get list of objects in provided the GCS bucket and key
func (p *GcsStorageProvider) ListDirectory(key string) ([]string, error) {
	log.Printf("List Directory (provider='GCS', bucket=%q, key=%q)\n", p.spec.Bucket, key)
	q := &storage.Query{
		Prefix: key,
	}
	var res []string

	iter := p.bucketHandle.List(q)

	for {
		obj, err := iter.Next()
		if err == iterator.Done {
			return res, nil
		}
		if err != nil {
			return nil, errors.WithStack(err)
		}

		res = append(res, obj.Name)
	}
}

// getCredentials gets sa Json value from the provided map.
func getSaJSON(k8snamespace string, k8sClient client.Client, spec *v1alpha1.GcsBackupStorage) (string, error) {
	saJSON := spec.SaJson.ValueFrom.SecretKeyRef
	if saJSON.Namespace == "" {
		saJSON.Namespace = k8snamespace
	}

	saJSONValue, err := getSecretValue(k8sClient, saJSON.Namespace, saJSON.Name, saJSON.Key)

	if err != nil {
		log.Printf("Could not retrieve key id %s/%s/%s", saJSON.Namespace,
			saJSON.Name, saJSON.Key)
		return "", err
	}

	return saJSONValue, nil
}

func (p *GcsStorageProvider) CopyToGCS(gcsSpec *v1alpha1.GcsBackupStorage, backupTime, srcFolder string) (string, error) {
	prefix := gcsSpec.Folder + "/" + backupTime

	files, err := ioutil.ReadDir(srcFolder)
	if err != nil {
		return "", err
	}

	// Loop through files in the source directory, send to GCS
	for _, file := range files {
		localFile := srcFolder + "/" + file.Name()
		f, err := os.Open(localFile)
		if err != nil {
			return "", err
		}

		log.Printf("CopyToGCS: [%s] to [%s]\n", localFile, prefix+"/"+file.Name())
		err = p.Store(prefix+"/"+file.Name(), f)
		if err != nil {
			return "", err
		}
	}

	return "gs://" + gcsSpec.Bucket + "/" + prefix, nil
}

func (p *GcsStorageProvider) CopyFromGCS(remoteFolder, destinationBase string, k8s *remote.K8sClient) error {

	keys, err := p.ListDirectory(remoteFolder)
	if err != nil {
		log.Printf("Error while listing directory: %v\n", err)
		return err
	}
	log.Printf("ListDirectory %v\n", keys)
	for _, key := range keys {
		log.Printf("key=%s,remoteFolder=%s\n", key, remoteFolder)
		// need to strip off prefix, just want name.
		localFileName := strings.TrimPrefix(key, remoteFolder)

		r, size, err := p.Retrieve(key)
		if err != nil {
			log.Printf("Unable to fetch key %s: %v", key, err)
			return err
		}

		// Stream directly to k8s pod
		destination := fmt.Sprintf("%s%s", destinationBase, localFileName)
		log.Printf("localFileName is %s, size is %d, destination %s\n", localFileName, *size, destination)

		err = k8s.CopyToK8s(destination, size, &r)
		if err != nil {
			log.Printf("Error while copying to k8s: %+v\n", err)
			r.Close()
			return err
		}
		r.Close()
	}
	return nil
}
