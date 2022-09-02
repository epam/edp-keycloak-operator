package chain

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1/v1"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

var annotationKey = "openid-configuration"

type PutOpenIdConfigAnnotation struct {
	next   handler.RealmHandler
	client client.Client
}

func (h PutOpenIdConfigAnnotation) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm spec", realm.Spec)
	rLog.Info("Start put openid configuration annotation...")

	if !realm.Spec.SSOEnabled() {
		rLog.Info("sso realm disabled skip openid configuration annotation")
		return nextServeOrNil(ctx, h.next, realm, kClient)
	}

	con, err := kClient.GetOpenIdConfig(dto.ConvertSpecToRealm(&realm.Spec))
	if err != nil {
		return fmt.Errorf("failed to get openId config: %w", err)
	}

	an := realm.GetAnnotations()
	if an == nil {
		an = make(map[string]string)
	}

	an[annotationKey] = con
	realm.SetAnnotations(an)

	err = h.client.Update(ctx, realm)
	if err != nil {
		return fmt.Errorf("failed to update realm CR: %w", err)
	}

	rLog.Info("end put openid configuration annotation")

	return nextServeOrNil(ctx, h.next, realm, kClient)
}
