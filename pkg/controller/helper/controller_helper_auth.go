package helper

import (
	"context"
	"fmt"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	keycloakTokenSecretPrefix = "kc-token-"
	keycloakTokenSecretKey    = "token"
)

func (h *Helper) CreateKeycloakClientForRealm(ctx context.Context,
	realm *v1alpha1.KeycloakRealm, log logr.Logger) (keycloak.Client, error) {

	o, err := h.GetOrCreateKeycloakOwnerRef(realm)
	if err != nil {
		return nil, err
	}

	if !o.Status.Connected {
		return nil, errors.New("Owner keycloak is not in connected status")
	}

	clientAdapter, err := h.CreateKeycloakClientFromTokenSecret(ctx, o, log)
	if err == nil {
		return clientAdapter, nil
	}

	if !k8sErrors.IsNotFound(err) && !adapter.IsErrTokenExpired(err) {
		return nil, err
	}

	var secret coreV1.Secret
	if err = h.client.Get(ctx, types.NamespacedName{
		Name:      o.Spec.Secret,
		Namespace: o.Namespace,
	}, &secret); err != nil {
		return nil, err
	}

	clientAdapter, err = h.CreateKeycloakClient(o.Spec.Url, string(secret.Data["username"]), string(secret.Data["password"]),
		log)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init kc client adapter")
	}

	return clientAdapter, nil
}

func (h *Helper) CreateKeycloakClient(url, user, password string, log logr.Logger) (keycloak.Client, error) {
	clientAdapter, err := adapter.Make(url, user, password, log)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init kc client adapter")
	}

	return clientAdapter, nil
}

func (h *Helper) CreateKeycloakClientFromTokenSecret(ctx context.Context, kc *v1alpha1.Keycloak,
	log logr.Logger) (keycloak.Client, error) {
	var tokenSecret coreV1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s%s", keycloakTokenSecretPrefix, kc.Name),
		Namespace: kc.Namespace,
	}, &tokenSecret); err != nil {
		return nil, errors.Wrap(err, "unable to get token secret")
	}

	clientAdapter, err := adapter.MakeFromToken(kc.Spec.Url, tokenSecret.Data[keycloakTokenSecretKey], log)
	if err != nil {
		return nil, errors.Wrap(err, "unable to make kc client from token")
	}

	return clientAdapter, nil
}
