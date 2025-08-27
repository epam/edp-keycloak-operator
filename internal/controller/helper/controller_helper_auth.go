package helper

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

const (
	keycloakTokenSecretPrefix = "kc-token-"
	keycloakTokenSecretKey    = "token"
)

var ErrKeycloakIsNotAvailable = errors.New("keycloak is not available")
var ErrKeycloakRealmNotFound = errors.New("keycloak realm is not available")

// KeycloakAuthData contains data for keycloak authentication.
type KeycloakAuthData struct {
	// Url is keycloak url.
	Url string

	// SecretName is name of secret with keycloak credentials.
	SecretName string

	// SecretNamespace is namespace of secret with keycloak credentials.
	SecretNamespace string

	// AdminType is type of keycloak admin.
	AdminType string

	// KeycloakCRName is name of keycloak CR.
	KeycloakCRName string

	// CACert is root certificate authority.
	CACert string `json:"caCert,omitempty"`

	// InsecureSkipVerify controls whether api client verifies the server's certificate chain and host name.
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}

func (h *Helper) CreateKeycloakClientFromRealmRef(ctx context.Context, object ObjectWithRealmRef) (keycloak.Client, error) {
	authData, err := h.getKeycloakAuthDataFromRealmRef(ctx, object)
	if err != nil {
		return nil, err
	}

	return h.CreateKeycloakClientFomAuthData(ctx, authData)
}

func (h *Helper) CreateKeycloakClientFromRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (keycloak.Client, error) {
	authData, err := h.getKeycloakAuthDataFromRealm(ctx, realm)
	if err != nil {
		return nil, err
	}

	return h.CreateKeycloakClientFomAuthData(ctx, authData)
}

func (h *Helper) CreateKeycloakClientFromClusterRealm(ctx context.Context, realm *keycloakAlpha.ClusterKeycloakRealm) (keycloak.Client, error) {
	authData, err := h.getKeycloakAuthDataFromClusterRealm(ctx, realm)
	if err != nil {
		return nil, err
	}

	return h.CreateKeycloakClientFomAuthData(ctx, authData)
}

func (h *Helper) CreateKeycloakClient(ctx context.Context, url, user, password, adminType, caCert string, insecureSkipVerify bool) (keycloak.Client, error) {
	clientAdapter, err := h.adapterBuilder(
		ctx,
		adapter.GoCloakConfig{
			Url:                url,
			User:               user,
			Password:           password,
			RootCertificate:    caCert,
			InsecureSkipVerify: insecureSkipVerify,
		},
		adminType, ctrl.LoggerFrom(ctx), h.restyClient)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init kc client adapter")
	}

	return clientAdapter, nil
}

func (h *Helper) InvalidateKeycloakClientTokenSecret(ctx context.Context, namespace, rootKeycloakName string) error {
	var secret coreV1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tokenSecretName(rootKeycloakName)},
		&secret); err != nil {
		return errors.Wrap(err, "unable to get client token secret")
	}

	if err := h.client.Delete(ctx, &secret); err != nil {
		return errors.Wrap(err, "unable to delete client token secret")
	}

	return nil
}

func (h *Helper) CreateKeycloakClientFomAuthData(ctx context.Context, authData *KeycloakAuthData) (keycloak.Client, error) {
	h.tokenSecretLock.Lock()
	defer h.tokenSecretLock.Unlock()

	clientAdapter, err := h.createKeycloakClientFromTokenSecret(ctx, authData)
	if err == nil {
		return clientAdapter, nil
	}

	if !k8sErrors.IsNotFound(err) && !adapter.IsErrTokenExpired(err) {
		return nil, fmt.Errorf("unable to create kc client from token secret: %w", err)
	}

	clientAdapter, err = h.createKeycloakClientFromLoginPassword(ctx, authData)
	if err != nil {
		return nil, fmt.Errorf("unable to create kc client from login password: %w", err)
	}

	return clientAdapter, nil
}

