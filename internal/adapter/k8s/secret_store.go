package k8s

import (
	"context"

	"karden/internal/domain/workload"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// compile-time check
var _ workload.SecretStore = (*SecretStore)(nil)

type SecretStore struct {
	client kubernetes.Interface
}

func NewSecretStore(client kubernetes.Interface) *SecretStore {
	return &SecretStore{client: client}
}

func (s *SecretStore) Get(ctx context.Context, namespace, name, key string) (string, error) {
	secret, err := s.client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	val, ok := secret.Data[key]
	if !ok {
		return "", nil
	}

	return string(val), nil
}

func (s *SecretStore) Set(ctx context.Context, namespace, name string, data map[string]string) error {
	encoded := make(map[string][]byte, len(data))
	for k, v := range data {
		encoded[k] = []byte(v)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: encoded,
	}

	_, err := s.client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = s.client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	_, err = s.client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	return err
}

func (s *SecretStore) Delete(ctx context.Context, namespace, name string) error {
	return s.client.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (s *SecretStore) GetSecretData(ctx context.Context, namespace, name string) (map[string]string, error) {
	secret, err := s.client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	data := make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		data[k] = string(v)
	}
	return data, nil
}
