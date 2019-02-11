package storage

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

	return strings.TrimSuffix(string(value), "\n"), nil
}
