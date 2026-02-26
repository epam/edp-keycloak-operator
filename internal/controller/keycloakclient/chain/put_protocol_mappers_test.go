package chain

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
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
		adapterClient  func(t *testing.T) *mocks.MockClient
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
				adapterClient: func(t *testing.T) *mocks.MockClient {
					m := mocks.NewMockClient(t)
					m.On("SyncClientProtocolMapper",
						mock.Anything,
						mockMatchProtocolMappers(1),
						false, // not add-only (full reconciliation)
					).Return(nil)
					return m
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
				adapterClient: func(t *testing.T) *mocks.MockClient {
					m := mocks.NewMockClient(t)
					m.On("SyncClientProtocolMapper",
						mock.Anything,
						mockMatchProtocolMappers(2),
						false,
					).Return(nil)
					return m
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
				adapterClient: func(t *testing.T) *mocks.MockClient {
					m := mocks.NewMockClient(t)
					m.On("SyncClientProtocolMapper",
						mock.Anything,
						mockMatchProtocolMappers(1),
						true, // add-only reconciliation
					).Return(nil)
					return m
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
				adapterClient: func(t *testing.T) *mocks.MockClient {
					m := mocks.NewMockClient(t)
					m.On("SyncClientProtocolMapper",
						mock.Anything,
						mockMatchProtocolMappers(0), // empty slice
						false,
					).Return(nil)
					return m
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
				adapterClient: func(t *testing.T) *mocks.MockClient {
					m := mocks.NewMockClient(t)
					m.On("SyncClientProtocolMapper",
						mock.Anything,
						mockMatchProtocolMappers(0),
						false,
					).Return(nil)
					return m
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
				adapterClient: func(t *testing.T) *mocks.MockClient {
					m := mocks.NewMockClient(t)
					m.On("SyncClientProtocolMapper",
						mock.Anything,
						mockMatchProtocolMappers(1),
						false,
					).Return(nil)
					return m
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
				adapterClient: func(t *testing.T) *mocks.MockClient {
					m := mocks.NewMockClient(t)
					m.On("SyncClientProtocolMapper",
						mock.Anything,
						mockMatchProtocolMappers(1),
						false,
					).Return(nil)
					return m
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "error when SyncClientProtocolMapper fails",
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
				adapterClient: func(t *testing.T) *mocks.MockClient {
					m := mocks.NewMockClient(t)
					m.On("SyncClientProtocolMapper",
						mock.Anything,
						mockMatchProtocolMappers(1),
						false,
					).Return(errors.New("sync failed"))
					return m
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to put protocol mappers")
				require.Contains(t, err.Error(), "unable to sync protocol mapper")
				require.Contains(t, err.Error(), "sync failed")
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
			err := el.Serve(context.Background(), tt.args.keycloakClient, tt.args.realmName)
			tt.wantErr(t, err)
		})
	}
}

// mockMatchProtocolMappers is a helper function to match protocol mappers slice in mock calls
func mockMatchProtocolMappers(expectedCount int) any {
	return mock.MatchedBy(func(mappers any) bool {
		if mappers == nil {
			return expectedCount == 0
		}

		// Use reflection to check slice length without importing gocloak
		if reflect.TypeOf(mappers).Kind() == reflect.Slice {
			sliceValue := reflect.ValueOf(mappers)
			return sliceValue.Len() == expectedCount
		}

		return false
	})
}
