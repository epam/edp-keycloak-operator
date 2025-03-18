package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type AuthFlow struct {
}

func NewAuthFlow() *AuthFlow {
	return &AuthFlow{}
}

func (a AuthFlow) ServeRequest(ctx context.Context, realm *keycloakApi.ClusterKeycloakRealm, kClient keycloak.Client) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start configuring authentication flow")

	if realm.Spec.AuthenticationFlow == nil || realm.Spec.AuthenticationFlow.BrowserFlow == "" {
		log.Info("Authentication flow is not provided, skip configuring")
		return nil
	}

	if err := kClient.SetRealmBrowserFlow(ctx, realm.Spec.RealmName, realm.Spec.AuthenticationFlow.BrowserFlow); err != nil {
		return fmt.Errorf("setting realm browser flow: %w", err)
	}

	log.Info("Authentication flow has been configured")

	return nil
}
