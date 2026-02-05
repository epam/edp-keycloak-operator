package chain

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

const TargetRealmLabel = "targetRealm"

type SetLabels struct {
	next   handler.RealmHandler
	client client.Client
}

func (s SetLabels) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client, kClientV2 *keycloakv2.KeycloakClient) error {
	if realm.Labels == nil {
		realm.Labels = make(map[string]string)
	}

	if tr, ok := realm.Labels[TargetRealmLabel]; !ok || tr != realm.Spec.RealmName {
		realm.Labels[TargetRealmLabel] = realm.Spec.RealmName
	}

	if err := s.client.Update(ctx, realm); err != nil {
		return fmt.Errorf("unable to update realm with new labels, realm: %+v: %w", realm, err)
	}

	return nextServeOrNil(ctx, s.next, realm, kClient, kClientV2)
}
