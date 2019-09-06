package api

type IdentityProviderRepresentation struct {
	Alias       string                 `json:"alias"`
	DisplayName string                 `json:"displayName"`
	Enabled     bool                   `json:"enabled"`
	ProviderId  string                 `json:"providerId"`
	Config      IdentityProviderConfig `json:"config"`
}

type IdentityProviderConfig struct {
	UserInfoUrl      string `json:"userInfoUrl"`
	TokenUrl         string `json:"tokenUrl"`
	JwksUrl          string `json:"jwksUrl"`
	Issuer           string `json:"issuer"`
	AuthorizationUrl string `json:"authorizationUrl"`
	LogoutUrl        string `json:"logoutUrl"`
	ClientId         string `json:"clientId"`
	ClientSecret     string `json:"clientSecret"`
}

type IdentityProviderMapperRepresentation struct {
	Config                 map[string]string `json:"config"`
	IdentityProviderAlias  string            `json:"identityProviderAlias"`
	IdentityProviderMapper string            `json:"identityProviderMapper"`
	Name                   string            `json:"name"`
}
