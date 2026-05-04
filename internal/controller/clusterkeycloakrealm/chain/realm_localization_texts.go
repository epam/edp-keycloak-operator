package chain

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakrealmchain "github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type PutRealmLocalizationTexts struct{}

func NewPutRealmLocalizationTexts() *PutRealmLocalizationTexts {
	return &PutRealmLocalizationTexts{}
}

func (h PutRealmLocalizationTexts) ServeRequest(ctx context.Context, realm *v1alpha1.ClusterKeycloakRealm, kClient *keycloakapi.KeycloakClient) error {
	if realm.Spec.Localization == nil || len(realm.Spec.Localization.LocalizationTexts) == 0 {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Start applying realm localization texts")

	if err := keycloakrealmchain.SyncLocalizationTexts(ctx, kClient, realm.Spec.RealmName, realm.Spec.Localization.LocalizationTexts); err != nil {
		return err
	}

	log.Info("Realm localization texts applied")

	return nil
}
