package chain

import (
	"context"
	"fmt"
	"maps"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type PutIDPMappers struct {
	idpClient keycloakv2.IdentityProvidersClient
}

func NewPutIDPMappers(idpClient keycloakv2.IdentityProvidersClient) *PutIDPMappers {
	return &PutIDPMappers{idpClient: idpClient}
}

func (h *PutIDPMappers) Serve(ctx context.Context, keycloakRealmIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start put keycloak idp mappers")

	if len(keycloakRealmIDP.Spec.Mappers) == 0 {
		return nil
	}

	mappers, _, err := h.idpClient.GetIDPMappers(ctx, realmName, keycloakRealmIDP.Spec.Alias)
	if err != nil {
		return fmt.Errorf("unable to get idp mappers: %w", err)
	}

	for _, m := range mappers {
		if m.Id == nil {
			continue
		}

		if _, err = h.idpClient.DeleteIDPMapper(ctx, realmName, keycloakRealmIDP.Spec.Alias, *m.Id); err != nil {
			return fmt.Errorf("unable to delete idp mapper: %w", err)
		}
	}

	for _, m := range keycloakRealmIDP.Spec.Mappers {
		alias := m.IdentityProviderAlias
		if alias == "" {
			alias = keycloakRealmIDP.Spec.Alias
		}

		mapperRep := specToIDPMapperRepresentation(&m, alias)

		if _, err = h.idpClient.CreateIDPMapper(ctx, realmName, keycloakRealmIDP.Spec.Alias, mapperRep); err != nil {
			return fmt.Errorf("unable to create idp mapper: %w", err)
		}
	}

	log.Info("End put keycloak idp mappers")

	return nil
}

func specToIDPMapperRepresentation(spec *keycloakApi.IdentityProviderMapper, alias string) keycloakv2.IdentityProviderMapperRepresentation {
	config := make(map[string]string, len(spec.Config))
	maps.Copy(config, spec.Config)

	return keycloakv2.IdentityProviderMapperRepresentation{
		Name:                   &spec.Name,
		IdentityProviderAlias:  &alias,
		IdentityProviderMapper: &spec.IdentityProviderMapper,
		Config:                 &config,
	}
}
