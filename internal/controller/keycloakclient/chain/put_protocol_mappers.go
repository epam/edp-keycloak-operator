package chain

import (
	"context"
	"fmt"
	"maps"

	"github.com/Nerzal/gocloak/v12"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type PutProtocolMappers struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewPutProtocolMappers(keycloakApiClient keycloak.Client, k8sClient client.Client) *PutProtocolMappers {
	return &PutProtocolMappers{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient}
}

func (el *PutProtocolMappers) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	if err := el.putProtocolMappers(keycloakClient, realmName); err != nil {
		el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync protocol mappers: %s", err.Error()))

		return fmt.Errorf("unable to put protocol mappers: %w", err)
	}

	el.setSuccessCondition(ctx, keycloakClient, "Protocol mappers synchronized")

	return nil
}

func (el *PutProtocolMappers) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		ConditionProtocolMappersSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (el *PutProtocolMappers) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		ConditionProtocolMappersSynced,
		metav1.ConditionTrue,
		ReasonProtocolMappersSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
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
