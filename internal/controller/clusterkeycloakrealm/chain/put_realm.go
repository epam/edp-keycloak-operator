package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type PutRealm struct {
	client client.Client
}

// NewPutRealm returns PutRealm chain handler.
func NewPutRealm(k8sClient client.Client) *PutRealm {
	return &PutRealm{client: k8sClient}
}

func (h PutRealm) ServeRequest(ctx context.Context, realm *v1alpha1.ClusterKeycloakRealm, kClient keycloak.Client) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start putting realm")

	rDto := convertSpecToRealm(&realm.Spec)

	exist, err := kClient.ExistRealm(realm.Spec.RealmName)
	if err != nil {
		return fmt.Errorf("failed to check realm existence: %w", err)
	}

	if exist {
		log.Info("Realm already exists")

		return nil
	}

	err = kClient.CreateRealmWithDefaultConfig(rDto)
	if err != nil {
		return fmt.Errorf("failed to create realm: %w", err)
	}

	log.Info("Realm has been created")

	return nil
}

func convertSpecToRealm(spec *v1alpha1.ClusterKeycloakRealmSpec) *dto.Realm {
	return &dto.Realm{
		Name: spec.RealmName,
	}
}