func (h *Helper) createKeycloakClientFromLoginPassword(ctx context.Context, authData *KeycloakAuthData) (keycloak.Client, error) {
	var secret coreV1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      authData.SecretName,
		Namespace: authData.SecretNamespace,
	}, &secret); err != nil {
		return nil, errors.Wrap(err, "authData login password secret not found")
	}

	clientAdapter, err := h.CreateKeycloakClient(ctx, authData.Url, string(secret.Data["username"]),
		string(secret.Data["password"]), authData.AdminType, authData.CACert, authData.InsecureSkipVerify)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init authData client adapter")
	}

	jwtToken, err := clientAdapter.ExportToken()
	if err != nil {
		return nil, errors.Wrap(err, "unable to export authData client token")
	}

	if err := h.saveKeycloakClientTokenSecret(ctx, tokenSecretName(authData.SecretName), secret.Namespace, jwtToken); err != nil {
		return nil, errors.Wrap(err, "unable to save authData token to secret")
	}

	return clientAdapter, nil
}

func (h *Helper) createKeycloakClientFromTokenSecret(ctx context.Context, authData *KeycloakAuthData) (keycloak.Client, error) {
	var tokenSecret coreV1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      tokenSecretName(authData.KeycloakCRName),
		Namespace: authData.SecretNamespace,
	}, &tokenSecret); err != nil {
		return nil, errors.Wrap(err, "unable to get token secret")
	}

	clientAdapter, err := adapter.MakeFromToken(adapter.GoCloakConfig{
		Url:                authData.Url,
		RootCertificate:    authData.CACert,
		InsecureSkipVerify: authData.InsecureSkipVerify,
	}, tokenSecret.Data[keycloakTokenSecretKey], ctrl.LoggerFrom(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "unable to make authData client from token")
	}

	return clientAdapter, nil
}

func (h *Helper) saveKeycloakClientTokenSecret(ctx context.Context, secretName, secretNamespace string, token []byte) error {
	var secret coreV1.Secret

	err := h.client.Get(ctx, types.NamespacedName{Namespace: secretNamespace, Name: secretName}, &secret)
	if err == nil {
		secret.Data = map[string][]byte{
			keycloakTokenSecretKey: token,
		}

		if err = h.client.Update(ctx, &secret); err != nil {
			return errors.Wrap(err, "unable to update token secret")
		}

		return nil
	}

	if k8sErrors.IsNotFound(err) {
		secret = coreV1.Secret{ObjectMeta: metav1.ObjectMeta{
			Namespace: secretNamespace,
			Name:      secretName,
		}, Data: map[string][]byte{
			keycloakTokenSecretKey: token,
		}}

		if err = h.client.Create(ctx, &secret); err != nil {
			return errors.Wrap(err, "unable to create token secret")
		}

		return nil
	}

	return errors.Wrap(err, "error during token secret retrieval")
}

func (h *Helper) getKeycloakAuthDataFromRealmRef(ctx context.Context, object ObjectWithRealmRef) (*KeycloakAuthData, error) {
	kind := object.GetRealmRef().Kind
	name := object.GetRealmRef().Name

	switch kind {
	case keycloakApi.KeycloakRealmKind:
		realm := &keycloakApi.KeycloakRealm{}
		if err := h.client.Get(ctx, types.NamespacedName{Name: name, Namespace: object.GetNamespace()}, realm); err != nil {
			if k8sErrors.IsNotFound(err) && object.GetDeletionTimestamp() != nil {
				return nil, ErrKeycloakRealmNotFound
			}

			return nil, fmt.Errorf("unable to get realm: %w", err)
		}

		return h.getKeycloakAuthDataFromRealm(ctx, realm)
	case keycloakAlpha.ClusterKeycloakRealmKind:
		clusterRealm := &keycloakAlpha.ClusterKeycloakRealm{}
		if err := h.client.Get(ctx, types.NamespacedName{Name: name}, clusterRealm); err != nil {
			if k8sErrors.IsNotFound(err) && object.GetDeletionTimestamp() != nil {
				return nil, ErrKeycloakRealmNotFound
			}

			return nil, fmt.Errorf("unable to get cluster realm: %w", err)
		}

		return h.getKeycloakAuthDataFromClusterRealm(ctx, clusterRealm)
	default:
		return nil, fmt.Errorf("unknown realm kind: %s", kind)
	}
}

