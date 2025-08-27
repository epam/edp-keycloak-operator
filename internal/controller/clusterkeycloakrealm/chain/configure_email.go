package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakrealmchain "github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type ConfigureEmail struct {
	client     client.Client
	operatorNs string
}

func NewConfigureEmail(k8sClient client.Client, operatorNs string) *ConfigureEmail {
	return &ConfigureEmail{client: k8sClient, operatorNs: operatorNs}
}

func (s ConfigureEmail) ServeRequest(ctx context.Context, realm *keycloakApi.ClusterKeycloakRealm, kClient keycloak.Client) error {
	if realm.Spec.Smtp == nil {
		return nil
	}

	l := ctrl.LoggerFrom(ctx)
	l.Info("Configuring email for realm")

	if err := keycloakrealmchain.ConfigureRamlEmail(
		ctx,
		realm.Spec.RealmName,
		realm.Spec.Smtp,
		s.operatorNs,
		kClient,
		s.client,
	); err != nil {
		return fmt.Errorf("failed to configure email: %w", err)
	}

	l.Info("Email has been configured")

	return nil
}
