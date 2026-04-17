package chain

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

type PutClientScope struct {
	kClient   *keycloakapi.APIClient
	k8sClient client.Client
}

func NewPutClientScope(kClient *keycloakapi.APIClient, k8sClient client.Client) *PutClientScope {
	return &PutClientScope{kClient: kClient, k8sClient: k8sClient}
}

func (h *PutClientScope) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string, clientCtx *ClientContext) error {
	if err := h.putClientScope(ctx, keycloakClient, realmName, clientCtx.ClientUUID); err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync client scopes: %s", err.Error()))

		return fmt.Errorf("error during putClientScope: %w", err)
	}

	h.setSuccessCondition(ctx, keycloakClient, "Client scopes synchronized")

	return nil
}

func (h *PutClientScope) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionClientScopesSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (h *PutClientScope) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionClientScopesSynced,
		metav1.ConditionTrue,
		ReasonClientScopesSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (h *PutClientScope) putClientScope(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName, clientUUID string) error {
	kCloakSpec := keycloakClient.Spec

	if len(kCloakSpec.DefaultClientScopes) == 0 && len(kCloakSpec.OptionalClientScopes) == 0 {
		return nil
	}

	// Get all realm client scopes once to build name->id mapping
	realmScopes, _, err := h.kClient.Clients.GetRealmClientScopes(ctx, realmName)
	if err != nil {
		return fmt.Errorf("error during GetRealmClientScopes: %w", err)
	}

	scopeNameToID := maputil.SliceToMap(realmScopes,
		func(s keycloakapi.ClientScopeRepresentation) (string, bool) {
			return *s.Name, s.Name != nil && s.Id != nil
		},
		func(s keycloakapi.ClientScopeRepresentation) string { return *s.Id },
	)

	if err := h.putDefaultClientScope(ctx, keycloakClient, realmName, clientUUID, scopeNameToID); err != nil {
		return err
	}

	if err := h.putOptionalClientScope(ctx, keycloakClient, realmName, clientUUID, scopeNameToID); err != nil {
		return err
	}

	return nil
}

func (h *PutClientScope) putDefaultClientScope(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName, clientUUID string, scopeNameToID map[string]string) error {
	kCloakSpec := keycloakClient.Spec

	if len(kCloakSpec.DefaultClientScopes) == 0 {
		return nil
	}

	// Get existing default scopes for this client
	existingDefaults, _, err := h.kClient.Clients.GetDefaultClientScopes(ctx, realmName, clientUUID)
	if err != nil {
		return fmt.Errorf("error getting existing default scopes: %w", err)
	}

	existingDefaultIDs := make(map[string]bool, len(existingDefaults))

	for _, s := range existingDefaults {
		if s.Id != nil {
			existingDefaultIDs[*s.Id] = true
		}
	}

	// Add missing default scopes
	for _, scopeName := range kCloakSpec.DefaultClientScopes {
		scopeID, ok := scopeNameToID[scopeName]
		if !ok {
			return fmt.Errorf("client scope %s not found in realm %s", scopeName, realmName)
		}

		if existingDefaultIDs[scopeID] {
			continue
		}

		if _, err := h.kClient.Clients.AddDefaultClientScope(ctx, realmName, clientUUID, scopeID); err != nil {
			return fmt.Errorf("failed to add default scope %s to client %s: %w", scopeName, keycloakClient.Name, err)
		}
	}

	return nil
}

func (h *PutClientScope) putOptionalClientScope(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName, clientUUID string, scopeNameToID map[string]string) error {
	kCloakSpec := keycloakClient.Spec

	if len(kCloakSpec.OptionalClientScopes) == 0 {
		return nil
	}

	// Get existing optional scopes for this client
	existingOptionals, _, err := h.kClient.Clients.GetOptionalClientScopes(ctx, realmName, clientUUID)
	if err != nil {
		return fmt.Errorf("error getting existing optional scopes: %w", err)
	}

	existingOptionalIDs := make(map[string]bool, len(existingOptionals))

	for _, s := range existingOptionals {
		if s.Id != nil {
			existingOptionalIDs[*s.Id] = true
		}
	}

	// Add missing optional scopes
	for _, scopeName := range kCloakSpec.OptionalClientScopes {
		scopeID, ok := scopeNameToID[scopeName]
		if !ok {
			return fmt.Errorf("client scope %s not found in realm %s", scopeName, realmName)
		}

		if existingOptionalIDs[scopeID] {
			continue
		}

		if _, err := h.kClient.Clients.AddOptionalClientScope(ctx, realmName, clientUUID, scopeID); err != nil {
			return fmt.Errorf("failed to add optional scope %s to client %s: %w", scopeName, keycloakClient.Name, err)
		}
	}

	return nil
}
