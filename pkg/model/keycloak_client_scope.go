package model

type ClientScope struct {
	ID                    *string                `json:"id,omitempty"`
	Name                  *string                `json:"name,omitempty"`
	Description           *string                `json:"description,omitempty"`
	Protocol              *string                `json:"protocol,omitempty"`
	ClientScopeAttributes *ClientScopeAttributes `json:"attributes,omitempty"`
	ProtocolMappers       []*ProtocolMappers     `json:"protocolMappers,omitempty"`
}

type ClientScopeAttributes struct {
	ConsentScreenText      *string `json:"consent.screen.text,omitempty"`
	DisplayOnConsentScreen *string `json:"display.on.consent.screen,omitempty"`
	IncludeInTokenScope    *string `json:"include.in.token.scope,omitempty"`
}

type ProtocolMappers struct {
	ID                    *string                `json:"id,omitempty"`
	Name                  *string                `json:"name,omitempty"`
	Protocol              *string                `json:"protocol,omitempty"`
	ProtocolMapper        *string                `json:"protocolMapper,omitempty"`
	ConsentRequired       *bool                  `json:"consentRequired"`
	ProtocolMappersConfig *ProtocolMappersConfig `json:"config,omitempty"`
}

type ProtocolMappersConfig struct {
	UserinfoTokenClaim                 *string `json:"userinfo.token.claim,omitempty"`
	UserAttribute                      *string `json:"user.attribute,omitempty"`
	IDTokenClaim                       *string `json:"id.token.claim,omitempty"`
	AccessTokenClaim                   *string `json:"access.token.claim,omitempty"`
	ClaimName                          *string `json:"claim.name,omitempty"`
	ClaimValue                         *string `json:"claim.value,omitempty"`
	JSONTypeLabel                      *string `json:"jsonType.label,omitempty"`
	Multivalued                        *string `json:"multivalued,omitempty"`
	UsermodelClientRoleMappingClientID *string `json:"usermodel.clientRoleMapping.clientId,omitempty"`
	IncludedClientAudience             *string `json:"included.client.audience,omitempty"`
}
