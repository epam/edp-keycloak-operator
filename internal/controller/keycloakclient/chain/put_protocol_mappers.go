package chain

import (
	"context"
	"fmt"
	"maps"

	"github.com/Nerzal/gocloak/v12"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type PutProtocolMappers struct {
	keycloakApiClient keycloak.Client
}

func NewPutProtocolMappers(keycloakApiClient keycloak.Client) *PutProtocolMappers {
	return &PutProtocolMappers{keycloakApiClient: keycloakApiClient}
}

func (el *PutProtocolMappers) Serve(_ context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	if err := el.putProtocolMappers(keycloakClient, realmName); err != nil {
		return fmt.Errorf("unable to put protocol mappers: %w", err)
	}

	return nil
}

func (el *PutProtocolMappers) putProtocolMappers(keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	var protocolMappers []gocloak.ProtocolMapperRepresentation

	if keycloakClient.Spec.ProtocolMappers != nil {
		protocolMappers = make([]gocloak.ProtocolMapperRepresentation, 0,
			len(*keycloakClient.Spec.ProtocolMappers))

		for _, mapper := range *keycloakClient.Spec.ProtocolMappers {
			configCopy := make(map[string]string, len(mapper.Config))
			maps.Copy(configCopy, mapper.Config)

			protocolMappers = append(protocolMappers, gocloak.ProtocolMapperRepresentation{
				Name:           gocloak.StringP(mapper.Name),
				Protocol:       gocloak.StringP(mapper.Protocol),
				ProtocolMapper: gocloak.StringP(mapper.ProtocolMapper),
				Config:         &configCopy,
			})
		}
	}

	if err := el.keycloakApiClient.SyncClientProtocolMapper(
		dto.ConvertSpecToClient(&keycloakClient.Spec, "", realmName, nil),
		protocolMappers, keycloakClient.GetReconciliationStrategy() == keycloakApi.ReconciliationStrategyAddOnly); err != nil {
		return fmt.Errorf("unable to sync protocol mapper: %w", err)
	}

	return nil
}
