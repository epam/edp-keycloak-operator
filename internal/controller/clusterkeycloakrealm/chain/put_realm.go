package chain

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type PutRealm struct {
	client client.Client
}

// NewPutRealm returns PutRealm chain handler.
func NewPutRealm(k8sClient client.Client) *PutRealm {
	return &PutRealm{client: k8sClient}
}

func (h PutRealm) ServeRequest(ctx context.Context, realm *v1alpha1.ClusterKeycloakRealm, kClientV2 *keycloakv2.KeycloakClient) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start putting realm")

	_, _, err := kClientV2.Realms.GetRealm(ctx, realm.Spec.RealmName)
	if err == nil {
		log.Info("Realm already exists")

		return nil
	}

	if !keycloakv2.IsNotFound(err) {
		return fmt.Errorf("failed to check realm existence: %w", err)
	}

	realmName := realm.Spec.RealmName
	if _, err = kClientV2.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{Realm: &realmName, Enabled: ptr.To(true)}); err != nil {
		return fmt.Errorf("failed to create realm: %w", err)
	}

	log.Info("Realm has been created")

	return nil
}
