package storage

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	s3credentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
	"github.com/influxdata-operator/pkg/apis/influxdata/v1alpha1"
	"github.com/pkg/errors"
	"io"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type S3StorageProvider struct {
	spec *v1alpha1.S3BackupStorage
	session *session.Session
	s3 *s3.S3
	uploader *s3manager.Uploader
	k8snamespace string
}

// NewProvider creates a new S3 (compatible) storage provider.
func NewS3StorageProvider(k8snamespace string, k8sClient client.Client, s3spec *v1alpha1.S3BackupStorage) (*S3StorageProvider, error) {
	log.Println("Creating S3 Storage Provider")

	accessKey, secretKey, err := getCredentials(k8snamespace, k8sClient, s3spec)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sess := session.Must(session.NewSession(
		aws.NewConfig().
			WithCredentials(s3credentials.NewStaticCredentials(accessKey, secretKey, "")).
			WithRegion(s3spec.Region)))

	if err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err := sess.Config.Credentials.Get(); err != nil {
		return nil, errors.WithStack(err)
	}

	return &S3StorageProvider{
		spec:           s3spec,
		session:        sess,
		s3:             s3.New(sess),
		uploader:       s3manager.NewUploader(sess),
		k8snamespace:   k8snamespace,
	}, nil
}

// Store the given data at the given key.
func (p *S3StorageProvider) Store(key string, body io.ReadCloser) error {
	glog.V(2).Infof("Storing file (provider=\"S3\", bucket=%q, key=%q)", p.spec.Bucket, key)

	defer body.Close()

	_, err := p.uploader.Upload(&s3manager.UploadInput{
		Bucket: &p.spec.Bucket,
		Key:    &key,
		Body:   body,
	})
	return errors.Wrapf(err, "Error storing backup (provider=\"S3\", bucket=%q, key=%q)", p.spec.Bucket, key)
}

// Retrieve the given key from S3 storage service.
func (p *S3StorageProvider) Retrieve(key string) (io.ReadCloser, error) {
	glog.V(2).Infof("Retrieving backup (provider=\"s3\", endpoint=%q, bucket=%q, key=%q)", p.spec.Bucket, key)

	obj, err := p.s3.GetObject(&s3.GetObjectInput{Bucket: &p.spec.Bucket, Key: &key})
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving backup (provider='S3', endpoint='%s', bucket='%s', key='%s')", p.spec.Bucket, key)
	}

	return obj.Body, nil
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

func getSecretValue(k8sClient client.Client, namespace string, secretName string, secretKey string) (string, error) {
	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      secretName,
		},
	}

	secretLocation := types.NamespacedName{namespace, secretName}

	err := k8sClient.Get(context.TODO(), secretLocation, &secret)
	if err != nil {
		return "", err
	}

	value, ok := secret.Data[secretKey]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %q in namespace %q", secretKey, secret.ObjectMeta.Name, secret.ObjectMeta.Namespace)
	}

	decoded, err := base64.StdEncoding.DecodeString(string(value))
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}
