package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const (
	orgSettingsMinMajor = 26
	orgSettingsMinMinor = 8
)

// ValidateServerVersion rejects fields that need Keycloak orgSettingsMinMajor.orgSettingsMinMinor or later
// on an older server, before any handler writes to Keycloak.
// A version that cannot be read fails the reconcile; a version that cannot be parsed skips the check.
type ValidateServerVersion struct {
	serverInfo keycloakapi.ServerInfoClient
}

func NewValidateServerVersion(kc *keycloakapi.KeycloakClient) *ValidateServerVersion {
	return &ValidateServerVersion{serverInfo: kc.Server}
}

func (h *ValidateServerVersion) ServeRequest(
	ctx context.Context,
	organization *keycloakApi.KeycloakOrganization,
	_ string,
) error {
	field := firstGatedField(organization)
	if field == "" {
		return nil
	}

	info, err := h.serverInfo.GetServerInfo(ctx)
	if err != nil {
		return fmt.Errorf("%s: unable to read the Keycloak version to check the %d.%d requirement: %w",
			field, orgSettingsMinMajor, orgSettingsMinMinor, err)
	}

	supported, err := keycloakapi.ServerVersionAtLeast(info, orgSettingsMinMajor, orgSettingsMinMinor)
	if err != nil {
		ctrl.LoggerFrom(ctx).Info(
			fmt.Sprintf("Keycloak version not recognized, skipping the %d.%d field check",
				orgSettingsMinMajor, orgSettingsMinMinor),
			"error", err.Error())

		return nil
	}

	if !supported {
		return fmt.Errorf("%s requires Keycloak %d.%d or later, server version is %s",
			field, orgSettingsMinMajor, orgSettingsMinMinor, info.SystemInfo.ServerVersion)
	}

	return nil
}

// firstGatedField names the first field that needs the minimum Keycloak version: domainsV2 in spec order,
// then identityProviders. Returns "" when the spec sets none.
func firstGatedField(organization *keycloakApi.KeycloakOrganization) string {
	for i, d := range organization.Spec.DomainsV2 {
		if field := gatedDomainField(d); field != "" {
			return fmt.Sprintf("domainsV2[%d] %q: %s", i, d.Name, field)
		}
	}

	for i, idp := range organization.Spec.IdentityProviders {
		if !hasLinkSettings(idp) {
			continue
		}

		field := "membershipType"
		if idp.AutoMembership != nil {
			field = "autoMembership"
		}

		return fmt.Sprintf("identityProviders[%d] %q: %s", i, idp.Alias, field)
	}

	return ""
}
