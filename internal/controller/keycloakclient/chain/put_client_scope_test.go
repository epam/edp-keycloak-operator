package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapiMocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestPutClientScope_Serve(t *testing.T) {
	tests := []struct {
		name              string
		client            func(t *testing.T) client.Client
		keycloakClient    client.ObjectKey
		keycloakApiClient func(t *testing.T) *keycloakapi.KeycloakClient
		wantErr           require.ErrorAssertionFunc
		wantCondition     *metav1.Condition
	}{
		{
			name: "with default scopes",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApi.KeycloakClient{}).
					WithObjects(
						&keycloakApi.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId:            "test-client-id",
								DefaultClientScopes: []string{"default-scope"},
							},
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)

				clientsMock.On("GetRealmClientScopes", mock.Anything, "realm").
					Return([]keycloakapi.ClientScopeRepresentation{
						{Id: ptr.To("default-scope-id"), Name: ptr.To("default-scope")},
					}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("GetDefaultClientScopes", mock.Anything, "realm", "client-uuid").
					Return([]keycloakapi.ClientScopeRepresentation{}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("AddDefaultClientScope", mock.Anything, "realm", "client-uuid", "default-scope-id").
					Return((*keycloakapi.Response)(nil), nil)

				return &keycloakapi.KeycloakClient{Clients: clientsMock}
			},
			wantErr: require.NoError,
			wantCondition: &metav1.Condition{
				Type:   ConditionClientScopesSynced,
				Status: metav1.ConditionTrue,
				Reason: ReasonClientScopesSynced,
			},
		},
		{
			name: "with optional scopes",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApi.KeycloakClient{}).
					WithObjects(
						&keycloakApi.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId:             "test-client-id",
								OptionalClientScopes: []string{"optional-scope"},
							},
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)

				clientsMock.On("GetRealmClientScopes", mock.Anything, "realm").
					Return([]keycloakapi.ClientScopeRepresentation{
						{Id: ptr.To("optional-scope-id"), Name: ptr.To("optional-scope")},
					}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("GetOptionalClientScopes", mock.Anything, "realm", "client-uuid").
					Return([]keycloakapi.ClientScopeRepresentation{}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("AddOptionalClientScope", mock.Anything, "realm", "client-uuid", "optional-scope-id").
					Return((*keycloakapi.Response)(nil), nil)

				return &keycloakapi.KeycloakClient{Clients: clientsMock}
			},
			wantErr: require.NoError,
		},
		{
			name: "with both default and optional scopes",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApi.KeycloakClient{}).
					WithObjects(
						&keycloakApi.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId:             "test-client-id",
								DefaultClientScopes:  []string{"default-scope-1", "default-scope-2"},
								OptionalClientScopes: []string{"optional-scope-1", "optional-scope-2"},
							},
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)

				// Default scopes flow
				clientsMock.On("GetRealmClientScopes", mock.Anything, "realm").
					Return([]keycloakapi.ClientScopeRepresentation{
						{Id: ptr.To("default-scope-1-id"), Name: ptr.To("default-scope-1")},
						{Id: ptr.To("default-scope-2-id"), Name: ptr.To("default-scope-2")},
						{Id: ptr.To("optional-scope-1-id"), Name: ptr.To("optional-scope-1")},
						{Id: ptr.To("optional-scope-2-id"), Name: ptr.To("optional-scope-2")},
					}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("GetDefaultClientScopes", mock.Anything, "realm", "client-uuid").
					Return([]keycloakapi.ClientScopeRepresentation{}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("AddDefaultClientScope", mock.Anything, "realm", "client-uuid", "default-scope-1-id").
					Return((*keycloakapi.Response)(nil), nil)
				clientsMock.On("AddDefaultClientScope", mock.Anything, "realm", "client-uuid", "default-scope-2-id").
					Return((*keycloakapi.Response)(nil), nil)

				// Optional scopes flow
				clientsMock.On("GetOptionalClientScopes", mock.Anything, "realm", "client-uuid").
					Return([]keycloakapi.ClientScopeRepresentation{}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("AddOptionalClientScope", mock.Anything, "realm", "client-uuid", "optional-scope-1-id").
					Return((*keycloakapi.Response)(nil), nil)
				clientsMock.On("AddOptionalClientScope", mock.Anything, "realm", "client-uuid", "optional-scope-2-id").
					Return((*keycloakapi.Response)(nil), nil)

				return &keycloakapi.KeycloakClient{Clients: clientsMock}
			},
			wantErr: require.NoError,
		},
		{
			name: "with no scopes",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApi.KeycloakClient{}).
					WithObjects(
						&keycloakApi.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId: "test-client-id",
							},
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				return &keycloakapi.KeycloakClient{}
			},
			wantErr: require.NoError,
		},
		{
			name: "error when AddDefaultScopeToClient fails",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApi.KeycloakClient{}).
					WithObjects(
						&keycloakApi.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId:            "test-client-id",
								DefaultClientScopes: []string{"default-scope"},
							},
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)

				clientsMock.On("GetRealmClientScopes", mock.Anything, "realm").
					Return([]keycloakapi.ClientScopeRepresentation{
						{Id: ptr.To("default-scope-id"), Name: ptr.To("default-scope")},
					}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("GetDefaultClientScopes", mock.Anything, "realm", "client-uuid").
					Return([]keycloakapi.ClientScopeRepresentation{}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("AddDefaultClientScope", mock.Anything, "realm", "client-uuid", "default-scope-id").
					Return((*keycloakapi.Response)(nil), errors.New("failed to add default scope"))

				return &keycloakapi.KeycloakClient{Clients: clientsMock}
			},
			wantErr: require.Error,
		},
		{
			name: "error when AddOptionalScopeToClient fails",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApi.KeycloakClient{}).
					WithObjects(
						&keycloakApi.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId:             "test-client-id",
								OptionalClientScopes: []string{"optional-scope"},
							},
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)

				clientsMock.On("GetRealmClientScopes", mock.Anything, "realm").
					Return([]keycloakapi.ClientScopeRepresentation{
						{Id: ptr.To("optional-scope-id"), Name: ptr.To("optional-scope")},
					}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("GetOptionalClientScopes", mock.Anything, "realm", "client-uuid").
					Return([]keycloakapi.ClientScopeRepresentation{}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("AddOptionalClientScope", mock.Anything, "realm", "client-uuid", "optional-scope-id").
					Return((*keycloakapi.Response)(nil), errors.New("failed to add optional scope"))

				return &keycloakapi.KeycloakClient{Clients: clientsMock}
			},
			wantErr: require.Error,
		},
		{
			name: "error when AddOptionalScopeToClient fails with both default and optional scopes",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApi.KeycloakClient{}).
					WithObjects(
						&keycloakApi.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId:             "test-client-id",
								DefaultClientScopes:  []string{"default-scope"},
								OptionalClientScopes: []string{"optional-scope"},
							},
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)

				// Default scopes flow (succeeds)
				clientsMock.On("GetRealmClientScopes", mock.Anything, "realm").
					Return([]keycloakapi.ClientScopeRepresentation{
						{Id: ptr.To("default-scope-id"), Name: ptr.To("default-scope")},
						{Id: ptr.To("optional-scope-id"), Name: ptr.To("optional-scope")},
					}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("GetDefaultClientScopes", mock.Anything, "realm", "client-uuid").
					Return([]keycloakapi.ClientScopeRepresentation{}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("AddDefaultClientScope", mock.Anything, "realm", "client-uuid", "default-scope-id").
					Return((*keycloakapi.Response)(nil), nil)

				// Optional scopes flow (fails)
				clientsMock.On("GetOptionalClientScopes", mock.Anything, "realm", "client-uuid").
					Return([]keycloakapi.ClientScopeRepresentation{}, (*keycloakapi.Response)(nil), nil)
				clientsMock.On("AddOptionalClientScope", mock.Anything, "realm", "client-uuid", "optional-scope-id").
					Return((*keycloakapi.Response)(nil), errors.New("failed to add optional scope"))

				return &keycloakapi.KeycloakClient{Clients: clientsMock}
			},
			wantErr: require.Error,
		},
		{
			name: "with empty scope arrays",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApi.KeycloakClient{}).
					WithObjects(
						&keycloakApi.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId:             "test-client-id",
								DefaultClientScopes:  []string{},
								OptionalClientScopes: []string{},
							},
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				return &keycloakapi.KeycloakClient{}
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := &keycloakApi.KeycloakClient{}
			require.NoError(t, tt.client(t).Get(context.Background(), tt.keycloakClient, cl))

			el := NewPutClientScope(tt.keycloakApiClient(t), tt.client(t))
			err := el.Serve(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				cl,
				"realm",
				&ClientContext{ClientUUID: "client-uuid"},
			)
			tt.wantErr(t, err)

			// Assert condition is set correctly
			if tt.wantCondition != nil {
				cond := meta.FindStatusCondition(cl.Status.Conditions, tt.wantCondition.Type)
				require.NotNil(t, cond, "condition not found")
				require.Equal(t, tt.wantCondition.Status, cond.Status)
				require.Equal(t, tt.wantCondition.Reason, cond.Reason)
				require.Equal(t, cl.Generation, cond.ObservedGeneration)
			}
		})
	}
}
