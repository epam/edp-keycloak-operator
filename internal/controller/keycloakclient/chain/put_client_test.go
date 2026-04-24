package chain

import (
	"context"
	"net/http"
	"testing"

	"errors"

	"github.com/go-logr/logr"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapiMocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
	"github.com/epam/edp-keycloak-operator/pkg/secretref/mocks"
)

func TestPutClient_Serve(t *testing.T) {
	type fields struct {
		client    func(t *testing.T) client.Client
		secretRef func(t *testing.T) secretRef
	}

	type args struct {
		keycloakClient client.ObjectKey
		kClient        func(t *testing.T) *keycloakapi.KeycloakClient
	}

	tests := []struct {
		name          string
		fields        fields
		args          args
		wantErr       require.ErrorAssertionFunc
		wantCondition *metav1.Condition
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
						WithStatusSubresource(&keycloakApi.KeycloakClient{}).
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
					m := mocks.NewMockRefClient(t)

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
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)

					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
						Once()

					clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
						Return(&keycloakapi.Response{
							HTTPResponse: &http.Response{
								Header: http.Header{"Location": []string{"http://host/admin/realms/realm/clients/123"}},
							},
						}, nil)

					return &keycloakapi.KeycloakClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionTrue,
				Reason: ReasonClientUpdated,
			},
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
						WithStatusSubresource(&keycloakApi.KeycloakClient{}).
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
					m := mocks.NewMockRefClient(t)

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
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)

					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
						Once()

					clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
						Return(&keycloakapi.Response{
							HTTPResponse: &http.Response{
								Header: http.Header{"Location": []string{"http://host/admin/realms/realm/clients/123"}},
							},
						}, nil)

					return &keycloakapi.KeycloakClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionTrue,
				Reason: ReasonClientUpdated,
			},
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
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)

					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
						Once()

					clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
						Return(&keycloakapi.Response{
							HTTPResponse: &http.Response{
								Header: http.Header{"Location": []string{"http://host/admin/realms/realm/clients/123"}},
							},
						}, nil)

					return &keycloakapi.KeycloakClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionTrue,
				Reason: ReasonClientUpdated,
			},
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
						WithStatusSubresource(&keycloakApi.KeycloakClient{}).
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
					m := mocks.NewMockRefClient(t)

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
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)

					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return(&keycloakapi.ClientRepresentation{Id: ptr.To("123")}, (*keycloakapi.Response)(nil), nil)

					clientsMock.On("UpdateClient", testifymock.Anything, "realm", "123", testifymock.Anything).
						Return((*keycloakapi.Response)(nil), nil)

					return &keycloakapi.KeycloakClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionTrue,
				Reason: ReasonClientUpdated,
			},
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
						WithStatusSubresource(&keycloakApi.KeycloakClient{}).
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
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)

					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
						Once()

					clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
						Return(&keycloakapi.Response{
							HTTPResponse: &http.Response{
								Header: http.Header{"Location": []string{"http://host/admin/realms/realm/clients/123"}},
							},
						}, nil)

					return &keycloakapi.KeycloakClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionTrue,
				Reason: ReasonClientUpdated,
			},
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
						WithStatusSubresource(&keycloakApi.KeycloakClient{}).
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
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)
					realmsMock := keycloakapiMocks.NewMockRealmClient(t)

					realmsMock.On("GetAuthenticationFlows", testifymock.Anything, "realm").
						Return([]keycloakapi.AuthenticationFlowRepresentation{
							{
								Id:    ptr.To("A39C9C6E-430A-4891-8592-FC195AF2581B"),
								Alias: ptr.To("browser"),
							},
							{
								Id:    ptr.To("8BF514C6-922C-44A0-8F90-488D1B9DE437"),
								Alias: ptr.To("direct grant"),
							},
						}, (*keycloakapi.Response)(nil), nil)

					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
						Once()

					clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
						Return(&keycloakapi.Response{
							HTTPResponse: &http.Response{
								Header: http.Header{"Location": []string{"http://host/admin/realms/realm/clients/123"}},
							},
						}, nil)

					return &keycloakapi.KeycloakClient{Clients: clientsMock, Realms: realmsMock}
				},
			},
			wantErr: require.NoError,
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionTrue,
				Reason: ReasonClientUpdated,
			},
		},
		{
			name: "error when GetAuthenticationFlows fails",
			fields: fields{
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
									AuthenticationFlowBindingOverrides: &keycloakApi.AuthenticationFlowBindingOverrides{
										Browser: "browser",
									},
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					realmsMock := keycloakapiMocks.NewMockRealmClient(t)
					realmsMock.On("GetAuthenticationFlows", testifymock.Anything, "realm").
						Return(nil, (*keycloakapi.Response)(nil), errors.New("realm api error"))

					return &keycloakapi.KeycloakClient{Realms: realmsMock}
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get auth flows")
			},
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionFalse,
				Reason: ReasonKeycloakAPIError,
			},
		},
		{
			name: "error when browser flow not found in realm",
			fields: fields{
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
									AuthenticationFlowBindingOverrides: &keycloakApi.AuthenticationFlowBindingOverrides{
										Browser: "custom-browser",
									},
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					realmsMock := keycloakapiMocks.NewMockRealmClient(t)
					realmsMock.On("GetAuthenticationFlows", testifymock.Anything, "realm").
						Return([]keycloakapi.AuthenticationFlowRepresentation{
							{Id: ptr.To("id1"), Alias: ptr.To("browser")},
						}, (*keycloakapi.Response)(nil), nil)

					return &keycloakapi.KeycloakClient{Realms: realmsMock}
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "auth flow custom-browser not found in realm")
			},
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionFalse,
				Reason: ReasonKeycloakAPIError,
			},
		},
		{
			name: "error when direct grant flow not found in realm",
			fields: fields{
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
									AuthenticationFlowBindingOverrides: &keycloakApi.AuthenticationFlowBindingOverrides{
										Browser:     "browser",
										DirectGrant: "custom-grant",
									},
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					realmsMock := keycloakapiMocks.NewMockRealmClient(t)
					realmsMock.On("GetAuthenticationFlows", testifymock.Anything, "realm").
						Return([]keycloakapi.AuthenticationFlowRepresentation{
							{Id: ptr.To("id1"), Alias: ptr.To("browser")},
						}, (*keycloakapi.Response)(nil), nil)

					return &keycloakapi.KeycloakClient{Realms: realmsMock}
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "auth flow custom-grant not found in realm")
			},
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionFalse,
				Reason: ReasonKeycloakAPIError,
			},
		},
		{
			name: "error when GetClientByClientID returns non-NotFound error",
			fields: fields{
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
									Public:   true,
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), errors.New("connection refused"))

					return &keycloakapi.KeycloakClient{Clients: clientsMock}
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to check client id")
			},
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionFalse,
				Reason: ReasonKeycloakAPIError,
			},
		},
		{
			name: "error when UpdateClient fails",
			fields: fields{
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
									Public:   true,
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return(&keycloakapi.ClientRepresentation{Id: ptr.To("123")}, (*keycloakapi.Response)(nil), nil)
					clientsMock.On("UpdateClient", testifymock.Anything, "realm", "123", testifymock.Anything).
						Return((*keycloakapi.Response)(nil), errors.New("update failed"))

					return &keycloakapi.KeycloakClient{Clients: clientsMock}
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to update keycloak client")
			},
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionFalse,
				Reason: ReasonKeycloakAPIError,
			},
		},
		{
			name: "error when CreateClient fails",
			fields: fields{
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
									Public:   true,
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
						Once()
					clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
						Return((*keycloakapi.Response)(nil), errors.New("create failed"))

					return &keycloakapi.KeycloakClient{Clients: clientsMock}
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to create client")
			},
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionFalse,
				Reason: ReasonKeycloakAPIError,
			},
		},
		{
			name: "create client fallback when Location header missing",
			fields: fields{
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
									Public:   true,
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
						Once()
					clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
						Return(&keycloakapi.Response{
							HTTPResponse: &http.Response{Header: http.Header{}},
						}, nil)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return(&keycloakapi.ClientRepresentation{Id: ptr.To("456")}, (*keycloakapi.Response)(nil), nil)

					return &keycloakapi.KeycloakClient{Clients: clientsMock}
				},
			},
			wantErr: require.NoError,
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionTrue,
				Reason: ReasonClientUpdated,
			},
		},
		{
			name: "error when fallback GetClientByClientID fails after create",
			fields: fields{
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
									Public:   true,
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
						Once()
					clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
						Return(&keycloakapi.Response{
							HTTPResponse: &http.Response{Header: http.Header{}},
						}, nil)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), errors.New("lookup failed"))

					return &keycloakapi.KeycloakClient{Clients: clientsMock}
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get client id after creation")
			},
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionFalse,
				Reason: ReasonKeycloakAPIError,
			},
		},
		{
			name: "error when fallback returns nil client after create",
			fields: fields{
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
									Public:   true,
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
						Once()
					clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
						Return(&keycloakapi.Response{
							HTTPResponse: &http.Response{Header: http.Header{}},
						}, nil)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", "test-client-id").
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), nil)

					return &keycloakapi.KeycloakClient{Clients: clientsMock}
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "created client has no ID")
			},
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionFalse,
				Reason: ReasonKeycloakAPIError,
			},
		},
		{
			name: "error when GetSecretFromRef fails",
			fields: fields{
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
									Secret:   secretref.GenerateSecretRef("my-secret", "key"),
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					m := mocks.NewMockRefClient(t)
					m.On("GetSecretFromRef", testifymock.Anything, testifymock.Anything, "default").
						Return("", errors.New("secret not found"))

					return m
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					return &keycloakapi.KeycloakClient{}
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error getting client secret")
			},
			wantCondition: &metav1.Condition{
				Type:   ConditionClientSynced,
				Status: metav1.ConditionFalse,
				Reason: ReasonKeycloakAPIError,
			},
		},
		{
			name: "error when setSecretRef k8s Update fails",
			fields: fields{
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
									Secret:   "plain-old-secret",
								},
							},
						).
						WithInterceptorFuncs(interceptor.Funcs{
							Update: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.UpdateOption) error {
								return errors.New("update conflict")
							},
						}).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					return &keycloakapi.KeycloakClient{}
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to update client with secret ref")
			},
			wantCondition: nil,
		},
		{
			name: "error when k8s Create fails in generateSecret",
			fields: fields{
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
							},
						).
						WithInterceptorFuncs(interceptor.Funcs{
							Create: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.CreateOption) error {
								return errors.New("create denied")
							},
						}).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					return &keycloakapi.KeycloakClient{}
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to create secret")
			},
			wantCondition: nil,
		},
		{
			name: "error when k8s Update fails in generateSecret",
			fields: fields{
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
							},
						).
						WithInterceptorFuncs(interceptor.Funcs{
							Update: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.UpdateOption) error {
								return errors.New("update failed")
							},
						}).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					return mocks.NewMockRefClient(t)
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      "test-client",
					Namespace: "default",
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					return &keycloakapi.KeycloakClient{}
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to update client with new secret")
			},
			wantCondition: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			require.NoError(t, keycloakApi.AddToScheme(s))
			require.NoError(t, corev1.AddToScheme(s))

			cl := &keycloakApi.KeycloakClient{}
			require.NoError(t, tt.fields.client(t).Get(context.Background(), tt.args.keycloakClient, cl))

			el := NewPutClient(tt.args.kClient(t), tt.fields.client(t), tt.fields.secretRef(t))
			clientCtx := &ClientContext{}
			err := el.Serve(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				cl,
				"realm",
				clientCtx,
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

func TestConvertSpecToClientRepresentation(t *testing.T) {
	tests := []struct {
		name              string
		spec              keycloakApi.KeycloakClientSpec
		clientSecret      string
		authFlowOverrides map[string]string
		check             func(t *testing.T, cr keycloakapi.ClientRepresentation)
	}{
		{
			name: "minimal - ClientId only",
			spec: keycloakApi.KeycloakClientSpec{ClientId: "my-client"},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.Equal(t, "my-client", *cr.ClientId)
				require.Nil(t, cr.Protocol)
				require.Nil(t, cr.RootUrl)
				require.Nil(t, cr.BaseUrl)
				require.Nil(t, cr.AdminUrl)
				require.Nil(t, cr.Secret)
				require.Nil(t, cr.Attributes)
				require.Nil(t, cr.RedirectUris)
				require.Nil(t, cr.WebOrigins)
				require.Nil(t, cr.AuthenticationFlowBindingOverrides)
			},
		},
		{
			name: "with protocol",
			spec: keycloakApi.KeycloakClientSpec{ClientId: "c", Protocol: ptr.To("saml")},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.NotNil(t, cr.Protocol)
				require.Equal(t, "saml", *cr.Protocol)
			},
		},
		{
			name: "WebUrl without HomeUrl",
			spec: keycloakApi.KeycloakClientSpec{ClientId: "c", WebUrl: "https://example.com"},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.Equal(t, "https://example.com", *cr.RootUrl)
				require.Equal(t, "https://example.com", *cr.BaseUrl)
				require.Equal(t, []string{"https://example.com/*"}, *cr.RedirectUris)
				require.Equal(t, []string{"https://example.com"}, *cr.WebOrigins)
			},
		},
		{
			name: "WebUrl with HomeUrl",
			spec: keycloakApi.KeycloakClientSpec{ClientId: "c", WebUrl: "https://example.com", HomeUrl: "https://home.example.com"},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.Equal(t, "https://example.com", *cr.RootUrl)
				require.Equal(t, "https://home.example.com", *cr.BaseUrl)
			},
		},
		{
			name: "explicit RedirectUris overrides WebUrl fallback",
			spec: keycloakApi.KeycloakClientSpec{ClientId: "c", WebUrl: "https://example.com", RedirectUris: []string{"https://example.com/callback"}},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.Equal(t, []string{"https://example.com/callback"}, *cr.RedirectUris)
			},
		},
		{
			name: "explicit WebOrigins overrides WebUrl fallback",
			spec: keycloakApi.KeycloakClientSpec{ClientId: "c", WebUrl: "https://example.com", WebOrigins: []string{"https://other.example.com"}},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.Equal(t, []string{"https://other.example.com"}, *cr.WebOrigins)
			},
		},
		{
			name: "with Attributes",
			spec: keycloakApi.KeycloakClientSpec{ClientId: "c", Attributes: map[string]string{"key": "val"}},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.NotNil(t, cr.Attributes)
				require.Equal(t, "val", (*cr.Attributes)["key"])
			},
		},
		{
			name: "with AdminUrl",
			spec: keycloakApi.KeycloakClientSpec{ClientId: "c", AdminUrl: "https://admin.example.com"},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.Equal(t, "https://admin.example.com", *cr.AdminUrl)
			},
		},
		{
			name:         "with clientSecret",
			spec:         keycloakApi.KeycloakClientSpec{ClientId: "c"},
			clientSecret: "supersecret",
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.Equal(t, "supersecret", *cr.Secret)
			},
		},
		{
			name:              "with authFlowOverrides",
			spec:              keycloakApi.KeycloakClientSpec{ClientId: "c"},
			authFlowOverrides: map[string]string{"browser": "flow-id-1"},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.NotNil(t, cr.AuthenticationFlowBindingOverrides)
				require.Equal(t, "flow-id-1", (*cr.AuthenticationFlowBindingOverrides)["browser"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := convertSpecToClientRepresentation(&tt.spec, tt.clientSecret, tt.authFlowOverrides)
			tt.check(t, cr)
		})
	}
}
