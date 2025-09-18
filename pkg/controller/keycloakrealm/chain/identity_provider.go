package chain

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
)

type PutIdentityProvider struct {
	next   handler.RealmHandler
	client client.Client
}

func (h PutIdentityProvider) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Name, "realm namespace", realm.Namespace)
	rLog.Info("Start put identity provider for realm...")

	rDto := dto.ConvertSpecToRealm(realm.Spec)
	if !rDto.SsoRealmEnabled {
		rLog.Info("sso realm disabled, skip put identity provider step")
		return nextServeOrNil(ctx, h.next, realm, kClient)
	}

	if err := h.setupIdentityProvider(ctx, realm, kClient, rLog, rDto); err != nil {
		return errors.Wrap(err, "unable to setup identity provider")
	}

	if realm.Spec.SSORealmMappers != nil {
		if err := kClient.SyncRealmIdentityProviderMappers(realm.Spec.RealmName,
			dto.ConvertSSOMappersToIdentityProviderMappers(realm.Spec.SsoRealmName,
				*realm.Spec.SSORealmMappers)); err != nil {
			return errors.Wrap(err, "unable to sync idp mappers")
		}
	}

	rLog.Info("End put identity provider for realm")
	return nextServeOrNil(ctx, h.next, realm, kClient)
}

func (h PutIdentityProvider) setupIdentityProvider(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client,
	rLog logr.Logger, rDto *dto.Realm) error {

	cl := &keycloakApi.KeycloakClient{}
	if err := h.client.Get(ctx, types.NamespacedName{
		Namespace: realm.Namespace,
		Name:      realm.Spec.RealmName,
	}, cl); err != nil {
		return errors.Wrapf(err, "unable to get client: %s", realm.Spec.RealmName)
	}

	e, err := kClient.ExistCentralIdentityProvider(rDto)
	if err != nil {
		return err
	}
	if e {
		rLog.Info("IdP already exists")
		return nil
	}

	s := &coreV1.Secret{}
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      cl.Spec.Secret,
		Namespace: cl.Namespace,
	}, s); err != nil {
		return errors.Wrapf(err, "unable to get secret: %s", cl.Spec.Secret)
	}

	if err := kClient.CreateCentralIdentityProvider(rDto, &dto.Client{
		ClientId:     realm.Spec.RealmName,
		ClientSecret: string(s.Data["clientSecret"]),
	}); err != nil {
		return errors.Wrap(err, "unable to create central identity provider")
	}

	return nil
}
