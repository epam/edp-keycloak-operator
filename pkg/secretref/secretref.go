package secretref

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretRefPrefix         = "$"
	keycloakSecretRefPrefix = "${"
)

//go:generate mockery --name RefClient --filename ref_mock.go
type RefClient interface {
	MapConfigSecretsRefs(ctx context.Context, config map[string]string, namespace string) error
	MapComponentConfigSecretsRefs(ctx context.Context, config map[string][]string, namespace string) error
	GetSecretFromRef(ctx context.Context, refVal, secretNamespace string) (string, error)
}

// SecretRef provides methods to work with secret references.
type SecretRef struct {
	client client.Client
}

// NewSecretRef returns a new instance of SecretRef.
func NewSecretRef(k8sClient client.Client) *SecretRef {
	return &SecretRef{client: k8sClient}
}

// MapConfigSecretsRefs maps secret references in config map to actual values.
func (s *SecretRef) MapConfigSecretsRefs(ctx context.Context, config map[string]string, namespace string) error {
	for k, v := range config {
		if !HasSecretRef(v) {
			continue
		}

		secretVal, err := s.GetSecretFromRef(ctx, v, namespace)
		if err != nil {
			return err
		}

		config[k] = secretVal
	}

	return nil
}

// MapConfigSecretsRefs maps secret references in config map to actual values.
func (s *SecretRef) MapComponentConfigSecretsRefs(
	ctx context.Context,
	config map[string][]string,
	namespace string,
) error {
	for k, values := range config {
		for i, v := range values {
			if !HasSecretRef(v) {
				continue
			}

			secretVal, err := s.GetSecretFromRef(ctx, v, namespace)
			if err != nil {
				return err
			}

			config[k][i] = secretVal
		}
	}

	return nil
}

// GetSecretFromRef returns secret value from secret reference.
func (s *SecretRef) GetSecretFromRef(ctx context.Context, refVal, secretNamespace string) (string, error) {
	if !HasSecretRef(refVal) {
		return "", fmt.Errorf("invalid config secret reference %s is not in format '$secretName:secretKey'", refVal)
	}

	// Skip keycloak references format. This mapping is managed by the Keycloak service.
	if strings.HasPrefix(refVal, keycloakSecretRefPrefix) {
		return refVal, nil
	}

	ref := strings.Split(refVal[1:], ":")
	if len(ref) != 2 {
		return "", fmt.Errorf("invalid config secret  reference %s is not in format '$secretName:secretKey'", refVal)
	}

	secret := &corev1.Secret{}
	if err := s.client.Get(ctx, client.ObjectKey{
		Namespace: secretNamespace,
		Name:      ref[0],
	}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", ref[0], err)
	}

	secretVal, ok := secret.Data[ref[1]]
	if !ok {
		return "", fmt.Errorf("secret %s does not contain key %s", ref[0], ref[1])
	}

	return string(secretVal), nil
}

// HasSecretRef checks if value has secret reference.
func HasSecretRef(val string) bool {
	return strings.HasPrefix(val, secretRefPrefix)
}

// GenerateSecretRef generates secret reference.
func GenerateSecretRef(secretName, secretFiled string) string {
	return fmt.Sprintf("%s%s:%s", secretRefPrefix, secretName, secretFiled)
}
