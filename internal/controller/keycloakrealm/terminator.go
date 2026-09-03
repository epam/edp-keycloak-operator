package keycloakrealm

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// masterRealmName is the Keycloak admin realm. Keycloak rejects DELETE on it with 400
// "Can't remove master realm"; the name is fixed in the Quarkus distribution.
const masterRealmName = "master"

// Terminator deletes a Keycloak realm during resource cleanup.
type Terminator struct {
	realmName                   string
	realmClient                 keycloakapi.RealmClient
	preserveResourcesOnDeletion bool
}

func (t *Terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("keycloak_realm", t.realmName)
	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	if t.realmName == masterRealmName {
		log.Info("Master realm cannot be deleted in Keycloak, skipping deletion.")
		return nil
	}

	log.Info("Start deleting keycloak realm")

	if _, err := t.realmClient.DeleteRealm(ctx, t.realmName); err != nil {
		if keycloakapi.IsNotFound(err) {
			log.Info("Realm not found, skipping deletion.")

			return nil
		}

		return fmt.Errorf("failed to delete keycloak realm: %w", err)
	}

	log.Info("Realm has been deleted")

	return nil
}

// MakeTerminator creates a Terminator for the given realm.
func MakeTerminator(realmName string, realmClient keycloakapi.RealmClient, preserveResourcesOnDeletion bool) *Terminator {
	return &Terminator{
		realmName:                   realmName,
		realmClient:                 realmClient,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}

func makeTerminator(realmName string, realmClient keycloakapi.RealmClient, preserveResourcesOnDeletion bool) *Terminator {
	return MakeTerminator(realmName, realmClient, preserveResourcesOnDeletion)
}
