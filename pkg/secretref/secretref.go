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
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretRefPrefix         = "$"
	keycloakSecretRefPrefix = "${"
)

type RefClient interface {
	MapConfigSecretsRefs(ctx context.Context, config map[string]string, namespace string) (map[string]string, error)
	MapComponentConfigSecretsRefs(
		ctx context.Context,
		config map[string][]string,
		namespace string,
	) (map[string][]string, error)
	GetSecretFromRef(ctx context.Context, refVal, secretNamespace string) (value, version string, err error)
}

// SecretRef provides methods to work with secret references.
type SecretRef struct {
	client client.Client
}

// NewSecretRef returns a new instance of SecretRef.
func NewSecretRef(k8sClient client.Client) *SecretRef {
	return &SecretRef{client: k8sClient}
}

// MapConfigSecretsRefs maps secret references in config map to actual values. It returns a
// version token per ref-backed key for status.configSecretsHash.
func (s *SecretRef) MapConfigSecretsRefs(
	ctx context.Context,
	config map[string]string,
	namespace string,
) (map[string]string, error) {
	versions := make(map[string]string)

	for k, v := range config {
		if !HasSecretRef(v) {
			continue
		}

		secretVal, version, err := s.GetSecretFromRef(ctx, v, namespace)
		if err != nil {
			return nil, err
		}

		config[k] = secretVal
		versions[k] = version
	}

	return versions, nil
}

// MapComponentConfigSecretsRefs maps secret references in config map to actual values. It
// returns version tokens per ref-backed key (ref-valued entries only, in spec order).
func (s *SecretRef) MapComponentConfigSecretsRefs(
	ctx context.Context,
	config map[string][]string,
	namespace string,
) (map[string][]string, error) {
	versions := make(map[string][]string)

	for k, values := range config {
		for i, v := range values {
			if !HasSecretRef(v) {
				continue
			}

			secretVal, version, err := s.GetSecretFromRef(ctx, v, namespace)
			if err != nil {
				return nil, err
			}

			config[k][i] = secretVal

			versions[k] = append(versions[k], version)
		}
	}

	return versions, nil
}

// GetSecretFromRef returns the secret value and its version token from a secret reference.
// Value and version come from the same object read so a concurrent rotation cannot produce
// a token newer than the value.
func (s *SecretRef) GetSecretFromRef(ctx context.Context, refVal, secretNamespace string) (string, string, error) {
	if !HasSecretRef(refVal) {
		return "", "", fmt.Errorf("invalid config secret reference %s is not in format '$secretName:secretKey'", refVal)
	}

	// Skip keycloak references format. This mapping is managed by the Keycloak service.
	if strings.HasPrefix(refVal, keycloakSecretRefPrefix) {
		return refVal, refVal, nil
	}

	ref := strings.Split(refVal[1:], ":")
	if len(ref) != 2 {
		return "", "", fmt.Errorf("invalid config secret  reference %s is not in format '$secretName:secretKey'", refVal)
	}

	secret := &corev1.Secret{}
	if err := s.client.Get(ctx, client.ObjectKey{
		Namespace: secretNamespace,
		Name:      ref[0],
	}, secret); err != nil {
		return "", "", fmt.Errorf("failed to get secret %s: %w", ref[0], err)
	}

	secretVal, ok := secret.Data[ref[1]]
	if !ok {
		return "", "", fmt.Errorf("secret %s does not contain key %s", ref[0], ref[1])
	}

	return string(secretVal), SecretKeyVersion(secret, ref[1]), nil
}

// versionToken formats "<kind>:<name>:<key>@<uid>@<resourceVersion>". K8s object names and
// data keys cannot contain ':' or '@', so the framing is unambiguous.
func versionToken(kind, name, key string, uid k8stypes.UID, resourceVersion string) string {
	return fmt.Sprintf("%s:%s:%s@%s@%s", kind, name, key, uid, resourceVersion)
}

// SecretKeyVersion is the version token of one secret key. It carries no secret data;
// rotating the Secret changes resourceVersion and therefore the token.
func SecretKeyVersion(secret *corev1.Secret, key string) string {
	return versionToken("secret", secret.Name, key, secret.UID, secret.ResourceVersion)
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

// ValuesHash hashes secret version tokens so that rotating the backing k8s Secret (which
// does not bump the CR generation) still forces a write. Input is object metadata
// (name/key/uid/resourceVersion), never secret material, so the digest cannot be tested
// against guessed secret values.
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

// ValuesHashSingle is ValuesHash for single-value maps: each value is wrapped as a
// one-element slice.
func ValuesHashSingle(cfg map[string]string) string {
	wrapped := make(map[string][]string, len(cfg))
	for k, v := range cfg {
		wrapped[k] = []string{v}
	}

	return ValuesHash(wrapped)
}
