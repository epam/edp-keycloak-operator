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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestPutClientScope_Serve(t *testing.T) {
	tests := []struct {
		name              string
		client            func(t *testing.T) client.Client
		keycloakClient    client.ObjectKey
		keycloakApiClient func(t *testing.T) *mocks.MockClient
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
				m.On("AddDefaultScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

				return m
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
				m.On("AddOptionalScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

				return m
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, []string{"default-scope-1", "default-scope-2"}).Return(nil, nil)
				m.On("AddDefaultScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, []string{"optional-scope-1", "optional-scope-2"}).Return(nil, nil)
				m.On("AddOptionalScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

				return m
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "error when GetClientScopesByNames fails for default scopes",
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("failed to get client scopes"))

				return m
			},
			wantErr: require.Error,
			wantCondition: &metav1.Condition{
				Type:   ConditionClientScopesSynced,
				Status: metav1.ConditionFalse,
				Reason: ReasonKeycloakAPIError,
			},
		},
		{
			name: "error when GetClientScopesByNames fails for optional scopes",
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("failed to get client scopes"))

				return m
			},
			wantErr: require.Error,
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
				m.On("AddDefaultScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("failed to add default scope"))

				return m
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
				m.On("AddOptionalScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("failed to add optional scope"))

				return m
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, []string{"default-scope"}).Return(nil, nil)
				m.On("AddDefaultScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, []string{"optional-scope"}).Return(nil, nil)
				m.On("AddOptionalScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("failed to add optional scope"))

				return m
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				return m
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
