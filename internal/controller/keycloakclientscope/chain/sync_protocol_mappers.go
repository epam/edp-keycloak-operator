package chain

import (
	"context"
	"fmt"
	"maps"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

type SyncProtocolMappers struct {
	kClient *keycloakapi.KeycloakClient
}

func NewSyncProtocolMappers(kClient *keycloakapi.KeycloakClient) *SyncProtocolMappers {
	return &SyncProtocolMappers{kClient: kClient}
}

func (h *SyncProtocolMappers) Serve(
	ctx context.Context,
	scope *keycloakApi.KeycloakClientScope,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing protocol mappers for client scope")

	scopesClient := h.kClient.ClientScopes
	scopeID := scope.Status.ID

	existingMappers, _, err := scopesClient.GetClientScopeProtocolMappers(ctx, realmName, scopeID)
	if err != nil {
		return fmt.Errorf("failed to get existing protocol mappers: %w", err)
	}

	existingByName := maputil.SliceToMapSelf(existingMappers, func(m keycloakapi.ProtocolMapperRepresentation) (string, bool) {
		if m.Name == nil {
			return "", false
		}

		return *m.Name, true
	})

	forceUpdate := specChanged(scope)

	for _, specMapper := range scope.Spec.ProtocolMappers {
		desired := convertProtocolMapper(specMapper)

		existing, exists := existingByName[specMapper.Name]
		if !exists {
			if _, err := scopesClient.CreateClientScopeProtocolMapper(ctx, realmName, scopeID, desired); err != nil {
				return fmt.Errorf("failed to create protocol mapper %s: %w", specMapper.Name, err)
			}

			continue
		}

		delete(existingByName, specMapper.Name)

		if existing.Id == nil || (!forceUpdate && protocolMapperMatches(existing, desired)) {
			continue
		}

		desired.Id = existing.Id

		if _, err := scopesClient.UpdateClientScopeProtocolMapper(ctx, realmName, scopeID, *existing.Id, desired); err != nil {
			return fmt.Errorf("failed to update protocol mapper %s: %w", specMapper.Name, err)
		}
	}

	// Only mappers removed from the spec are deleted.
	for name, mapper := range existingByName {
		if mapper.Id == nil {
			continue
		}

		if _, err := scopesClient.DeleteClientScopeProtocolMapper(ctx, realmName, scopeID, *mapper.Id); err != nil {
			return fmt.Errorf("failed to delete protocol mapper %s: %w", name, err)
		}
	}

	log.Info("Protocol mappers have been synced")

	return nil
}

func protocolMapperMatches(existing, desired keycloakapi.ProtocolMapperRepresentation) bool {
	return ptr.Deref(existing.Protocol, "") == ptr.Deref(desired.Protocol, "") &&
		ptr.Deref(existing.ProtocolMapper, "") == ptr.Deref(desired.ProtocolMapper, "") &&
		containsConfig(ptr.Deref(existing.Config, nil), ptr.Deref(desired.Config, nil))
}

func convertProtocolMapper(m keycloakApi.ProtocolMapper) keycloakapi.ProtocolMapperRepresentation {
	config := make(map[string]string, len(m.Config))
	maps.Copy(config, m.Config)

	return keycloakapi.ProtocolMapperRepresentation{
		Name:           &m.Name,
		Protocol:       &m.Protocol,
		ProtocolMapper: &m.ProtocolMapper,
		Config:         &config,
	}
}
