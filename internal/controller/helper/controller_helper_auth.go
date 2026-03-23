package helper

import (
	"context"
	"errors"
	"fmt"

	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	keycloakclientv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
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

	// AuthSpec is the new auth configuration. When set, takes precedence over SecretName/AdminType.
	AuthSpec *common.AuthSpec
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

func (h *Helper) CreateKeycloakClientV2FromRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (*keycloakclientv2.KeycloakClient, error) {
	authData, err := h.getKeycloakAuthDataFromRealm(ctx, realm)
	if err != nil {
		return nil, err
	}

	return h.createKeycloakClientV2FromAuthData(ctx, authData)
}

func (h *Helper) CreateKeycloakClientV2FromKeycloak(ctx context.Context, kc *keycloakApi.Keycloak) (*keycloakclientv2.KeycloakClient, error) {
	authData, err := MakeKeycloakAuthDataFromKeycloak(ctx, kc, h.client)
	if err != nil {
		return nil, err
	}

	return h.createKeycloakClientV2FromAuthData(ctx, authData)
}

func (h *Helper) CreateKeycloakClientV2FromClusterKeycloak(ctx context.Context, kc *keycloakAlpha.ClusterKeycloak) (*keycloakclientv2.KeycloakClient, error) {
	authData, err := MakeKeycloakAuthDataFromClusterKeycloak(ctx, kc, h.operatorNamespace, h.client)
	if err != nil {
		return nil, err
	}

	return h.createKeycloakClientV2FromAuthData(ctx, authData)
}

func (h *Helper) CreateKeycloakClientV2FromRealmRef(ctx context.Context, object ObjectWithRealmRef) (*keycloakclientv2.KeycloakClient, error) {
	authData, err := h.getKeycloakAuthDataFromRealmRef(ctx, object)
	if err != nil {
		return nil, err
	}

	return h.createKeycloakClientV2FromAuthData(ctx, authData)
}

func (h *Helper) CreateKeycloakClientV2FromClusterRealm(ctx context.Context, realm *keycloakAlpha.ClusterKeycloakRealm) (*keycloakclientv2.KeycloakClient, error) {
	authData, err := h.getKeycloakAuthDataFromClusterRealm(ctx, realm)
	if err != nil {
		return nil, err
	}

	return h.createKeycloakClientV2FromAuthData(ctx, authData)
}

func (h *Helper) createKeycloakClientV2FromAuthData(ctx context.Context, authData *KeycloakAuthData) (*keycloakclientv2.KeycloakClient, error) {
	var options []keycloakclientv2.ClientOption

	clientID := keycloakclientv2.DefaultAdminClientID

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

			options = append(options, keycloakclientv2.WithClientSecret(password))
		} else {
			options = append(options, keycloakclientv2.WithPasswordGrant(username, password))
		}
	}

	if authData.CACert != "" {
		options = append(options, keycloakclientv2.WithCACert(authData.CACert))
	}

	if authData.InsecureSkipVerify {
		options = append(options, keycloakclientv2.WithTLSInsecureSkipVerify(true))
	}

	kcClient, err := keycloakclientv2.NewKeycloakClient(ctx, authData.Url, clientID, options...)
	if err != nil {
		return nil, fmt.Errorf("unable to create keycloak v2 client: %w", err)
	}

	return kcClient, nil
}

func (h *Helper) buildV2AuthOptions(
	ctx context.Context,
	authData *KeycloakAuthData,
) (clientID string, options []keycloakclientv2.ClientOption, err error) {
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

		return keycloakclientv2.DefaultAdminClientID,
			[]keycloakclientv2.ClientOption{keycloakclientv2.WithPasswordGrant(username, password)},
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
			[]keycloakclientv2.ClientOption{keycloakclientv2.WithClientSecret(clientSecret)},
			nil

	default:
		return "", nil, errors.New("one of passwordGrant or clientCredentials must be set")
	}
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
		return nil, fmt.Errorf("unable to init kc client adapter: %w", err)
	}

	return clientAdapter, nil
}

