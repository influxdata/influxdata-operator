package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/influxdata-operator/pkg/apis/influxdata/v1alpha1"
	"github.com/pkg/errors"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"cloud.google.com/go/storage"
)

type bucket struct {
	bh *storage.BucketHandle
}

func (b *bucket) getWriteCloser(key string) io.WriteCloser {
	return b.bh.Object(key).NewWriter(context.Background())
}

func (b *bucket) getReadCloser(key string) (io.ReadCloser, error) {
	return b.bh.Object(key).NewReader(context.Background())
}

func (b *bucket) getAttrs(key string) (*storage.ObjectAttrs, error) {
	return b.bh.Object(key).Attrs(context.Background())
}

type GcsStorageProvider struct {
	spec         *v1alpha1.GcsBackupStorage
	bucketHandle *bucket
	k8snamespace string
}

const credentialsFile = "/tmp/sa"

// NewProvider creates a new GCS (compatible) storage provider.
func NewGcsStorageProvider(k8snamespace string, k8sClient client.Client, gcsSpec *v1alpha1.GcsBackupStorage) (*GcsStorageProvider, error) {
	saJSON, err := getSaJSON(k8snamespace, k8sClient, gcsSpec)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// write out the gcp service account json string to a file
	f, err := os.Create(credentialsFile)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer f.Close()
	bytesWritten, err := fmt.Fprintln(f, saJSON)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Printf("%d is written to %s", bytesWritten, credentialsFile)

	// create a StorageClient
	client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(credentialsFile))
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

	w := p.bucketHandle.getWriteCloser(key)

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
	r, err := p.bucketHandle.getReadCloser(key)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error retrieving backup (provider='GCS', bucket='%s', key='%s')", p.spec.Bucket, key)
	}

	attrs, err := p.bucketHandle.getAttrs(key)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error retrieving backup (provider='GCS', bucket='%s', key='%s')", p.spec.Bucket, key)
	}

	return r, &attrs.Size, nil
}

// Get list of objects in provided the GCS bucket and key
func (p *GcsStorageProvider) ListDirectory(key string) ([]string, error) {
	log.Printf("Retrieving backup list (provider='GCS', bucket=%q, key=%q)\n", p.spec.Bucket, key)
	q := &storage.Query{
		Prefix: key,
	}
	var res []string

	iter := p.bucketHandle.bh.Objects(context.Background(), q)

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
