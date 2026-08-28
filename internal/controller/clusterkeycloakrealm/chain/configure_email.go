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
)

type ConfigureEmail struct {
	client     client.Client
	operatorNs string
}

func NewConfigureEmail(k8sClient client.Client, operatorNs string) *ConfigureEmail {
	return &ConfigureEmail{client: k8sClient, operatorNs: operatorNs}
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
	)
	if err != nil {
		return fmt.Errorf("failed to configure email: %w", err)
	}

	realm.Status.ConfigSecretsHash = newHash

	l.Info("Email has been configured")

	return nil
}
