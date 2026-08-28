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
)

type PutIDPMappers struct {
	idpClient keycloakapi.IdentityProvidersClient
}

func NewPutIDPMappers(idpClient keycloakapi.IdentityProvidersClient) *PutIDPMappers {
	return &PutIDPMappers{idpClient: idpClient}
}

func (h *PutIDPMappers) Serve(ctx context.Context, keycloakRealmIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start put keycloak idp mappers")

	// Omitted field (nil slice): mappers are managed out-of-band, left untouched.
	// Explicit list, including an empty one, means the operator owns mapper state.
	if keycloakRealmIDP.Spec.Mappers == nil {
		log.Info("Mappers field is not set, leaving existing mappers untouched")
		return nil
	}

	if err := validateMapperNames(keycloakRealmIDP.Spec.Mappers); err != nil {
		return err
	}

	alias := keycloakRealmIDP.Spec.Alias

	existingMappers, _, err := h.idpClient.GetIDPMappers(ctx, realmName, alias)
	if err != nil {
		return fmt.Errorf("unable to get idp mappers: %w", err)
	}

	existingByName := maputil.SliceToMapSelf(existingMappers, func(m keycloakapi.IdentityProviderMapperRepresentation) (string, bool) {
		if m.Name == nil {
			return "", false
		}

		return *m.Name, true
	})

	forceUpdate := helper.SpecChanged(keycloakRealmIDP.Generation, keycloakRealmIDP.Status.ObservedGeneration)

	for _, specMapper := range keycloakRealmIDP.Spec.Mappers {
		mapperAlias := specMapper.IdentityProviderAlias
		if mapperAlias == "" {
			mapperAlias = alias
		}

		desired := specToIDPMapperRepresentation(&specMapper, mapperAlias)

		existing, exists := existingByName[specMapper.Name]
		if !exists {
			if _, err = h.idpClient.CreateIDPMapper(ctx, realmName, alias, desired); err != nil {
				return fmt.Errorf("unable to create idp mapper %s: %w", specMapper.Name, err)
			}

			continue
		}

		delete(existingByName, specMapper.Name)

		if existing.Id == nil || (!forceUpdate && idpMapperMatches(existing, desired)) {
			continue
		}

		desired.Id = existing.Id

		if _, err = h.idpClient.UpdateIDPMapper(ctx, realmName, alias, *existing.Id, desired); err != nil {
			return fmt.Errorf("unable to update idp mapper %s: %w", specMapper.Name, err)
		}
	}

	// Only mappers removed from the spec are deleted.
	for name, m := range existingByName {
		if m.Id == nil {
			continue
		}

		if _, err = h.idpClient.DeleteIDPMapper(ctx, realmName, alias, *m.Id); err != nil {
			return fmt.Errorf("unable to delete idp mapper %s: %w", name, err)
		}
	}

	log.Info("End put keycloak idp mappers")

	return nil
}

// validateMapperNames rejects specs that would break the name-keyed diff in Serve:
// an empty or repeated name can never map back to a single Keycloak mapper.
func validateMapperNames(mappers []keycloakApi.IdentityProviderMapper) error {
	seen := make(map[string]struct{}, len(mappers))

	for _, m := range mappers {
		if _, ok := seen[m.Name]; m.Name == "" || ok {
			return fmt.Errorf("identity provider mapper names must be unique and non-empty; duplicate or empty name %q", m.Name)
		}

		seen[m.Name] = struct{}{}
	}

	return nil
}

func idpMapperMatches(existing, desired keycloakapi.IdentityProviderMapperRepresentation) bool {
	return ptr.Deref(existing.IdentityProviderMapper, "") == ptr.Deref(desired.IdentityProviderMapper, "") &&
		ptr.Deref(existing.IdentityProviderAlias, "") == ptr.Deref(desired.IdentityProviderAlias, "") &&
		maputil.ContainsSubset(ptr.Deref(existing.Config, nil), ptr.Deref(desired.Config, nil))
}

func specToIDPMapperRepresentation(spec *keycloakApi.IdentityProviderMapper, alias string) keycloakapi.IdentityProviderMapperRepresentation {
	config := make(map[string]string, len(spec.Config))
	maps.Copy(config, spec.Config)

	return keycloakapi.IdentityProviderMapperRepresentation{
		Name:                   &spec.Name,
		IdentityProviderAlias:  &alias,
		IdentityProviderMapper: &spec.IdentityProviderMapper,
		Config:                 &config,
	}
}
