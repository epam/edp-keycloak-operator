package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapimocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
	secretrefmocks "github.com/epam/edp-keycloak-operator/pkg/secretref/mocks"
)

// inSyncIDPSpec and inSyncIDPRepresentation describe the same identity provider from the spec
// side and the fetched Keycloak side, used as a baseline for the idempotency test cases below.
func inSyncIDPSpec() keycloakApi.KeycloakRealmIdentityProviderSpec {
	return keycloakApi.KeycloakRealmIdentityProviderSpec{
		Alias:                     "test-idp",
		ProviderID:                "github",
		Enabled:                   true,
		AddReadTokenRoleOnCreate:  true,
		AuthenticateByDefault:     true,
		DisplayName:               "Test IDP",
		FirstBrokerLoginFlowAlias: "first-broker",
		PostBrokerLoginFlowAlias:  "post-broker",
		LinkOnly:                  true,
		StoreToken:                true,
		TrustEmail:                true,
		HideOnLogin:               ptr.To(true),
		Config:                    map[string]string{"clientId": "test-client"},
	}
}

func inSyncIDPRepresentation() *keycloakapi.IdentityProviderRepresentation {
	return &keycloakapi.IdentityProviderRepresentation{
		Alias:                     ptr.To("test-idp"),
		ProviderId:                ptr.To("github"),
		Enabled:                   ptr.To(true),
		AddReadTokenRoleOnCreate:  ptr.To(true),
		AuthenticateByDefault:     ptr.To(true),
		DisplayName:               ptr.To("Test IDP"),
		FirstBrokerLoginFlowAlias: ptr.To("first-broker"),
		PostBrokerLoginFlowAlias:  ptr.To("post-broker"),
		LinkOnly:                  ptr.To(true),
		StoreToken:                ptr.To(true),
		TrustEmail:                ptr.To(true),
		HideOnLogin:               ptr.To(true),
		Config:                    &map[string]string{"clientId": "test-client"},
	}
}

func noSecretRefMock(t *testing.T) refClient {
	m := secretrefmocks.NewMockRefClient(t)
	m.On("MapConfigSecretsRefs", mock.Anything, mock.Anything, mock.Anything, "default").Return(map[string]string{}, nil)

	return m
}

const clientSecretVersionToken = "secret:secret:key@uid-1@100"

