package chain

import (
	"context"

	"github.com/Nerzal/gocloak/v10"
	"github.com/pkg/errors"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type PutProtocolMappers struct {
	BaseElement
	next Element
}

func (el *PutProtocolMappers) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client) error {
	if err := el.putProtocolMappers(keycloakClient, adapterClient); err != nil {
		return errors.Wrap(err, "unable to put protocol mappers")
	}

	return el.NextServeOrNil(ctx, el.next, keycloakClient, adapterClient)
}

func copyMap(in map[string]string) map[string]string {
	out := make(map[string]string)
	for k, v := range in {
		out[k] = v
	}

	return out
}

func (el *PutProtocolMappers) putProtocolMappers(keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client) error {
	var protocolMappers []gocloak.ProtocolMapperRepresentation

	if keycloakClient.Spec.ProtocolMappers != nil {
		protocolMappers = make([]gocloak.ProtocolMapperRepresentation, 0,
			len(*keycloakClient.Spec.ProtocolMappers))

		for _, mapper := range *keycloakClient.Spec.ProtocolMappers {
			configCopy := copyMap(mapper.Config)

			protocolMappers = append(protocolMappers, gocloak.ProtocolMapperRepresentation{
				Name:           gocloak.StringP(mapper.Name),
				Protocol:       gocloak.StringP(mapper.Protocol),
				ProtocolMapper: gocloak.StringP(mapper.ProtocolMapper),
				Config:         &configCopy,
			})
		}
	}

	if err := adapterClient.SyncClientProtocolMapper(
		dto.ConvertSpecToClient(&keycloakClient.Spec, ""),
		protocolMappers, keycloakClient.GetReconciliationStrategy() == keycloakApi.ReconciliationStrategyAddOnly); err != nil {
		return errors.Wrap(err, "unable to sync protocol mapper")
	}

	return nil
}
