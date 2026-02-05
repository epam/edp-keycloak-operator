package keycloakv2

import "context"

type Client interface {
}

type UsersClient interface {
	GetUsersProfile(
		ctx context.Context,
		realm string,
	) (*UserProfileConfig, *Response, error)
	UpdateUsersProfile(
		ctx context.Context,
		realm string,
		userProfile UserProfileConfig,
	) (*UserProfileConfig, *Response, error)
}

type RealmClient interface {
	GetRealm(ctx context.Context, realm string) (*RealmRepresentation, *Response, error)
	CreateRealm(ctx context.Context, realmRep RealmRepresentation) (*Response, error)
	UpdateRealm(ctx context.Context, realm string, realmRep RealmRepresentation) (*Response, error)
	DeleteRealm(ctx context.Context, realm string) (*Response, error)
}
