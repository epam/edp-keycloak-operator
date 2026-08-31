package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	keycloakrealmchain "github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/destination"
)

type ConfigureEmail struct {
	client     client.Client
	operatorNs string
	guard      *destination.Guard
}

func NewConfigureEmail(k8sClient client.Client, operatorNs string, guard *destination.Guard) *ConfigureEmail {
	return &ConfigureEmail{client: k8sClient, operatorNs: operatorNs, guard: guard}
}

func (s ConfigureEmail) ServeRequest(ctx context.Context, realm *keycloakApi.ClusterKeycloakRealm, kClient *keycloakapi.KeycloakClient) error {
	if realm.Spec.Smtp == nil {
		return nil
	}

	l := ctrl.LoggerFrom(ctx)
	l.Info("Configuring email for realm")

	newHash, err := keycloakrealmchain.ConfigureRealmEmail(
		ctx,
		realm.Spec.RealmName,
		realm.Spec.Smtp,
		s.operatorNs,
		kClient.Realms,
		s.client,
		realm.Status.ConfigSecretsHash,
		helper.SpecChanged(realm.Generation, realm.Status.ObservedGeneration),
		s.guard,
	)
	if err != nil {
		return fmt.Errorf("failed to configure email: %w", err)
	}

	realm.Status.ConfigSecretsHash = newHash

	l.Info("Email has been configured")

	return nil
}
