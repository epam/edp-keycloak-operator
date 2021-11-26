package chain

import (
	"github.com/Nerzal/gocloak/v10"
	v1v1alpha1 "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/pkg/errors"
)

type PutProtocolMappers struct {
	BaseElement
	next Element
}

func (el *PutProtocolMappers) Serve(keycloakClient *v1v1alpha1.KeycloakClient, adapterClient keycloak.Client) error {
	if err := el.putProtocolMappers(keycloakClient, adapterClient); err != nil {
		return errors.Wrap(err, "unable to put protocol mappers")
	}

	return el.NextServeOrNil(el.next, keycloakClient, adapterClient)
}

func copyMap(in map[string]string) map[string]string {
	out := make(map[string]string)
	for k, v := range in {
		out[k] = v
	}

	return out
}

func (el *PutProtocolMappers) putProtocolMappers(keycloakClient *v1v1alpha1.KeycloakClient, adapterClient keycloak.Client) error {
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
		protocolMappers, keycloakClient.GetReconciliationStrategy() == v1v1alpha1.ReconciliationStrategyAddOnly); err != nil {
		return errors.Wrap(err, "unable to sync protocol mapper")
	}

	return nil
}
