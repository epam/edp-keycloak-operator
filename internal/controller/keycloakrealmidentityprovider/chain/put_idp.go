package chain

import (
	"context"
	"fmt"
	"maps"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

type refClient interface {
	MapConfigSecretsRefs(ctx context.Context, config map[string]string, namespace string) error
}

type PutIDP struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
	secretRef         refClient
}

func NewPutIDP(keycloakApiClient keycloak.Client, k8sClient client.Client, secretRef refClient) *PutIDP {
	return &PutIDP{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient, secretRef: secretRef}
}

func (el *PutIDP) Serve(ctx context.Context, keycloakRealmIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	err := el.putKeycloakIDP(ctx, keycloakRealmIDP, realmName)
	if err != nil {
		return fmt.Errorf("unable to put keycloak idp: %w", err)
	}

	return nil
}

func (el *PutIDP) putKeycloakIDP(ctx context.Context, keycloakRealmIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start creation of Keycloak idp")

	var err error

	keycloakIDP := createKeycloakIDPFromSpec(&keycloakRealmIDP.Spec)

	if err = el.secretRef.MapConfigSecretsRefs(ctx, keycloakIDP.Config, keycloakRealmIDP.Namespace); err != nil {
		return fmt.Errorf("unable to map config secrets: %w", err)
	}

	providerExists, err := el.keycloakApiClient.IdentityProviderExists(ctx, realmName, keycloakRealmIDP.Spec.Alias)
	if err != nil {
		return fmt.Errorf("failed to check if the identity provider exists: %w", err)
	}

	if providerExists {
		if err = el.keycloakApiClient.UpdateIdentityProvider(ctx, realmName, keycloakIDP); err != nil {
			return fmt.Errorf("unable to update idp: %w", err)
		}
	} else {
		if err = el.keycloakApiClient.CreateIdentityProvider(ctx, realmName, keycloakIDP); err != nil {
			return fmt.Errorf("unable to create idp: %w", err)
		}
	}

	log.Info("End put keycloak idp")

	return nil
}

func createKeycloakIDPFromSpec(spec *keycloakApi.KeycloakRealmIdentityProviderSpec) *adapter.IdentityProvider {
	p := &adapter.IdentityProvider{
		Config:                    make(map[string]string, len(spec.Config)),
		ProviderID:                spec.ProviderID,
		Alias:                     spec.Alias,
		Enabled:                   spec.Enabled,
		AddReadTokenRoleOnCreate:  spec.AddReadTokenRoleOnCreate,
		AuthenticateByDefault:     spec.AuthenticateByDefault,
		DisplayName:               spec.DisplayName,
		FirstBrokerLoginFlowAlias: spec.FirstBrokerLoginFlowAlias,
		LinkOnly:                  spec.LinkOnly,
		StoreToken:                spec.StoreToken,
		TrustEmail:                spec.TrustEmail,
	}

	maps.Copy(p.Config, spec.Config)

	return p
}