func (h *Helper) getKeycloakAuthDataFromRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (*KeycloakAuthData, error) {
	kind := realm.Spec.KeycloakRef.Kind
	name := realm.Spec.KeycloakRef.Name

	switch kind {
	case keycloakApi.KeycloakKind:
		kc := &keycloakApi.Keycloak{}
		if err := h.client.Get(ctx, types.NamespacedName{Name: name, Namespace: realm.GetNamespace()}, kc); err != nil {
			return nil, fmt.Errorf("unable to get keycloak: %w", err)
		}

		if !kc.Status.Connected {
			return nil, ErrKeycloakIsNotAvailable
		}

		return MakeKeycloakAuthDataFromKeycloak(ctx, kc, h.client)
	case keycloakAlpha.ClusterKeycloakKind:
		kc := &keycloakAlpha.ClusterKeycloak{}
		if err := h.client.Get(ctx, types.NamespacedName{Name: name}, kc); err != nil {
			return nil, fmt.Errorf("unable to get cluster keycloak: %w", err)
		}

		if !kc.Status.Connected {
			return nil, ErrKeycloakIsNotAvailable
		}

		return MakeKeycloakAuthDataFromClusterKeycloak(ctx, kc, h.operatorNamespace, h.client)
	default:
		return nil, fmt.Errorf("unknown keycloak kind: %s", kind)
	}
}

func (h *Helper) getKeycloakAuthDataFromClusterRealm(ctx context.Context, realm *keycloakAlpha.ClusterKeycloakRealm) (*KeycloakAuthData, error) {
	kc := &keycloakAlpha.ClusterKeycloak{}
	if err := h.client.Get(ctx, types.NamespacedName{Name: realm.GetKeycloakRef().Name}, kc); err != nil {
		return nil, fmt.Errorf("unable to get cluster keycloak: %w", err)
	}

	if !kc.Status.Connected {
		return nil, ErrKeycloakIsNotAvailable
	}

	return MakeKeycloakAuthDataFromClusterKeycloak(ctx, kc, h.operatorNamespace, h.client)
}

func MakeKeycloakAuthDataFromKeycloak(
	ctx context.Context,
	keycloakCR *keycloakApi.Keycloak,
	k8sClient client.Client,
) (*KeycloakAuthData, error) {
	auth := &KeycloakAuthData{
		Url:                keycloakCR.Spec.Url,
		SecretName:         keycloakCR.Spec.Secret,
		SecretNamespace:    keycloakCR.Namespace,
		AdminType:          keycloakCR.Spec.AdminType,
		KeycloakCRName:     keycloakCR.Name,
		InsecureSkipVerify: keycloakCR.Spec.InsecureSkipVerify,
	}

	caCert, err := secretref.GetValueFromSourceRef(ctx, keycloakCR.Spec.CACert, keycloakCR.Namespace, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("unable to get ca cert: %w", err)
	}

	auth.CACert = caCert

	return auth, nil
}

func MakeKeycloakAuthDataFromClusterKeycloak(
	ctx context.Context,
	keycloakCR *keycloakAlpha.ClusterKeycloak,
	secretNamespace string,
	k8sClient client.Client,
) (*KeycloakAuthData, error) {
	auth := &KeycloakAuthData{
		Url:                keycloakCR.Spec.Url,
		SecretName:         keycloakCR.Spec.Secret,
		SecretNamespace:    secretNamespace,
		AdminType:          keycloakCR.Spec.AdminType,
		KeycloakCRName:     keycloakCR.Name,
		InsecureSkipVerify: keycloakCR.Spec.InsecureSkipVerify,
	}

	caCert, err := secretref.GetValueFromSourceRef(ctx, keycloakCR.Spec.CACert, secretNamespace, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("unable to get ca cert: %w", err)
	}

	auth.CACert = caCert

	return auth, nil
}

func tokenSecretName(keycloakName string) string {
	return fmt.Sprintf("%s%s", keycloakTokenSecretPrefix, keycloakName)
}
