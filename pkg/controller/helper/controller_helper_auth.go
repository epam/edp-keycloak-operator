package helper

import (
	"context"
	"fmt"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	keycloakTokenSecretPrefix = "kc-token-"
	keycloakTokenSecretKey    = "token"
)

func (h *Helper) CreateKeycloakClientForRealm(ctx context.Context,
	realm *v1alpha1.KeycloakRealm) (keycloak.Client, error) {

	kc, err := h.GetOrCreateKeycloakOwnerRef(realm)
	if err != nil {
		return nil, err
	}

	if !kc.Status.Connected {
		return nil, errors.New("Owner keycloak is not in connected status")
	}

	clientAdapter, err := h.CreateKeycloakClientFromTokenSecret(ctx, kc)
	if err == nil {
		return clientAdapter, nil
	}

	if !k8sErrors.IsNotFound(err) && !adapter.IsErrTokenExpired(err) {
		return nil, err
	}

	clientAdapter, err = h.CreateKeycloakClientFromLoginPassword(ctx, kc)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create kc client from login password")
	}

	return clientAdapter, nil
}

func (h *Helper) CreateKeycloakClientFromLoginPassword(ctx context.Context, kc *v1alpha1.Keycloak) (keycloak.Client, error) {
	var secret coreV1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      kc.Spec.Secret,
		Namespace: kc.Namespace,
	}, &secret); err != nil {
		return nil, errors.Wrap(err, "kc login password secret not found")
	}

	clientAdapter, err := h.CreateKeycloakClient(ctx, kc.Spec.Url, string(secret.Data["username"]),
		string(secret.Data["password"]))
	if err != nil {
		return nil, errors.Wrap(err, "unable to init kc client adapter")
	}

	jwtToken, err := clientAdapter.ExportToken()
	if err != nil {
		return nil, errors.Wrap(err, "unable to export kc client token")
	}

	if err := h.SaveKeycloakClientTokenSecret(ctx, kc, jwtToken); err != nil {
		return nil, errors.Wrap(err, "unable to save kc token to secret")
	}

	return clientAdapter, nil
}

func (h *Helper) CreateKeycloakClient(ctx context.Context, url, user, password string) (keycloak.Client, error) {
	clientAdapter, err := adapter.Make(ctx, url, user, password, h.logger, h.restyClient)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init kc client adapter")
	}

	return clientAdapter, nil
}

func (h *Helper) SaveKeycloakClientTokenSecret(ctx context.Context, kc *v1alpha1.Keycloak, token []byte) error {
	var secret coreV1.Secret
	err := h.client.Get(ctx, types.NamespacedName{Namespace: kc.Namespace, Name: tokenSecretName(kc.Name)}, &secret)
	if err == nil {
		secret.Data = map[string][]byte{
			keycloakTokenSecretKey: token,
		}

		if err := h.client.Update(ctx, &secret); err != nil {
			return errors.Wrap(err, "unable to update token secret")
		}

		return nil
	}

	if k8sErrors.IsNotFound(err) {
		secret = coreV1.Secret{ObjectMeta: metav1.ObjectMeta{
			Namespace: kc.Namespace,
			Name:      tokenSecretName(kc.Name),
		}, Data: map[string][]byte{
			keycloakTokenSecretKey: token,
		}}

		if err := h.client.Create(ctx, &secret); err != nil {
			return errors.Wrap(err, "unable to create token secret")
		}

		return nil
	}

	return errors.Wrap(err, "error during token secret retrieval")
}

func (h *Helper) CreateKeycloakClientFromTokenSecret(ctx context.Context, kc *v1alpha1.Keycloak) (keycloak.Client, error) {
	var tokenSecret coreV1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      tokenSecretName(kc.Name),
		Namespace: kc.Namespace,
	}, &tokenSecret); err != nil {
		return nil, errors.Wrap(err, "unable to get token secret")
	}

	clientAdapter, err := adapter.MakeFromToken(kc.Spec.Url, tokenSecret.Data[keycloakTokenSecretKey], h.logger)
	if err != nil {
		return nil, errors.Wrap(err, "unable to make kc client from token")
	}

	return clientAdapter, nil
}

func tokenSecretName(keycloakName string) string {
	return fmt.Sprintf("%s%s", keycloakTokenSecretPrefix, keycloakName)
}
