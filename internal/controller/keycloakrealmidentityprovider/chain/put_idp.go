package chain

import (
	"context"
	"fmt"
	"maps"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type refClient interface {
	MapConfigSecretsRefs(ctx context.Context, config map[string]string, namespace string) error
}

type PutIDP struct {
	idpClient keycloakapi.IdentityProvidersClient
	secretRef refClient
}

func NewPutIDP(idpClient keycloakapi.IdentityProvidersClient, secretRef refClient) *PutIDP {
	return &PutIDP{idpClient: idpClient, secretRef: secretRef}
}

func (h *PutIDP) Serve(ctx context.Context, keycloakRealmIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start creation of Keycloak idp")

	config := make(map[string]string, len(keycloakRealmIDP.Spec.Config))
	maps.Copy(config, keycloakRealmIDP.Spec.Config)

	if err := h.secretRef.MapConfigSecretsRefs(ctx, config, keycloakRealmIDP.Namespace); err != nil {
		return fmt.Errorf("unable to map config secrets: %w", err)
	}

	idpRep := specToIdentityProviderRepresentation(&keycloakRealmIDP.Spec, config)

	existingIDP, _, err := h.idpClient.GetIdentityProvider(ctx, realmName, keycloakRealmIDP.Spec.Alias)
	if err != nil && !keycloakapi.IsNotFound(err) {
		return fmt.Errorf("failed to check if the identity provider exists: %w", err)
	}

	if existingIDP != nil {
		if _, err = h.idpClient.UpdateIdentityProvider(ctx, realmName, keycloakRealmIDP.Spec.Alias, idpRep); err != nil {
			return fmt.Errorf("unable to update idp: %w", err)
		}
	} else {
		if _, err = h.idpClient.CreateIdentityProvider(ctx, realmName, idpRep); err != nil {
			return fmt.Errorf("unable to create idp: %w", err)
		}
	}

	log.Info("End put keycloak idp")

	return nil
}

func specToIdentityProviderRepresentation(spec *keycloakApi.KeycloakRealmIdentityProviderSpec, config map[string]string) keycloakapi.IdentityProviderRepresentation {
	return keycloakapi.IdentityProviderRepresentation{
		Alias:                     &spec.Alias,
		ProviderId:                &spec.ProviderID,
		Enabled:                   &spec.Enabled,
		AddReadTokenRoleOnCreate:  &spec.AddReadTokenRoleOnCreate,
		AuthenticateByDefault:     &spec.AuthenticateByDefault,
		DisplayName:               &spec.DisplayName,
		FirstBrokerLoginFlowAlias: ptr.To(spec.FirstBrokerLoginFlowAlias),
		PostBrokerLoginFlowAlias:  ptr.To(spec.PostBrokerLoginFlowAlias),
		LinkOnly:                  &spec.LinkOnly,
		StoreToken:                &spec.StoreToken,
		TrustEmail:                &spec.TrustEmail,
		HideOnLogin:               spec.HideOnLogin,
		Config:                    &config,
	}
}
