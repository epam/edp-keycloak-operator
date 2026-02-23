package keycloakrealm

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

// Terminator deletes a Keycloak realm during resource cleanup.
type Terminator struct {
	realmName                   string
	realmClient                 keycloakv2.RealmClient
	preserveResourcesOnDeletion bool
}

func (t *Terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("keycloak_realm", t.realmName)
	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	log.Info("Start deleting keycloak realm")

	if _, err := t.realmClient.DeleteRealm(ctx, t.realmName); err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("Realm not found, skipping deletion.")

			return nil
		}

		return fmt.Errorf("failed to delete keycloak realm: %w", err)
	}

	log.Info("Realm has been deleted")

	return nil
}

// MakeTerminator creates a Terminator for the given realm.
func MakeTerminator(realmName string, realmClient keycloakv2.RealmClient, preserveResourcesOnDeletion bool) *Terminator {
	return &Terminator{
		realmName:                   realmName,
		realmClient:                 realmClient,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}

func makeTerminator(realmName string, realmClient keycloakv2.RealmClient, preserveResourcesOnDeletion bool) *Terminator {
	return MakeTerminator(realmName, realmClient, preserveResourcesOnDeletion)
}