func (h *Helper) InvalidateKeycloakClientTokenSecret(ctx context.Context, namespace, rootKeycloakName string) error {
	var secret coreV1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tokenSecretName(rootKeycloakName)},
		&secret); err != nil {
		return fmt.Errorf("unable to get client token secret: %w", err)
	}

	if err := h.client.Delete(ctx, &secret); err != nil {
		return fmt.Errorf("unable to delete client token secret: %w", err)
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

func (h *Helper) createKeycloakClientFromLoginPassword(ctx context.Context, authData *KeycloakAuthData) (keycloak.Client, error) {
	var (
		username, password, adminType string
		err                           error
	)

	if authData.AuthSpec != nil {
		username, password, adminType, err = h.resolveLegacyAuthFromSpec(ctx, authData)
		if err != nil {
			return nil, err
		}
	} else {
		username, password, err = h.getCredentialsFromSecret(ctx, authData.SecretName, authData.SecretNamespace)
		if err != nil {
			return nil, fmt.Errorf("unable to get credentials: %w", err)
		}

		adminType = authData.AdminType
	}

	clientAdapter, err := h.CreateKeycloakClient(ctx, authData.Url, username,
		password, adminType, authData.CACert, authData.InsecureSkipVerify)
	if err != nil {
		return nil, fmt.Errorf("unable to init authData client adapter: %w", err)
	}

	jwtToken, err := clientAdapter.ExportToken()
	if err != nil {
		return nil, fmt.Errorf("unable to export authData client token: %w", err)
	}

	if err := h.saveKeycloakClientTokenSecret(ctx, tokenSecretName(authData.KeycloakCRName), authData.SecretNamespace, jwtToken); err != nil {
		return nil, fmt.Errorf("unable to save authData token to secret: %w", err)
	}

	return clientAdapter, nil
}

func (h *Helper) resolveLegacyAuthFromSpec(ctx context.Context, authData *KeycloakAuthData) (username, password, adminType string, err error) {
	switch {
	case authData.AuthSpec.PasswordGrant != nil:
		username, err = secretref.GetValueFromSourceRefOrVal(
			ctx, &authData.AuthSpec.PasswordGrant.Username, authData.SecretNamespace, h.client,
		)
		if err != nil {
			return "", "", "", fmt.Errorf("unable to resolve username: %w", err)
		}

		password, err = secretref.GetValueFromSecretKeySelector(
			ctx, &authData.AuthSpec.PasswordGrant.PasswordRef, authData.SecretNamespace, h.client,
		)
		if err != nil {
			return "", "", "", fmt.Errorf("unable to resolve password: %w", err)
		}

		return username, password, keycloakApi.KeycloakAdminTypeUser, nil

	case authData.AuthSpec.ClientCredentials != nil:
		clientID, err := secretref.GetValueFromSourceRefOrVal(
			ctx, &authData.AuthSpec.ClientCredentials.ClientID, authData.SecretNamespace, h.client,
		)
		if err != nil {
			return "", "", "", fmt.Errorf("unable to resolve client id: %w", err)
		}

		clientSecret, err := secretref.GetValueFromSecretKeySelector(
			ctx, &authData.AuthSpec.ClientCredentials.ClientSecretRef, authData.SecretNamespace, h.client,
		)
		if err != nil {
			return "", "", "", fmt.Errorf("unable to resolve client secret: %w", err)
		}

		return clientID, clientSecret, keycloakApi.KeycloakAdminTypeServiceAccount, nil

	default:
		return "", "", "", errors.New("one of passwordGrant or clientCredentials must be set")
	}
}

func (h *Helper) createKeycloakClientFromTokenSecret(ctx context.Context, authData *KeycloakAuthData) (keycloak.Client, error) {
	var tokenSecret coreV1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      tokenSecretName(authData.KeycloakCRName),
		Namespace: authData.SecretNamespace,
	}, &tokenSecret); err != nil {
		return nil, fmt.Errorf("unable to get token secret: %w", err)
	}

	clientAdapter, err := adapter.MakeFromToken(adapter.GoCloakConfig{
		Url:                authData.Url,
		RootCertificate:    authData.CACert,
		InsecureSkipVerify: authData.InsecureSkipVerify,
	}, tokenSecret.Data[keycloakTokenSecretKey], ctrl.LoggerFrom(ctx))
	if err != nil {
		return nil, fmt.Errorf("unable to make authData client from token: %w", err)
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
			return fmt.Errorf("unable to update token secret: %w", err)
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
			return fmt.Errorf("unable to create token secret: %w", err)
		}

		return nil
	}

	return fmt.Errorf("error during token secret retrieval: %w", err)
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

func tokenSecretName(keycloakName string) string {
	return fmt.Sprintf("%s%s", keycloakTokenSecretPrefix, keycloakName)
}
