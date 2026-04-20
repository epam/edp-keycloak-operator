package chain

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type PutRealm struct {
	client client.Client
}

// NewPutRealm returns PutRealm chain handler.
func NewPutRealm(k8sClient client.Client) *PutRealm {
	return &PutRealm{client: k8sClient}
}

func (h PutRealm) ServeRequest(ctx context.Context, realm *v1alpha1.ClusterKeycloakRealm, kClient *keycloakapi.KeycloakClient) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start putting realm")

	_, _, err := kClient.Realms.GetRealm(ctx, realm.Spec.RealmName)
	if err == nil {
		log.Info("Realm already exists")

		return nil
	}

	if !keycloakapi.IsNotFound(err) {
		return fmt.Errorf("failed to check realm existence: %w", err)
	}

	realmName := realm.Spec.RealmName
	if _, err = kClient.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{Realm: &realmName, Enabled: ptr.To(true)}); err != nil {
		return fmt.Errorf("failed to create realm: %w", err)
	}

	log.Info("Realm has been created")

	return nil
}
