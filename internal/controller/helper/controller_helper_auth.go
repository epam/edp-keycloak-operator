package helper

import (
	"context"
	"errors"
	"fmt"

	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakClient "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
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
	CACert string

	// InsecureSkipVerify controls whether api client verifies the server's certificate chain and host name.
	InsecureSkipVerify bool

	// AuthSpec is the new auth configuration. When set, takes precedence over SecretName/AdminType.
	AuthSpec *common.AuthSpec
}

func (h *Helper) CreateKeycloakClientFromRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (*keycloakClient.KeycloakClient, error) {
	authData, err := h.getKeycloakAuthDataFromRealm(ctx, realm)
	if err != nil {
		return nil, err
	}

	return h.createKeycloakClientFromAuthData(ctx, authData)
}

func (h *Helper) CreateKeycloakClientFromKeycloak(ctx context.Context, kc *keycloakApi.Keycloak) (*keycloakClient.KeycloakClient, error) {
	authData, err := MakeKeycloakAuthDataFromKeycloak(ctx, kc, h.client)
	if err != nil {
		return nil, err
	}

	return h.createKeycloakClientFromAuthData(ctx, authData)
}

func (h *Helper) CreateKeycloakClientFromClusterKeycloak(ctx context.Context, kc *keycloakAlpha.ClusterKeycloak) (*keycloakClient.KeycloakClient, error) {
	authData, err := MakeKeycloakAuthDataFromClusterKeycloak(ctx, kc, h.operatorNamespace, h.client)
	if err != nil {
		return nil, err
	}

	return h.createKeycloakClientFromAuthData(ctx, authData)
}

func (h *Helper) CreateKeycloakClientFromRealmRef(ctx context.Context, object ObjectWithRealmRef) (*keycloakClient.KeycloakClient, error) {
	authData, err := h.getKeycloakAuthDataFromRealmRef(ctx, object)
	if err != nil {
		return nil, err
	}

	return h.createKeycloakClientFromAuthData(ctx, authData)
}

func (h *Helper) CreateKeycloakClientFromClusterRealm(ctx context.Context, realm *keycloakAlpha.ClusterKeycloakRealm) (*keycloakClient.KeycloakClient, error) {
	authData, err := h.getKeycloakAuthDataFromClusterRealm(ctx, realm)
	if err != nil {
		return nil, err
	}

	return h.createKeycloakClientFromAuthData(ctx, authData)
}

func (h *Helper) createKeycloakClientFromAuthData(ctx context.Context, authData *KeycloakAuthData) (*keycloakClient.KeycloakClient, error) {
	var options []keycloakClient.ClientOption

	clientID := keycloakClient.DefaultAdminClientID

	if authData.AuthSpec != nil {
		var err error

		clientID, options, err = h.buildV2AuthOptions(ctx, authData)
		if err != nil {
			return nil, err
		}
	} else {
		username, password, err := h.getCredentialsFromSecret(ctx, authData.SecretName, authData.SecretNamespace)
		if err != nil {
			return nil, fmt.Errorf("unable to get credentials: %w", err)
		}

		if authData.AdminType == keycloakApi.KeycloakAdminTypeServiceAccount {
			clientID = username

			options = append(options, keycloakClient.WithClientSecret(password))
		} else {
			options = append(options, keycloakClient.WithPasswordGrant(username, password))
		}
	}

	if authData.CACert != "" {
		options = append(options, keycloakClient.WithCACert(authData.CACert))
	}

	if authData.InsecureSkipVerify {
		options = append(options, keycloakClient.WithTLSInsecureSkipVerify(true))
	}

	kcClient, err := keycloakClient.NewKeycloakClient(ctx, authData.Url, clientID, options...)
	if err != nil {
		return nil, fmt.Errorf("unable to create keycloak v2 client: %w", err)
	}

	return kcClient, nil
}

func (h *Helper) buildV2AuthOptions(
	ctx context.Context,
	authData *KeycloakAuthData,
) (clientID string, options []keycloakClient.ClientOption, err error) {
	switch {
	case authData.AuthSpec.PasswordGrant != nil:
		username, err := secretref.GetValueFromSourceRefOrVal(
			ctx, &authData.AuthSpec.PasswordGrant.Username, authData.SecretNamespace, h.client,
		)
		if err != nil {
			return "", nil, fmt.Errorf("unable to resolve username: %w", err)
		}

		password, err := secretref.GetValueFromSecretKeySelector(
			ctx, &authData.AuthSpec.PasswordGrant.PasswordRef, authData.SecretNamespace, h.client,
		)
		if err != nil {
			return "", nil, fmt.Errorf("unable to resolve password: %w", err)
		}

		return keycloakClient.DefaultAdminClientID,
			[]keycloakClient.ClientOption{keycloakClient.WithPasswordGrant(username, password)},
			nil

	case authData.AuthSpec.ClientCredentials != nil:
		clientID, err := secretref.GetValueFromSourceRefOrVal(
			ctx, &authData.AuthSpec.ClientCredentials.ClientID, authData.SecretNamespace, h.client,
		)
		if err != nil {
			return "", nil, fmt.Errorf("unable to resolve client id: %w", err)
		}

		clientSecret, err := secretref.GetValueFromSecretKeySelector(
			ctx, &authData.AuthSpec.ClientCredentials.ClientSecretRef, authData.SecretNamespace, h.client,
		)
		if err != nil {
			return "", nil, fmt.Errorf("unable to resolve client secret: %w", err)
		}

		return clientID,
			[]keycloakClient.ClientOption{keycloakClient.WithClientSecret(clientSecret)},
			nil

	default:
		return "", nil, errors.New("one of passwordGrant or clientCredentials must be set")
	}
}

func (h *Helper) getCredentialsFromSecret(ctx context.Context, secretName, secretNamespace string) (username, password string, err error) {
	var secret coreV1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: secretNamespace,
	}, &secret); err != nil {
		return "", "", fmt.Errorf("secret not found: %w", err)
	}

	return string(secret.Data["username"]), string(secret.Data["password"]), nil
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
		AuthSpec:           keycloakCR.Spec.Auth,
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
		AuthSpec:           keycloakCR.Spec.Auth,
	}

	caCert, err := secretref.GetValueFromSourceRef(ctx, keycloakCR.Spec.CACert, secretNamespace, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("unable to get ca cert: %w", err)
	}

	auth.CACert = caCert

	return auth, nil
}
