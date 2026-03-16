package chain

import (
	"context"
	"fmt"
	"maps"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

type PutProtocolMappers struct {
	kClient   *keycloakv2.KeycloakClient
	k8sClient client.Client
}

func NewPutProtocolMappers(kClient *keycloakv2.KeycloakClient, k8sClient client.Client) *PutProtocolMappers {
	return &PutProtocolMappers{kClient: kClient, k8sClient: k8sClient}
}

func (h *PutProtocolMappers) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string, clientCtx *ClientContext) error {
	if err := h.putProtocolMappers(ctx, keycloakClient, realmName, clientCtx.ClientUUID); err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync protocol mappers: %s", err.Error()))

		return fmt.Errorf("unable to put protocol mappers: %w", err)
	}

	h.setSuccessCondition(ctx, keycloakClient, "Protocol mappers synchronized")

	return nil
}

func (h *PutProtocolMappers) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionProtocolMappersSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (h *PutProtocolMappers) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionProtocolMappersSynced,
		metav1.ConditionTrue,
		ReasonProtocolMappersSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (h *PutProtocolMappers) putProtocolMappers(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName, clientUUID string) error {
	addOnly := keycloakClient.GetReconciliationStrategy() == keycloakApi.ReconciliationStrategyAddOnly

	// Get existing protocol mappers
	existingMappers, _, err := h.kClient.Clients.GetClientProtocolMappers(ctx, realmName, clientUUID)
	if err != nil {
		return fmt.Errorf("unable to get existing protocol mappers: %w", err)
	}

	existingMapperMap := maputil.SliceToMapSelf(existingMappers, func(m keycloakv2.ProtocolMapperRepresentation) (string, bool) {
		return *m.Name, m.Name != nil
	})

	// Build desired mappers
	desiredMappers := make(map[string]keycloakv2.ProtocolMapperRepresentation)

	if keycloakClient.Spec.ProtocolMappers != nil {
		for _, mapper := range *keycloakClient.Spec.ProtocolMappers {
			configCopy := make(map[string]string, len(mapper.Config))
			maps.Copy(configCopy, mapper.Config)

			desiredMappers[mapper.Name] = keycloakv2.ProtocolMapperRepresentation{
				Name:           ptr.To(mapper.Name),
				Protocol:       ptr.To(mapper.Protocol),
				ProtocolMapper: ptr.To(mapper.ProtocolMapper),
				Config:         &configCopy,
			}
		}
	}

	// Create or update mappers
	for name, desired := range desiredMappers {
		existing, exists := existingMapperMap[name]
		if exists {
			// Update: set the ID from existing mapper
			if existing.Id != nil {
				desired.Id = existing.Id

				if _, err := h.kClient.Clients.UpdateClientProtocolMapper(ctx, realmName, clientUUID, *existing.Id, desired); err != nil {
					return fmt.Errorf("unable to update protocol mapper %s: %w", name, err)
				}
			}

			delete(existingMapperMap, name)
		} else {
			// Create
			if _, err := h.kClient.Clients.CreateClientProtocolMapper(ctx, realmName, clientUUID, desired); err != nil {
				return fmt.Errorf("unable to create protocol mapper %s: %w", name, err)
			}
		}
	}

	// Delete removed mappers (unless addOnly)
	if !addOnly {
		for name, mapper := range existingMapperMap {
			if mapper.Id != nil {
				if _, err := h.kClient.Clients.DeleteClientProtocolMapper(ctx, realmName, clientUUID, *mapper.Id); err != nil {
					if !keycloakv2.IsNotFound(err) {
						return fmt.Errorf("unable to delete protocol mapper %s: %w", name, err)
					}
				}
			}
		}
	}

	return nil
}
