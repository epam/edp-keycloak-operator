package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/realmbuilder"
)

// PutRealmSettings is responsible for updating of keycloak realm settings.
type PutRealmSettings struct {
	settingsBuilder *realmbuilder.SettingsBuilder
}

// NewPutRealmSettings creates a new PutRealmSettings handler.
func NewPutRealmSettings() *PutRealmSettings {
	return &PutRealmSettings{
		settingsBuilder: realmbuilder.NewSettingsBuilder(),
	}
}

func (h PutRealmSettings) ServeRequest(ctx context.Context, realm *v1alpha1.ClusterKeycloakRealm, kClient keycloak.Client) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start updating of keycloak realm settings")

	if err := h.settingsBuilder.SetRealmEventConfigFromV1Alpha1(kClient, realm.Spec.RealmName, realm.Spec.RealmEventConfig); err != nil {
		return err
	}

	settings := h.settingsBuilder.BuildFromV1Alpha1(realm)

	if err := kClient.UpdateRealmSettings(realm.Spec.RealmName, &settings); err != nil {
		return fmt.Errorf("unable to update realm settings: %w", err)
	}

	if err := kClient.SetRealmOrganizationsEnabled(ctx, realm.Spec.RealmName, realm.Spec.OrganizationsEnabled); err != nil {
		return fmt.Errorf("unable to set realm organizations enabled: %w", err)
	}

	log.Info("Realm settings is updating done.")

	return nil
}
