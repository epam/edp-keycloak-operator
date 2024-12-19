package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	keycloakmocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
	"github.com/epam/edp-keycloak-operator/pkg/secretref/mocks"
)

func TestPutClient_Serve(t *testing.T) {
	t.Parallel()

	type fields struct {
		client    func(t *testing.T) client.Client
		secretRef func(t *testing.T) secretRef
	}

	type args struct {
		keycloakClient client.ObjectKey
		adapterClient  func(t *testing.T) keycloak.Client
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "create client with secret ref",
			fields: fields{
				client: func(t *testing.T) client.Client {
					s := runtime.NewScheme()
					require.NoError(t, keycloakApi.AddToScheme(s))
					require.NoError(t, corev1.AddToScheme(s))

					return fake.NewClientBuilder().
						WithScheme(s).
						WithObjects(
							&keycloakApi.KeycloakClient{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-client",
									Namespace: "default",
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: "test-client-id",
									Secret:   secretref.GenerateSecretRef("client-secret", "secret"),
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					m := mocks.NewRefClient(t)

					m.On("GetSecretFromRef", testifymock.Anything, testifymock.Anything, "default").
						Return("client-secret", nil)

					return m
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				adapterClient: func(t *testing.T) keycloak.Client {
					m := keycloakmocks.NewMockClient(t)

					m.On("GetClientID", "test-client-id", "realm").
						Return("", adapter.NotFoundError("not found")).
						Once()

					m.On("CreateClient", testifymock.Anything, testifymock.Anything).
						Return(nil)

					m.On("GetClientID", "test-client-id", "realm").
						Return("123", nil)

					return m
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "create client with old secret format",
			fields: fields{
				client: func(t *testing.T) client.Client {
					s := runtime.NewScheme()
					require.NoError(t, keycloakApi.AddToScheme(s))
					require.NoError(t, corev1.AddToScheme(s))

					return fake.NewClientBuilder().
						WithScheme(s).
						WithObjects(
							&keycloakApi.KeycloakClient{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-client",
									Namespace: "default",
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: "test-client-id",
									Secret:   "client-secret",
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					m := mocks.NewRefClient(t)

					m.On("GetSecretFromRef", testifymock.Anything, testifymock.Anything, "default").
						Return("client-secret", nil)

					return m
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				adapterClient: func(t *testing.T) keycloak.Client {
					m := keycloakmocks.NewMockClient(t)

					m.On("GetClientID", "test-client-id", "realm").
						Return("", adapter.NotFoundError("not found")).
						Once()

					m.On("CreateClient", testifymock.Anything, testifymock.Anything).
						Return(nil)

					m.On("GetClientID", "test-client-id", "realm").
						Return("123", nil)

					return m
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "create client with empty secret ref",
			fields: fields{
				client: func(t *testing.T) client.Client {
					s := runtime.NewScheme()
					require.NoError(t, keycloakApi.AddToScheme(s))
					require.NoError(t, corev1.AddToScheme(s))

					return fake.NewClientBuilder().
						WithScheme(s).
						WithObjects(
							&keycloakApi.KeycloakClient{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-client",
									Namespace: "default",
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: "test-client-id",
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				adapterClient: func(t *testing.T) keycloak.Client {
					m := keycloakmocks.NewMockClient(t)

					m.On("GetClientID", "test-client-id", "realm").
						Return("", adapter.NotFoundError("not found")).
						Once()

					m.On("CreateClient", testifymock.Anything, testifymock.Anything).
						Return(nil)

					m.On("GetClientID", "test-client-id", "realm").
						Return("123", nil)

					return m
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "update client with secret ref",
			fields: fields{
				client: func(t *testing.T) client.Client {
					s := runtime.NewScheme()
					require.NoError(t, keycloakApi.AddToScheme(s))
					require.NoError(t, corev1.AddToScheme(s))

					return fake.NewClientBuilder().
						WithScheme(s).
						WithObjects(
							&keycloakApi.KeycloakClient{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-client",
									Namespace: "default",
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: "test-client-id",
									Secret:   secretref.GenerateSecretRef("client-secret", "secret"),
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					m := mocks.NewRefClient(t)

					m.On("GetSecretFromRef", testifymock.Anything, testifymock.Anything, "default").
						Return("client-secret", nil)

					return m
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				adapterClient: func(t *testing.T) keycloak.Client {
					m := keycloakmocks.NewMockClient(t)

					m.On("GetClientID", "test-client-id", "realm").
						Return("123", nil).
						Once()

					m.On("UpdateClient", testifymock.Anything, testifymock.Anything).
						Return(nil)

					m.On("GetClientID", "test-client-id", "realm").
						Return("123", nil)

					return m
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "create public client",
			fields: fields{
				client: func(t *testing.T) client.Client {
					s := runtime.NewScheme()
					require.NoError(t, keycloakApi.AddToScheme(s))
					require.NoError(t, corev1.AddToScheme(s))

					return fake.NewClientBuilder().
						WithScheme(s).
						WithObjects(
							&keycloakApi.KeycloakClient{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-client",
									Namespace: "default",
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: "test-client-id",
									Public:   true,
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				adapterClient: func(t *testing.T) keycloak.Client {
					m := keycloakmocks.NewMockClient(t)

					m.On("GetClientID", "test-client-id", "realm").
						Return("", adapter.NotFoundError("not found")).
						Once()

					m.On("CreateClient", testifymock.Anything, testifymock.Anything).
						Return(nil)

					m.On("GetClientID", "test-client-id", "realm").
						Return("123", nil)

					return m
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "create client with auth flows",
			fields: fields{
				client: func(t *testing.T) client.Client {
					s := runtime.NewScheme()
					require.NoError(t, keycloakApi.AddToScheme(s))
					require.NoError(t, corev1.AddToScheme(s))

					return fake.NewClientBuilder().
						WithScheme(s).
						WithObjects(
							&keycloakApi.KeycloakClient{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-client",
									Namespace: "default",
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: "test-client-id",
									AuthenticationFlowBindingOverrides: &keycloakApi.AuthenticationFlowBindingOverrides{
										Browser:     "browser",
										DirectGrant: "direct grant",
									},
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				adapterClient: func(t *testing.T) keycloak.Client {
					m := keycloakmocks.NewMockClient(t)

					m.On("GetClientID", "test-client-id", "realm").
						Return("", adapter.NotFoundError("not found")).
						Once()

					m.On("CreateClient", testifymock.Anything, testifymock.Anything).
						Return(nil)

					m.On("GetClientID", "test-client-id", "realm").
						Return("123", nil)
					m.On("GetRealmAuthFlows", "realm").
						Return([]adapter.KeycloakAuthFlow{
							{
								ID:    "A39C9C6E-430A-4891-8592-FC195AF2581B",
								Alias: "browser",
							},
							{
								ID:    "8BF514C6-922C-44A0-8F90-488D1B9DE437",
								Alias: "direct grant",
							},
						}, nil)

					return m
				},
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := runtime.NewScheme()
			require.NoError(t, keycloakApi.AddToScheme(s))
			require.NoError(t, corev1.AddToScheme(s))

			cl := &keycloakApi.KeycloakClient{}
			require.NoError(t, tt.fields.client(t).Get(context.Background(), tt.args.keycloakClient, cl))

			el := NewPutClient(tt.args.adapterClient(t), tt.fields.client(t), tt.fields.secretRef(t))
			err := el.Serve(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				cl,
				"realm",
			)
			tt.wantErr(t, err)
		})
	}
}
