package chain

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type PutRealm struct{}

// NewPutRealm returns PutRealm chain handler.
func NewPutRealm() *PutRealm {
	return &PutRealm{}
}

func (h PutRealm) ServeRequest(ctx context.Context, realm *v1alpha1.ClusterKeycloakRealm, kClient *keycloakapi.KeycloakClient) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start putting realm")

	realmName := realm.Spec.RealmName

	kcRealm, _, err := kClient.Realms.GetRealm(ctx, realmName)
	if err != nil {
		if !keycloakapi.IsNotFound(err) {
			return fmt.Errorf("failed to check realm existence: %w", err)
		}

		// The realm is gone from Keycloak; a persisted ID belongs to the deleted realm.
		realm.Status.ID = ""

		return h.createRealm(ctx, realm, kClient)
	}

	log.Info("Realm already exists")

	id, err := keycloakapi.RealmID(kcRealm)
	if err != nil {
		return fmt.Errorf("failed to read realm %q id: %w", realmName, err)
	}

	realm.Status.ID = id

	return nil
}

func (h PutRealm) createRealm(ctx context.Context, realm *v1alpha1.ClusterKeycloakRealm, kClient *keycloakapi.KeycloakClient) error {
	realmName := realm.Spec.RealmName

	if _, err := kClient.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{Realm: &realmName, Enabled: ptr.To(true)}); err != nil {
		return fmt.Errorf("failed to create realm: %w", err)
	}

	ctrl.LoggerFrom(ctx).Info("Realm has been created")

	// The create response carries no realm ID.
	kcRealm, _, err := kClient.Realms.GetRealm(ctx, realmName)
	if err != nil {
		return fmt.Errorf("failed to get created realm: %w", err)
	}

	id, err := keycloakapi.RealmID(kcRealm)
	if err != nil {
		return fmt.Errorf("failed to read created realm %q id: %w", realmName, err)
	}

	realm.Status.ID = id

	return nil
}