func TestPutIDP_Serve(t *testing.T) {
	t.Parallel()

	inSyncHash := secretref.ValuesHashSingle(nil)
	maskedSecretRefHash := secretref.ValuesHashSingle(
		map[string]string{"clientSecret": clientSecretVersionToken},
	)

	tests := []struct {
		name      string
		idp       *keycloakApi.KeycloakRealmIdentityProvider
		idpClient func(t *testing.T) keycloakapi.IdentityProvidersClient
		secretRef func(t *testing.T) refClient
		wantErr   require.ErrorAssertionFunc
		check     func(t *testing.T, idp *keycloakApi.KeycloakRealmIdentityProvider)
	}{
		{
			name: "create new identity provider",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:       "test-idp",
					ProviderID:  "github",
					Enabled:     true,
					HideOnLogin: ptr.To(true),
					Config: map[string]string{
						"clientId":     "test-client",
						"clientSecret": "$secret-name:secret-key",
					},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIdentityProvider", mock.Anything, "realm", "test-idp").
					Return((*keycloakapi.IdentityProviderRepresentation)(nil), (*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: 404}).Once()
				m.On("CreateIdentityProvider", mock.Anything, "realm", mock.MatchedBy(func(rep keycloakapi.IdentityProviderRepresentation) bool {
					return rep.HideOnLogin != nil && *rep.HideOnLogin == true
				})).Return((*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			secretRef: func(t *testing.T) refClient {
				m := secretrefmocks.NewMockRefClient(t)
				m.On("MapConfigSecretsRefs", mock.Anything, mock.Anything, mock.Anything, "default").
					Return(map[string]string{"clientSecret": "secret:secret-name:secret-key@uid-1@100"}, nil)
				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "differs from spec - update fires",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:      "test-idp",
					ProviderID: "github",
					Enabled:    true,
					Config:     map[string]string{"clientId": "test-client"},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIdentityProvider", mock.Anything, "realm", "test-idp").
					Return(&keycloakapi.IdentityProviderRepresentation{Alias: ptr.To("test-idp")}, (*keycloakapi.Response)(nil), nil).Once()
				m.On("UpdateIdentityProvider", mock.Anything, "realm", "test-idp", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			secretRef: noSecretRefMock,
			wantErr:   require.NoError,
		},
		{
			name: "secret ref mapping fails",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:      "test-idp",
					ProviderID: "github",
					Config:     map[string]string{"clientSecret": "$secret:key"},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				return keycloakapimocks.NewMockIdentityProvidersClient(t)
			},
			secretRef: func(t *testing.T) refClient {
				m := secretrefmocks.NewMockRefClient(t)
				m.On("MapConfigSecretsRefs", mock.Anything, mock.Anything, mock.Anything, "default").
					Return(nil, fmt.Errorf("secret not found"))
				return m
			},
			wantErr: require.Error,
		},
		{
			name: "already in sync - no update",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Generation: 3},
				Spec:       inSyncIDPSpec(),
				Status: keycloakApi.KeycloakRealmIdentityProviderStatus{
					ObservedGeneration: 3,
					ConfigSecretsHash:  inSyncHash,
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIdentityProvider", mock.Anything, "realm", "test-idp").
					Return(inSyncIDPRepresentation(), (*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			secretRef: noSecretRefMock,
			wantErr:   require.NoError,
			check: func(t *testing.T, idp *keycloakApi.KeycloakRealmIdentityProvider) {
				require.Equal(t, inSyncHash, idp.Status.ConfigSecretsHash)
			},
		},
		{
			name: "masked secret config key is skipped from comparison - no update",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Generation: 3},
				Spec: func() keycloakApi.KeycloakRealmIdentityProviderSpec {
					spec := inSyncIDPSpec()
					spec.Config = map[string]string{
						"clientId":     "test-client",
						"clientSecret": "$secret:key",
					}
					return spec
				}(),
				Status: keycloakApi.KeycloakRealmIdentityProviderStatus{
					ObservedGeneration: 3,
					ConfigSecretsHash:  maskedSecretRefHash,
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				existing := inSyncIDPRepresentation()
				existing.Config = &map[string]string{
					"clientId":     "test-client",
					"clientSecret": "**********",
				}

				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIdentityProvider", mock.Anything, "realm", "test-idp").
					Return(existing, (*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			secretRef: func(t *testing.T) refClient {
				m := secretrefmocks.NewMockRefClient(t)
				m.On("MapConfigSecretsRefs", mock.Anything, mock.Anything, mock.Anything, "default").
					Run(func(args mock.Arguments) {
						cfg, _ := args.Get(2).(map[string]string)
						cfg["clientSecret"] = "resolved-secret-value"
					}).
					Return(map[string]string{"clientSecret": clientSecretVersionToken}, nil)
				return m
			},
			wantErr: require.NoError,
			check: func(t *testing.T, idp *keycloakApi.KeycloakRealmIdentityProvider) {
				require.Equal(t, maskedSecretRefHash, idp.Status.ConfigSecretsHash)
			},
		},
		{
			name: "plain literal secret masked on GET is skipped from comparison - no update",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Generation: 3},
				Spec: func() keycloakApi.KeycloakRealmIdentityProviderSpec {
					spec := inSyncIDPSpec()
					spec.Config = map[string]string{
						"clientId":     "test-client",
						"clientSecret": "supersecret123",
					}
					return spec
				}(),
				Status: keycloakApi.KeycloakRealmIdentityProviderStatus{
					ObservedGeneration: 3,
					ConfigSecretsHash:  inSyncHash,
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				existing := inSyncIDPRepresentation()
				existing.Config = &map[string]string{
					"clientId":     "test-client",
					"clientSecret": keycloakapi.MaskedSecretValue,
				}

				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIdentityProvider", mock.Anything, "realm", "test-idp").
					Return(existing, (*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			secretRef: noSecretRefMock,
			wantErr:   require.NoError,
		},
		{
			name: "config secrets hash drift forces update",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Generation: 3},
				Spec:       inSyncIDPSpec(),
				Status: keycloakApi.KeycloakRealmIdentityProviderStatus{
					ObservedGeneration: 3,
					ConfigSecretsHash:  "stale-hash",
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIdentityProvider", mock.Anything, "realm", "test-idp").
					Return(inSyncIDPRepresentation(), (*keycloakapi.Response)(nil), nil).Once()
				m.On("UpdateIdentityProvider", mock.Anything, "realm", "test-idp", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			secretRef: noSecretRefMock,
			wantErr:   require.NoError,
			check: func(t *testing.T, idp *keycloakApi.KeycloakRealmIdentityProvider) {
				require.NotEqual(t, "stale-hash", idp.Status.ConfigSecretsHash)
			},
		},
		{
			name: "generation bump forces update",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Generation: 4},
				Spec:       inSyncIDPSpec(),
				Status: keycloakApi.KeycloakRealmIdentityProviderStatus{
					ObservedGeneration: 3,
					ConfigSecretsHash:  inSyncHash,
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIdentityProvider", mock.Anything, "realm", "test-idp").
					Return(inSyncIDPRepresentation(), (*keycloakapi.Response)(nil), nil).Once()
				m.On("UpdateIdentityProvider", mock.Anything, "realm", "test-idp", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			secretRef: noSecretRefMock,
			wantErr:   require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := NewPutIDP(tt.idpClient(t), tt.secretRef(t))
			err := h.Serve(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.idp,
				"realm",
			)
			tt.wantErr(t, err)

			if tt.check != nil {
				tt.check(t, tt.idp)
			}
		})
	}
}
