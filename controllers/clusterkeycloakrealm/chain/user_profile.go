package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakrealmchain "github.com/epam/edp-keycloak-operator/controllers/keycloakrealm/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type UserProfile struct {
}

func NewUserProfile() *UserProfile {
	return &UserProfile{}
}

func (h UserProfile) ServeRequest(ctx context.Context, realm *v1alpha1.ClusterKeycloakRealm, kClient keycloak.Client) error {
	l := ctrl.LoggerFrom(ctx)

	if realm.Spec.UserProfileConfig == nil {
		l.Info("User profile is empty, skipping configuration")

		return nil
	}

	l.Info("Start configuring keycloak realm user profile")

	err := keycloakrealmchain.ProcessUserProfile(ctx, realm.Spec.RealmName, realm.Spec.UserProfileConfig, kClient)
	if err != nil {
		return fmt.Errorf("unable to process user profile: %w", err)
	}

	l.Info("User profile has been configured")

	return nil
}
