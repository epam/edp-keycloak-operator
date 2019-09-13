package keycloak

import (
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
)

type Client interface {
	ExistRealm(realm dto.Realm) (*bool, error)

	CreateRealmWithDefaultConfig(realm dto.Realm) error

	ExistCentralIdentityProvider(realm dto.Realm) (*bool, error)

	CreateCentralIdentityProvider(realm dto.Realm, client dto.Client) error

	ExistClient(client dto.Client) (*bool, error)

	CreateClient(client dto.Client) error

	ExistRealmRole(realm dto.Realm, role dto.RealmRole) (*bool, error)

	CreateRealmRole(realm dto.Realm, role dto.RealmRole) error
}

type ClientFactory interface {
	New(keycloak dto.Keycloak) (Client, error)
}
