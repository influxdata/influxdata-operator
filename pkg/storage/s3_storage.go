package storage

import (
	"io"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	s3credentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
	"github.com/influxdata-operator/pkg/apis/influxdata/v1alpha1"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type S3StorageProvider struct {
	spec         *v1alpha1.S3BackupStorage
	session      *session.Session
	s3           *s3.S3
	uploader     *s3manager.Uploader
	k8snamespace string
}

// NewProvider creates a new S3 (compatible) storage provider.
func NewS3StorageProvider(k8snamespace string, k8sClient client.Client, s3spec *v1alpha1.S3BackupStorage) (*S3StorageProvider, error) {
	accessKey, secretKey, err := getCredentials(k8snamespace, k8sClient, s3spec)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sess := session.Must(session.NewSession(
		aws.NewConfig().
			WithCredentials(s3credentials.NewStaticCredentials(accessKey, secretKey, "")).
			WithRegion(s3spec.Region).
			WithDisableSSL(true)))

	if err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err := sess.Config.Credentials.Get(); err != nil {
		return nil, errors.WithStack(err)
	}

	return &S3StorageProvider{
		spec:         s3spec,
		session:      sess,
		s3:           s3.New(sess),
		uploader:     s3manager.NewUploader(sess),
		k8snamespace: k8snamespace,
	}, nil
}

// Store the given data at the given key.
func (p *S3StorageProvider) Store(key string, body io.ReadCloser) error {
	glog.V(2).Infof("Storing file (provider=\"S3\", bucket=%q, key=%q)\n", p.spec.Bucket, key)

	defer body.Close()

	_, err := p.uploader.Upload(&s3manager.UploadInput{
		Bucket: &p.spec.Bucket,
		Key:    &key,
		Body:   body,
	})
	return errors.Wrapf(err, "Error storing backup (provider=\"S3\", bucket=%q, key=%q)", p.spec.Bucket, key)
}

// Retrieve the given key from S3 storage service.
func (p *S3StorageProvider) Retrieve(key string) (io.ReadCloser, *int64, error) {
	glog.V(2).Infof("Retrieving backup (provider=\"s3\", bucket=%q, key=%q)", p.spec.Bucket, key)
	obj, err := p.s3.GetObject(&s3.GetObjectInput{Bucket: &p.spec.Bucket, Key: &key})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error retrieving backup (provider='S3', bucket='%s', key='%s')", p.spec.Bucket, key)
	}

	// maybe get file length & other header info here?

	return obj.Body, obj.ContentLength, nil
}

// Get list of objects in provided backup s3 bucket.
func (p *S3StorageProvider) ListDirectory(key string) ([]*string, error) {
	glog.V(2).Infof("Retrieving backup list (provider='S3', bucket=%q, key=%q)\n", p.spec.Bucket, key)
	log.Printf("Retrieving backup list (provider='S3', bucket=%q, key=%q)\n", p.spec.Bucket, key)
	obj, err := p.s3.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: &p.spec.Bucket,
		Delimiter: aws.String("/"), Prefix: aws.String(key + "/")})

	if err != nil {
		return nil, errors.Wrapf(err, "error fetching file list (provider='S3', bucket='%s', key='%s')", p.spec.Bucket, key)
	}

	var results []*string

	for _, item := range obj.Contents {
		results = append(results, item.Key)
	}

	return results, nil
}

// getCredentials gets an accesskey and secretKey from the provided map.
func getCredentials(k8snamespace string, k8sClient client.Client, spec *v1alpha1.S3BackupStorage) (string, string, error) {
	keyid := spec.AwsKeyId.ValueFrom.SecretKeyRef
	if keyid.Namespace == "" {
		keyid.Namespace = k8snamespace
	}

	accessKey, err := getSecretValue(k8sClient, keyid.Namespace, keyid.Name, keyid.Key)

	if err != nil {
		log.Printf("Could not retrieve key id %s/%s/%s", keyid.Namespace,
			keyid.Name, keyid.Key)
		return "", "", err
	}

	secretid := spec.AwsSecretKey.ValueFrom.SecretKeyRef
	if secretid.Namespace == "" {
		secretid.Namespace = k8snamespace
	}

	secretKey, err := getSecretValue(k8sClient, secretid.Namespace, secretid.Name, secretid.Key)
	if err != nil {
		log.Printf("Could not retrieve secret key %s/%s/%s", secretid.Namespace,
			secretid.Name, secretid.Key)
		return "", "", err
	}

	return accessKey, secretKey, nil
}
