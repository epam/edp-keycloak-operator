package chain

import (
	"context"

	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var annotationKey = "openid-configuration"

type PutOpenIdConfigAnnotation struct {
	next   handler.RealmHandler
	client client.Client
}

func (h PutOpenIdConfigAnnotation) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm spec", realm.Spec)
	rLog.Info("Start put openid configuration annotation...")
	if !realm.Spec.SSOEnabled() {
		rLog.Info("sso realm disabled skip openid configuration annotation")
		return nextServeOrNil(h.next, realm, kClient)
	}

	con, err := kClient.GetOpenIdConfig(dto.ConvertSpecToRealm(realm.Spec))
	if err != nil {
		return err
	}
	an := realm.GetAnnotations()
	if an == nil {
		an = make(map[string]string)
	}
	an[annotationKey] = *con
	realm.SetAnnotations(an)
	err = h.client.Update(context.TODO(), realm)
	if err != nil {
		return err
	}

	rLog.Info("end put openid configuration annotation")
	return nextServeOrNil(h.next, realm, kClient)
}
