package chain

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/realmbuilder"
)

// PutRealmSettings is responsible for updating of keycloak realm settings.
type PutRealmSettings struct{}

// NewPutRealmSettings creates a new PutRealmSettings handler.
func NewPutRealmSettings() *PutRealmSettings {
	return &PutRealmSettings{}
}

func (h PutRealmSettings) ServeRequest(ctx context.Context, realm *v1alpha1.ClusterKeycloakRealm, kClient *keycloakapi.KeycloakClient) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start updating of keycloak realm settings")

	if err := realmbuilder.ApplyRealmEventConfig(ctx, realm.Spec.RealmName, realm.Spec.RealmEventConfig, kClient.Realms); err != nil {
		return err
	}

	overlay := realmbuilder.BuildRealmRepresentationFromV1Alpha1(realm)

	if err := realmbuilder.ApplyRealmSettings(ctx, realm.Spec.RealmName, overlay, kClient.Realms); err != nil {
		return err
	}

	log.Info("Realm settings is updating done.")

	return nil
}
