package chain

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PutIdentityProvider struct {
	next   handler.RealmHandler
	client client.Client
}

func (h PutIdentityProvider) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Name, "realm namespace", realm.Namespace)
	rLog.Info("Start put identity provider for realm...")
	rDto := dto.ConvertSpecToRealm(realm.Spec)
	if !rDto.SsoRealmEnabled {
		rLog.Info("sso realm disabled, skip put identity provider step")
		return nextServeOrNil(h.next, realm, kClient)
	}

	cl := &v1alpha1.KeycloakClient{}
	err := h.client.Get(context.TODO(), types.NamespacedName{
		Namespace: realm.Namespace,
		Name:      realm.Spec.RealmName,
	}, cl)
	if err != nil {
		return err
	}

	e, err := kClient.ExistCentralIdentityProvider(rDto)
	if err != nil {
		return err
	}
	if e {
		rLog.Info("IdP already exists")
		return nextServeOrNil(h.next, realm, kClient)
	}
	s := &coreV1.Secret{}
	err = h.client.Get(context.TODO(), types.NamespacedName{
		Name:      cl.Spec.Secret,
		Namespace: cl.Namespace,
	}, s)
	if err != nil {
		return err
	}
	err = kClient.CreateCentralIdentityProvider(rDto, &dto.Client{
		ClientId:     realm.Spec.RealmName,
		ClientSecret: string(s.Data["clientSecret"]),
	})
	if err != nil {
		return err
	}
	rLog.Info("End put identity provider for realm")
	return nextServeOrNil(h.next, realm, kClient)
}
