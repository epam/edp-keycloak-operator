package chain

import (
	"context"
	"fmt"
	"maps"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type SyncProtocolMappers struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewSyncProtocolMappers(kClientV2 *keycloakv2.KeycloakClient) *SyncProtocolMappers {
	return &SyncProtocolMappers{kClientV2: kClientV2}
}

func (h *SyncProtocolMappers) Serve(
	ctx context.Context,
	scope *keycloakApi.KeycloakClientScope,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing protocol mappers for client scope")

	scopesClient := h.kClientV2.ClientScopes
	scopeID := scope.Status.ID

	existingMappers, _, err := scopesClient.GetClientScopeProtocolMappers(ctx, realmName, scopeID)
	if err != nil {
		return fmt.Errorf("failed to get existing protocol mappers: %w", err)
	}

	for _, mapper := range existingMappers {
		if mapper.Id == nil {
			continue
		}

		if _, err := scopesClient.DeleteClientScopeProtocolMapper(ctx, realmName, scopeID, *mapper.Id); err != nil {
			return fmt.Errorf("failed to delete protocol mapper %s: %w", *mapper.Id, err)
		}
	}

	for _, specMapper := range scope.Spec.ProtocolMappers {
		mapper := convertProtocolMapper(specMapper)

		if _, err := scopesClient.CreateClientScopeProtocolMapper(ctx, realmName, scopeID, mapper); err != nil {
			return fmt.Errorf("failed to create protocol mapper %s: %w", specMapper.Name, err)
		}
	}

	log.Info("Protocol mappers have been synced")

	return nil
}

func convertProtocolMapper(m keycloakApi.ProtocolMapper) keycloakv2.ProtocolMapperRepresentation {
	config := make(map[string]string, len(m.Config))
	maps.Copy(config, m.Config)

	return keycloakv2.ProtocolMapperRepresentation{
		Name:           &m.Name,
		Protocol:       &m.Protocol,
		ProtocolMapper: &m.ProtocolMapper,
		Config:         &config,
	}
}
