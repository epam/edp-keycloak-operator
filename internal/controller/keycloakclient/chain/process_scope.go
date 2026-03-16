package chain

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

const scopeLogKey = "scope"

type ProcessScope struct {
	kClient   *keycloakv2.KeycloakClient
	k8sClient client.Client
}

func NewProcessScope(kClient *keycloakv2.KeycloakClient, k8sClient client.Client) *ProcessScope {
	return &ProcessScope{kClient: kClient, k8sClient: k8sClient}
}

func (h *ProcessScope) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string, clientCtx *ClientContext) error {
	log := ctrl.LoggerFrom(ctx)

	if keycloakClient.Spec.Authorization == nil {
		log.Info("Authorization settings are not specified")
		return nil
	}

	clientUUID := clientCtx.ClientUUID

	scopesList, _, err := h.kClient.Authorization.GetScopes(ctx, realmName, clientUUID)
	if err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization scopes: %s", err.Error()))

		return fmt.Errorf("failed to get scopes: %w", err)
	}

	existingScopes := maputil.SliceToMapSelf(scopesList, func(s keycloakv2.ScopeRepresentation) (string, bool) {
		return *s.Name, s.Name != nil
	})

	for _, scope := range keycloakClient.Spec.Authorization.Scopes {
		log.Info("Processing scope", scopeLogKey, scope)

		_, ok := existingScopes[scope]
		if ok {
			log.Info("Scope already exists")
			delete(existingScopes, scope)

			continue
		}

		scopeRep := keycloakv2.ScopeRepresentation{Name: &scope}
		if _, err = h.kClient.Authorization.CreateScope(ctx, realmName, clientUUID, scopeRep); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization scopes: %s", err.Error()))

			return fmt.Errorf("failed to create scope: %w", err)
		}

		log.Info("Scope created", scopeLogKey, scope)

		delete(existingScopes, scope)
	}

	if keycloakClient.Spec.ReconciliationStrategy != keycloakApi.ReconciliationStrategyAddOnly {
		if err = h.deleteScopes(ctx, existingScopes, realmName, clientUUID); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization scopes: %s", err.Error()))

			return err
		}
	}

	h.setSuccessCondition(ctx, keycloakClient, "Authorization scopes synchronized")

	return nil
}

func (h *ProcessScope) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionAuthorizationScopesSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (h *ProcessScope) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionAuthorizationScopesSynced,
		metav1.ConditionTrue,
		ReasonAuthorizationScopesSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (h *ProcessScope) deleteScopes(ctx context.Context, existingScopes map[string]keycloakv2.ScopeRepresentation, realmName string, clientUUID string) error {
	log := ctrl.LoggerFrom(ctx)

	for name := range existingScopes {
		scope := existingScopes[name]
		if scope.Id == nil {
			continue
		}

		if _, err := h.kClient.Authorization.DeleteScope(ctx, realmName, clientUUID, *scope.Id); err != nil {
			if !keycloakv2.IsNotFound(err) {
				return fmt.Errorf("failed to delete scope: %w", err)
			}
		}

		log.Info("Scope deleted", scopeLogKey, name)
	}

	return nil
}
