package secretref

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretRefPrefix         = "$"
	keycloakSecretRefPrefix = "${"
)

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

// HasAnySecretRef reports whether any value in values is a secret reference.
func HasAnySecretRef(values []string) bool {
	return slices.ContainsFunc(values, HasSecretRef)
}

// GenerateSecretRef generates secret reference.
func GenerateSecretRef(secretName, secretFiled string) string {
	return fmt.Sprintf("%s%s:%s", secretRefPrefix, secretName, secretFiled)
}

// ValuesHash hashes resolved secret values so that rotating the backing k8s Secret (which
// does not bump the CR generation) still forces a write. No secret material is stored, only
// the digest.
func ValuesHash(cfg map[string][]string) string {
	h := sha256.New()

	for _, k := range slices.Sorted(maps.Keys(cfg)) {
		values := cfg[k]

		// Length- and count-prefixed to avoid delimiter ambiguity between adjacent key/value
		// entries. hash.Hash.Write never returns an error.
		_, _ = fmt.Fprintf(h, "%d:%s%d:", len(k), k, len(values))

		for _, v := range values {
			_, _ = fmt.Fprintf(h, "%d:%s", len(v), v)
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}

// ConfigSecretsHash is ValuesHash restricted to keys whose raw config value is a secret ref.
func ConfigSecretsHash(rawCfg, resolvedCfg map[string][]string) string {
	secretBacked := make(map[string][]string)

	for k, raw := range rawCfg {
		if HasAnySecretRef(raw) {
			secretBacked[k] = resolvedCfg[k]
		}
	}

	return ValuesHash(secretBacked)
}

// ConfigSecretsHashSingle is ConfigSecretsHash for single-value config maps: each value is
// wrapped as a one-element slice.
func ConfigSecretsHashSingle(rawCfg, resolvedCfg map[string]string) string {
	wrap := func(config map[string]string) map[string][]string {
		wrapped := make(map[string][]string, len(config))
		for k, v := range config {
			wrapped[k] = []string{v}
		}

		return wrapped
	}

	return ConfigSecretsHash(wrap(rawCfg), wrap(resolvedCfg))
}
