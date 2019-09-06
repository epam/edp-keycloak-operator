package keycloak

import "keycloak-operator/pkg/apis/v1/v1alpha1"

type Client interface {
	ExistRealm(spec v1alpha1.KeycloakRealmSpec) (*bool, error)

	CreateRealmWithDefaultConfig(spec v1alpha1.KeycloakRealmSpec) error

	ExistCentralIdentityProvider(spec v1alpha1.KeycloakRealmSpec) (*bool, error)

	CreateCentralIdentityProvider(rSpec v1alpha1.KeycloakRealmSpec, cSpec v1alpha1.KeycloakClientSpec) error
}

type ClientFactory interface {
	New(spec v1alpha1.KeycloakSpec) (Client, error)
}
