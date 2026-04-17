package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakv2Mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestPutProtocolMappers_Serve(t *testing.T) {
	const (
		testClientName      = "test-client"
		testClientNamespace = "default"
	)

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	type args struct {
		keycloakClient *keycloakApi.KeycloakClient
		realmName      string
		adapterClient  func(t *testing.T) *keycloakapi.APIClient
	}

	tests := []struct {
		name    string
		args    args
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success with protocol mappers",
			args: args{
				keycloakClient: &keycloakApi.KeycloakClient{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-client",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakClientSpec{
						ClientId: "test-client-id",
						RealmRef: common.RealmRef{Name: "test-realm"},
						ProtocolMappers: &[]keycloakApi.ProtocolMapper{
							{
								Name:           "test-mapper",
								Protocol:       "openid-connect",
								ProtocolMapper: "oidc-usermodel-property-mapper",
								Config: map[string]string{
									"user.attribute":     "username",
									"claim.name":         "preferred_username",
									"jsonType.label":     "String",
									"id.token.claim":     "true",
									"access.token.claim": "true",
								},
							},
						},
					},
				},
				realmName: "test-realm",
				adapterClient: func(t *testing.T) *keycloakapi.APIClient {
					clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
					clientsMock.On("GetClientProtocolMappers", mock.Anything, "test-realm", "client-uuid").
						Return([]keycloakapi.ProtocolMapperRepresentation{}, (*keycloakapi.Response)(nil), nil)
					clientsMock.On("CreateClientProtocolMapper", mock.Anything, "test-realm", "client-uuid", mock.Anything).
						Return((*keycloakapi.Response)(nil), nil)

					return &keycloakapi.APIClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "success with multiple protocol mappers",
			args: args{
				keycloakClient: &keycloakApi.KeycloakClient{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-client",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakClientSpec{
						ClientId: "test-client-id",
						RealmRef: common.RealmRef{Name: "test-realm"},
						ProtocolMappers: &[]keycloakApi.ProtocolMapper{
							{
								Name:           "username-mapper",
								Protocol:       "openid-connect",
								ProtocolMapper: "oidc-usermodel-property-mapper",
								Config: map[string]string{
									"user.attribute": "username",
									"claim.name":     "preferred_username",
								},
							},
							{
								Name:           "role-mapper",
								Protocol:       "openid-connect",
								ProtocolMapper: "oidc-usermodel-realm-role-mapper",
								Config: map[string]string{
									"claim.name":         "roles",
									"access.token.claim": "true",
								},
							},
						},
					},
				},
				realmName: "test-realm",
				adapterClient: func(t *testing.T) *keycloakapi.APIClient {
					clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
					clientsMock.On("GetClientProtocolMappers", mock.Anything, "test-realm", "client-uuid").
						Return([]keycloakapi.ProtocolMapperRepresentation{}, (*keycloakapi.Response)(nil), nil)
					clientsMock.On("CreateClientProtocolMapper", mock.Anything, "test-realm", "client-uuid", mock.MatchedBy(func(m keycloakapi.ProtocolMapperRepresentation) bool {
						return m.Name != nil && *m.Name == "username-mapper"
					})).Return((*keycloakapi.Response)(nil), nil)
					clientsMock.On("CreateClientProtocolMapper", mock.Anything, "test-realm", "client-uuid", mock.MatchedBy(func(m keycloakapi.ProtocolMapperRepresentation) bool {
						return m.Name != nil && *m.Name == "role-mapper"
					})).Return((*keycloakapi.Response)(nil), nil)

					return &keycloakapi.APIClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "success with add-only reconciliation strategy",
			args: args{
				keycloakClient: &keycloakApi.KeycloakClient{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-client",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakClientSpec{
						ClientId:               "test-client-id",
						RealmRef:               common.RealmRef{Name: "test-realm"},
						ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
						ProtocolMappers: &[]keycloakApi.ProtocolMapper{
							{
								Name:           "test-mapper",
								Protocol:       "openid-connect",
								ProtocolMapper: "oidc-usermodel-property-mapper",
								Config: map[string]string{
									"user.attribute": "username",
								},
							},
						},
					},
				},
				realmName: "test-realm",
				adapterClient: func(t *testing.T) *keycloakapi.APIClient {
					clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
					clientsMock.On("GetClientProtocolMappers", mock.Anything, "test-realm", "client-uuid").
						Return([]keycloakapi.ProtocolMapperRepresentation{}, (*keycloakapi.Response)(nil), nil)
					clientsMock.On("CreateClientProtocolMapper", mock.Anything, "test-realm", "client-uuid", mock.Anything).
						Return((*keycloakapi.Response)(nil), nil)
					// No DeleteClientProtocolMapper call expected (addOnly)

					return &keycloakapi.APIClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "success with nil protocol mappers",
			args: args{
				keycloakClient: &keycloakApi.KeycloakClient{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-client",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakClientSpec{
						ClientId:        "test-client-id",
						RealmRef:        common.RealmRef{Name: "test-realm"},
						ProtocolMappers: nil, // explicitly nil
					},
				},
				realmName: "test-realm",
				adapterClient: func(t *testing.T) *keycloakapi.APIClient {
					clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
					clientsMock.On("GetClientProtocolMappers", mock.Anything, "test-realm", "client-uuid").
						Return([]keycloakapi.ProtocolMapperRepresentation{}, (*keycloakapi.Response)(nil), nil)

					return &keycloakapi.APIClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "success with empty protocol mappers slice",
			args: args{
				keycloakClient: &keycloakApi.KeycloakClient{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-client",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakClientSpec{
						ClientId:        "test-client-id",
						RealmRef:        common.RealmRef{Name: "test-realm"},
						ProtocolMappers: &[]keycloakApi.ProtocolMapper{}, // empty slice
					},
				},
				realmName: "test-realm",
				adapterClient: func(t *testing.T) *keycloakapi.APIClient {
					clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
					clientsMock.On("GetClientProtocolMappers", mock.Anything, "test-realm", "client-uuid").
						Return([]keycloakapi.ProtocolMapperRepresentation{}, (*keycloakapi.Response)(nil), nil)

					return &keycloakapi.APIClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "success with protocol mapper with empty config",
			args: args{
				keycloakClient: &keycloakApi.KeycloakClient{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-client",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakClientSpec{
						ClientId: "test-client-id",
						RealmRef: common.RealmRef{Name: "test-realm"},
						ProtocolMappers: &[]keycloakApi.ProtocolMapper{
							{
								Name:           "empty-config-mapper",
								Protocol:       "openid-connect",
								ProtocolMapper: "oidc-audience-mapper",
								Config:         map[string]string{}, // empty config
							},
						},
					},
				},
				realmName: "test-realm",
				adapterClient: func(t *testing.T) *keycloakapi.APIClient {
					clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
					clientsMock.On("GetClientProtocolMappers", mock.Anything, "test-realm", "client-uuid").
						Return([]keycloakapi.ProtocolMapperRepresentation{}, (*keycloakapi.Response)(nil), nil)
					clientsMock.On("CreateClientProtocolMapper", mock.Anything, "test-realm", "client-uuid", mock.Anything).
						Return((*keycloakapi.Response)(nil), nil)

					return &keycloakapi.APIClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "success with protocol mapper with nil config",
			args: args{
				keycloakClient: &keycloakApi.KeycloakClient{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-client",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakClientSpec{
						ClientId: "test-client-id",
						RealmRef: common.RealmRef{Name: "test-realm"},
						ProtocolMappers: &[]keycloakApi.ProtocolMapper{
							{
								Name:           "nil-config-mapper",
								Protocol:       "openid-connect",
								ProtocolMapper: "oidc-audience-mapper",
								Config:         nil, // nil config
							},
						},
					},
				},
				realmName: "test-realm",
				adapterClient: func(t *testing.T) *keycloakapi.APIClient {
					clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
					clientsMock.On("GetClientProtocolMappers", mock.Anything, "test-realm", "client-uuid").
						Return([]keycloakapi.ProtocolMapperRepresentation{}, (*keycloakapi.Response)(nil), nil)
					clientsMock.On("CreateClientProtocolMapper", mock.Anything, "test-realm", "client-uuid", mock.Anything).
						Return((*keycloakapi.Response)(nil), nil)

					return &keycloakapi.APIClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "error when GetClientProtocolMappers fails",
			args: args{
				keycloakClient: &keycloakApi.KeycloakClient{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-client",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakClientSpec{
						ClientId: "test-client-id",
						RealmRef: common.RealmRef{Name: "test-realm"},
					},
				},
				realmName: "test-realm",
				adapterClient: func(t *testing.T) *keycloakapi.APIClient {
					clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
					clientsMock.On("GetClientProtocolMappers", mock.Anything, "test-realm", "client-uuid").
						Return(nil, (*keycloakapi.Response)(nil), errors.New("api error"))

					return &keycloakapi.APIClient{Clients: clientsMock}
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get existing protocol mappers")
			},
		},
		{
			name: "error when UpdateClientProtocolMapper fails",
			args: args{
				keycloakClient: &keycloakApi.KeycloakClient{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-client",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakClientSpec{
						ClientId: "test-client-id",
						RealmRef: common.RealmRef{Name: "test-realm"},
						ProtocolMappers: &[]keycloakApi.ProtocolMapper{
							{
								Name:           "test-mapper",
								Protocol:       "openid-connect",
								ProtocolMapper: "oidc-usermodel-property-mapper",
								Config:         map[string]string{"key": "val"},
							},
						},
					},
				},
				realmName: "test-realm",
				adapterClient: func(t *testing.T) *keycloakapi.APIClient {
					clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
					clientsMock.On("GetClientProtocolMappers", mock.Anything, "test-realm", "client-uuid").
						Return([]keycloakapi.ProtocolMapperRepresentation{
							{Name: ptr.To("test-mapper"), Id: ptr.To("mapper-id")},
						}, (*keycloakapi.Response)(nil), nil)
					clientsMock.On("UpdateClientProtocolMapper", mock.Anything, "test-realm", "client-uuid", "mapper-id", mock.Anything).
						Return((*keycloakapi.Response)(nil), errors.New("update failed"))

					return &keycloakapi.APIClient{Clients: clientsMock}
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to update protocol mapper test-mapper")
			},
		},
		{
			name: "error when DeleteClientProtocolMapper returns non-404 error",
			args: args{
				keycloakClient: &keycloakApi.KeycloakClient{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-client",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakClientSpec{
						ClientId:        "test-client-id",
						RealmRef:        common.RealmRef{Name: "test-realm"},
						ProtocolMappers: &[]keycloakApi.ProtocolMapper{},
					},
				},
				realmName: "test-realm",
				adapterClient: func(t *testing.T) *keycloakapi.APIClient {
					clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
					clientsMock.On("GetClientProtocolMappers", mock.Anything, "test-realm", "client-uuid").
						Return([]keycloakapi.ProtocolMapperRepresentation{
							{Name: ptr.To("stale-mapper"), Id: ptr.To("stale-id")},
						}, (*keycloakapi.Response)(nil), nil)
					clientsMock.On("DeleteClientProtocolMapper", mock.Anything, "test-realm", "client-uuid", "stale-id").
						Return((*keycloakapi.Response)(nil), errors.New("server error"))

					return &keycloakapi.APIClient{Clients: clientsMock}
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to delete protocol mapper stale-mapper")
			},
		},
		{
			name: "existing mapper with nil Id skips update",
			args: args{
				keycloakClient: &keycloakApi.KeycloakClient{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-client",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakClientSpec{
						ClientId: "test-client-id",
						RealmRef: common.RealmRef{Name: "test-realm"},
						ProtocolMappers: &[]keycloakApi.ProtocolMapper{
							{
								Name:           "test-mapper",
								Protocol:       "openid-connect",
								ProtocolMapper: "oidc-usermodel-property-mapper",
								Config:         map[string]string{"key": "val"},
							},
						},
					},
				},
				realmName: "test-realm",
				adapterClient: func(t *testing.T) *keycloakapi.APIClient {
					clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
					clientsMock.On("GetClientProtocolMappers", mock.Anything, "test-realm", "client-uuid").
						Return([]keycloakapi.ProtocolMapperRepresentation{
							{Name: ptr.To("test-mapper"), Id: nil},
						}, (*keycloakapi.Response)(nil), nil)

					return &keycloakapi.APIClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "delete skips mapper with nil Id",
			args: args{
				keycloakClient: &keycloakApi.KeycloakClient{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-client",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakClientSpec{
						ClientId:        "test-client-id",
						RealmRef:        common.RealmRef{Name: "test-realm"},
						ProtocolMappers: &[]keycloakApi.ProtocolMapper{},
					},
				},
				realmName: "test-realm",
				adapterClient: func(t *testing.T) *keycloakapi.APIClient {
					clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
					clientsMock.On("GetClientProtocolMappers", mock.Anything, "test-realm", "client-uuid").
						Return([]keycloakapi.ProtocolMapperRepresentation{
							{Name: ptr.To("orphan-mapper"), Id: nil},
						}, (*keycloakapi.Response)(nil), nil)

					return &keycloakapi.APIClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "error when CreateClientProtocolMapper fails",
			args: args{
				keycloakClient: &keycloakApi.KeycloakClient{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-client",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakClientSpec{
						ClientId: "test-client-id",
						RealmRef: common.RealmRef{Name: "test-realm"},
						ProtocolMappers: &[]keycloakApi.ProtocolMapper{
							{
								Name:           "test-mapper",
								Protocol:       "openid-connect",
								ProtocolMapper: "oidc-usermodel-property-mapper",
								Config: map[string]string{
									"user.attribute": "username",
								},
							},
						},
					},
				},
				realmName: "test-realm",
				adapterClient: func(t *testing.T) *keycloakapi.APIClient {
					clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
					clientsMock.On("GetClientProtocolMappers", mock.Anything, "test-realm", "client-uuid").
						Return([]keycloakapi.ProtocolMapperRepresentation{}, (*keycloakapi.Response)(nil), nil)
					clientsMock.On("CreateClientProtocolMapper", mock.Anything, "test-realm", "client-uuid", mock.Anything).
						Return((*keycloakapi.Response)(nil), errors.New("creation failed"))

					return &keycloakapi.APIClient{Clients: clientsMock}
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to put protocol mappers")
				require.Contains(t, err.Error(), "unable to create protocol mapper")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure test client has proper metadata
			if tt.args.keycloakClient.Name == "" {
				tt.args.keycloakClient.Name = testClientName
			}

			if tt.args.keycloakClient.Namespace == "" {
				tt.args.keycloakClient.Namespace = testClientNamespace
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.args.keycloakClient).
				WithStatusSubresource(tt.args.keycloakClient).
				Build()

			el := NewPutProtocolMappers(tt.args.adapterClient(t), k8sClient)
			err := el.Serve(context.Background(), tt.args.keycloakClient, tt.args.realmName, &ClientContext{ClientUUID: "client-uuid"})
			tt.wantErr(t, err)
		})
	}
}
