package chain

import (
	"context"
	"fmt"
	"maps"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
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

	rawSpecConfig := keycloakRealmIDP.Spec.Config

	config := make(map[string]string, len(rawSpecConfig))
	maps.Copy(config, rawSpecConfig)

	if err := h.secretRef.MapConfigSecretsRefs(ctx, config, keycloakRealmIDP.Namespace); err != nil {
		return fmt.Errorf("unable to map config secrets: %w", err)
	}

	idpRep := specToIdentityProviderRepresentation(&keycloakRealmIDP.Spec, config)
	newHash := secretref.ConfigSecretsHashSingle(rawSpecConfig, config)

	existingIDP, _, err := h.idpClient.GetIdentityProvider(ctx, realmName, keycloakRealmIDP.Spec.Alias)
	if err != nil && !keycloakapi.IsNotFound(err) {
		return fmt.Errorf("failed to check if the identity provider exists: %w", err)
	}

	if existingIDP != nil {
		needsUpdate := helper.SpecChanged(keycloakRealmIDP.Generation, keycloakRealmIDP.Status.ObservedGeneration) ||
			newHash != keycloakRealmIDP.Status.ConfigSecretsHash ||
			!idpMatchesSpec(existingIDP, idpRep, rawSpecConfig)

		if needsUpdate {
			if _, err = h.idpClient.UpdateIdentityProvider(ctx, realmName, keycloakRealmIDP.Spec.Alias, idpRep); err != nil {
				return fmt.Errorf("unable to update idp: %w", err)
			}
		}
	} else {
		if _, err = h.idpClient.CreateIdentityProvider(ctx, realmName, idpRep); err != nil {
			return fmt.Errorf("unable to create idp: %w", err)
		}
	}

	// ObservedGeneration is stamped by the controller after the whole chain succeeds; a later
	// handler failure re-runs this update via the generation check.
	keycloakRealmIDP.Status.ConfigSecretsHash = newHash

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

// idpMatchesSpec reports whether the fetched IDP already matches the desired representation.
// Config keys are excluded from comparison when Keycloak masks them: either the raw spec value
// is a secret ref, or the fetched value is already the mask sentinel (e.g. a plain-literal secret).
func idpMatchesSpec(
	existing *keycloakapi.IdentityProviderRepresentation,
	desired keycloakapi.IdentityProviderRepresentation,
	rawSpecConfig map[string]string,
) bool {
	if ptr.Deref(existing.Alias, "") != ptr.Deref(desired.Alias, "") ||
		ptr.Deref(existing.ProviderId, "") != ptr.Deref(desired.ProviderId, "") ||
		ptr.Deref(existing.Enabled, false) != ptr.Deref(desired.Enabled, false) ||
		ptr.Deref(existing.DisplayName, "") != ptr.Deref(desired.DisplayName, "") ||
		ptr.Deref(existing.FirstBrokerLoginFlowAlias, "") != ptr.Deref(desired.FirstBrokerLoginFlowAlias, "") ||
		ptr.Deref(existing.PostBrokerLoginFlowAlias, "") != ptr.Deref(desired.PostBrokerLoginFlowAlias, "") ||
		ptr.Deref(existing.StoreToken, false) != ptr.Deref(desired.StoreToken, false) ||
		ptr.Deref(existing.TrustEmail, false) != ptr.Deref(desired.TrustEmail, false) ||
		ptr.Deref(existing.HideOnLogin, false) != ptr.Deref(desired.HideOnLogin, false) ||
		ptr.Deref(existing.LinkOnly, false) != ptr.Deref(desired.LinkOnly, false) ||
		ptr.Deref(existing.AddReadTokenRoleOnCreate, false) != ptr.Deref(desired.AddReadTokenRoleOnCreate, false) ||
		ptr.Deref(existing.AuthenticateByDefault, false) != ptr.Deref(desired.AuthenticateByDefault, false) {
		return false
	}

	existingConfig := ptr.Deref(existing.Config, nil)
	desiredConfig := ptr.Deref(desired.Config, nil)

	comparableConfig := make(map[string]string, len(desiredConfig))

	for k, v := range desiredConfig {
		if secretref.HasSecretRef(rawSpecConfig[k]) || existingConfig[k] == keycloakapi.MaskedSecretValue {
			continue
		}

		comparableConfig[k] = v
	}

	return maputil.ContainsSubset(existingConfig, comparableConfig)
}
