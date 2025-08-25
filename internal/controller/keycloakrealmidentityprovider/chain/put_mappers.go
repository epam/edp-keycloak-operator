package chain

import (
	"context"
	"maps"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

type PutIDPMappers struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
	secretRef         refClient
}

func NewPutIDPMappers(keycloakApiClient keycloak.Client, k8sClient client.Client, secretRef refClient) *PutIDPMappers {
	return &PutIDPMappers{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient, secretRef: secretRef}
}

func (el *PutIDPMappers) Serve(ctx context.Context, keycloakRealmIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	err := syncIDPMappers(ctx, &keycloakRealmIDP.Spec, el.keycloakApiClient, realmName)
	if err != nil {
		return errors.Wrap(err, "unable to sync idp mappers")
	}

	return nil
}

func syncIDPMappers(ctx context.Context, idpSpec *keycloakApi.KeycloakRealmIdentityProviderSpec, kClient keycloak.Client, targetRealm string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak idp mappers")

	if len(idpSpec.Mappers) == 0 {
		return nil
	}

	mappers, err := kClient.GetIDPMappers(ctx, targetRealm, idpSpec.Alias)
	if err != nil {
		return errors.Wrap(err, "unable to get idp mappers")
	}

	for _, m := range mappers {
		if err = kClient.DeleteIDPMapper(ctx, targetRealm, idpSpec.Alias, m.ID); err != nil {
			return errors.Wrap(err, "unable to delete idp mapper")
		}
	}

	for _, m := range idpSpec.Mappers {
		if m.IdentityProviderAlias == "" {
			m.IdentityProviderAlias = idpSpec.Alias
		}

		if _, err = kClient.CreateIDPMapper(ctx, targetRealm, idpSpec.Alias,
			createKeycloakIDPMapperFromSpec(&m)); err != nil {
			return errors.Wrap(err, "unable to create idp mapper")
		}
	}

	reqLog.Info("End put keycloak idp mappers")

	return nil
}

func createKeycloakIDPMapperFromSpec(spec *keycloakApi.IdentityProviderMapper) *adapter.IdentityProviderMapper {
	m := &adapter.IdentityProviderMapper{
		IdentityProviderMapper: spec.IdentityProviderMapper,
		Name:                   spec.Name,
		Config:                 make(map[string]string, len(spec.Config)),
		IdentityProviderAlias:  spec.IdentityProviderAlias,
	}

	maps.Copy(m.Config, spec.Config)

	return m
}
