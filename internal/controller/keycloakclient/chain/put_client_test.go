package chain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
									Secret:   secretref.GenerateSecretRef("client-secret", "secret"),
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					m := mocks.NewMockRefClient(t)

					m.On("GetSecretFromRef", testifymock.Anything, testifymock.Anything, testClientNamespace).
						Return("client-secret", "secret:client-secret:secret@uid-1@100", nil)

					return m
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      testClientName,
					Namespace: testClientNamespace,
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)

					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
									Secret:   "client-secret",
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					m := mocks.NewMockRefClient(t)

					m.On("GetSecretFromRef", testifymock.Anything, testifymock.Anything, testClientNamespace).
						Return("client-secret", "secret:client-secret:secret@uid-1@100", nil)

					return m
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      testClientName,
					Namespace: testClientNamespace,
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)

					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)

					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
									Secret:   secretref.GenerateSecretRef("client-secret", "secret"),
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					m := mocks.NewMockRefClient(t)

					m.On("GetSecretFromRef", testifymock.Anything, testifymock.Anything, testClientNamespace).
						Return("client-secret", "secret:client-secret:secret@uid-1@100", nil)

					return m
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      testClientName,
					Namespace: testClientNamespace,
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)

					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)

					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
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

					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
						Once()
					clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
						Return(&keycloakapi.Response{
							HTTPResponse: &http.Response{Header: http.Header{}},
						}, nil)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
						Once()
					clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
						Return(&keycloakapi.Response{
							HTTPResponse: &http.Response{Header: http.Header{}},
						}, nil)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
				},
				kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
					clientsMock := keycloakapiMocks.NewMockClientsClient(t)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
						Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
						Once()
					clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
						Return(&keycloakapi.Response{
							HTTPResponse: &http.Response{Header: http.Header{}},
						}, nil)
					clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
									Secret:   secretref.GenerateSecretRef("my-secret", "key"),
								},
							},
						).
						Build()
				},
				secretRef: func(t *testing.T) secretRef {
					m := mocks.NewMockRefClient(t)
					m.On("GetSecretFromRef", testifymock.Anything, testifymock.Anything, testClientNamespace).
						Return("", "", errors.New("secret not found"))

					return m
				},
			},
			args: args{
				keycloakClient: client.ObjectKey{
					Name:      testClientName,
					Namespace: testClientNamespace,
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
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
									Name:      testClientName,
									Namespace: testClientNamespace,
								},
								Spec: keycloakApi.KeycloakClientSpec{
									ClientId: testClientID,
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
					Name:      testClientName,
					Namespace: testClientNamespace,
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
			spec: keycloakApi.KeycloakClientSpec{ClientId: "c", WebUrl: "https://example.com", RedirectUris: ptr.To([]string{"https://example.com/callback"})},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.Equal(t, []string{"https://example.com/callback"}, *cr.RedirectUris)
			},
		},
		{
			name: "explicit WebOrigins overrides WebUrl fallback",
			spec: keycloakApi.KeycloakClientSpec{ClientId: "c", WebUrl: "https://example.com", WebOrigins: ptr.To([]string{"https://other.example.com"})},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.Equal(t, []string{"https://other.example.com"}, *cr.WebOrigins)
			},
		},
		{
			name: "explicit empty RedirectUris and WebOrigins override WebUrl fallback",
			spec: keycloakApi.KeycloakClientSpec{
				ClientId:     "c",
				WebUrl:       "https://example.com",
				RedirectUris: ptr.To([]string{}),
				WebOrigins:   ptr.To([]string{}),
			},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.Equal(t, ptr.To([]string{}), cr.RedirectUris)
				require.Equal(t, ptr.To([]string{}), cr.WebOrigins)

				body, err := json.Marshal(cr)
				require.NoError(t, err)
				require.Contains(t, string(body), `"redirectUris":[]`)
				require.Contains(t, string(body), `"webOrigins":[]`)
			},
		},
		{
			name: "explicit empty RedirectUris and WebOrigins without WebUrl",
			spec: keycloakApi.KeycloakClientSpec{
				ClientId:     "c",
				RedirectUris: ptr.To([]string{}),
				WebOrigins:   ptr.To([]string{}),
			},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.Equal(t, ptr.To([]string{}), cr.RedirectUris)
				require.Equal(t, ptr.To([]string{}), cr.WebOrigins)
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
			name: "with AccessTokenLifespan",
			spec: keycloakApi.KeycloakClientSpec{
				ClientId:         "c",
				AdvancedSettings: &keycloakApi.KeycloakClientAdvancedSettings{AccessTokenLifespan: ptr.To(300)},
			},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.NotNil(t, cr.Attributes)
				require.Equal(t, "300", (*cr.Attributes)["access.token.lifespan"])
			},
		},
		{
			name: "AccessTokenLifespan merges with Attributes",
			spec: keycloakApi.KeycloakClientSpec{
				ClientId:         "c",
				Attributes:       map[string]string{"other": "val"},
				AdvancedSettings: &keycloakApi.KeycloakClientAdvancedSettings{AccessTokenLifespan: ptr.To(600)},
			},
			check: func(t *testing.T, cr keycloakapi.ClientRepresentation) {
				require.NotNil(t, cr.Attributes)
				require.Equal(t, "600", (*cr.Attributes)["access.token.lifespan"])
				require.Equal(t, "val", (*cr.Attributes)["other"])
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

func TestPutClient_Serve_KeepsExplicitEmptyListsAcrossSpecUpdate(t *testing.T) {
	tests := []struct {
		name       string
		secret     string
		wantSecret string
	}{
		{
			name:       "generateSecret Update",
			secret:     "",
			wantSecret: secretref.GenerateSecretRef("keycloak-client-"+testClientName+"-secret", keycloakApi.ClientSecretKey),
		},
		{
			name:       "setSecretRef Update",
			secret:     "client-secret",
			wantSecret: secretref.GenerateSecretRef("client-secret", keycloakApi.ClientSecretKey),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			require.NoError(t, keycloakApi.AddToScheme(s))
			require.NoError(t, corev1.AddToScheme(s))

			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithStatusSubresource(&keycloakApi.KeycloakClient{}).
				WithObjects(
					&keycloakApi.KeycloakClient{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testClientName,
							Namespace: testClientNamespace,
						},
						Spec: keycloakApi.KeycloakClientSpec{
							ClientId:     testClientID,
							Secret:       tt.secret,
							WebUrl:       "https://example.com",
							RedirectUris: ptr.To([]string{}),
							WebOrigins:   ptr.To([]string{}),
						},
					},
				).
				Build()

			refMock := mocks.NewMockRefClient(t)
			if tt.secret != "" {
				refMock.On("GetSecretFromRef", testifymock.Anything, tt.wantSecret, testClientNamespace).
					Return("secret-value", "secret:client-secret:clientSecret@uid-1@100", nil)
			}

			var sent keycloakapi.ClientRepresentation

			clientsMock := keycloakapiMocks.NewMockClientsClient(t)
			clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
				Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
				Once()
			clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
				Run(func(args testifymock.Arguments) {
					sent = args.Get(2).(keycloakapi.ClientRepresentation)
				}).
				Return(&keycloakapi.Response{
					HTTPResponse: &http.Response{
						Header: http.Header{"Location": []string{"http://host/admin/realms/realm/clients/123"}},
					},
				}, nil)

			cr := &keycloakApi.KeycloakClient{}
			key := client.ObjectKey{Name: testClientName, Namespace: testClientNamespace}
			require.NoError(t, k8sClient.Get(context.Background(), key, cr))

			el := NewPutClient(&keycloakapi.KeycloakClient{Clients: clientsMock}, k8sClient, refMock)
			require.NoError(t, el.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), cr, "realm", &ClientContext{}))

			require.Equal(t, ptr.To([]string{}), sent.RedirectUris)
			require.Equal(t, ptr.To([]string{}), sent.WebOrigins)

			stored := &keycloakApi.KeycloakClient{}
			require.NoError(t, k8sClient.Get(context.Background(), key, stored))
			require.Equal(t, tt.wantSecret, stored.Spec.Secret)
			require.Equal(t, ptr.To([]string{}), stored.Spec.RedirectUris)
			require.Equal(t, ptr.To([]string{}), stored.Spec.WebOrigins)
		})
	}
}

func TestPutClient_Serve_BlankClientIDCreatesWithoutLookup(t *testing.T) {
	var (
		mu       sync.Mutex
		requests []string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/master/protocol/openid-connect/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","refresh_token":"refresh","token_type":"Bearer"}`))
		case "/admin/realms/master":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			mu.Lock()
			requests = append(requests, r.Method+" "+r.URL.Path)
			mu.Unlock()

			if r.Method == http.MethodPost {
				w.Header().Set("Location", "http://host/admin/realms/"+testRealmName+"/clients/created-uuid")
				w.WriteHeader(http.StatusCreated)

				return
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"account-uuid","clientId":"account"}]`))
		}
	}))
	t.Cleanup(server.Close)

	kClient, err := keycloakapi.NewKeycloakClient(
		context.Background(), server.URL, "admin-cli", keycloakapi.WithClientSecret("secret"),
	)
	require.NoError(t, err)

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	cr := &keycloakApi.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{Name: testClientName, Namespace: testClientNamespace},
		Spec:       keycloakApi.KeycloakClientSpec{ClientId: " ", Public: true},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	clientCtx := &ClientContext{}

	err = NewPutClient(kClient, k8sClient, mocks.NewMockRefClient(t)).
		Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), cr, testRealmName, clientCtx)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	require.Equal(t, []string{"POST /admin/realms/" + testRealmName + "/clients"}, requests)
	require.Equal(t, "created-uuid", clientCtx.ClientUUID)
}

const testSecretVersion = "secret:s:k@uid@1"

func idempotencyCR() *keycloakApi.KeycloakClient {
	return &keycloakApi.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:       testClientName,
			Namespace:  testClientNamespace,
			Generation: 2,
		},
		Spec: keycloakApi.KeycloakClientSpec{
			ClientId:            testClientID,
			Secret:              secretref.GenerateSecretRef("s", "k"),
			WebUrl:              "https://example.com",
			Enabled:             true,
			StandardFlowEnabled: true,
			Attributes:          map[string]string{"a": "b"},
			RedirectUris:        ptr.To([]string{"https://example.com/b", "https://example.com/a"}),
		},
		Status: keycloakApi.KeycloakClientStatus{
			ObservedGeneration: 2,
			ConfigSecretsHash:  clientSecretsHash(testSecretVersion),
		},
	}
}

// existingFor returns the representation Keycloak would return for the CR: the PUT body plus Id.
func existingFor(t *testing.T, cr *keycloakApi.KeycloakClient) *keycloakapi.ClientRepresentation {
	t.Helper()

	rep := deepCopyClientRepresentation(t, convertSpecToClientRepresentation(&cr.Spec, "val", nil))
	rep.Id = ptr.To("123")

	return &rep
}

// deepCopyClientRepresentation returns a copy that shares no pointer, map or slice with rep.
func deepCopyClientRepresentation(t *testing.T, rep keycloakapi.ClientRepresentation) keycloakapi.ClientRepresentation {
	t.Helper()

	raw, err := json.Marshal(rep)
	require.NoError(t, err)

	var out keycloakapi.ClientRepresentation

	require.NoError(t, json.Unmarshal(raw, &out))

	return out
}

// serveExisting runs PutClient against an existing client and reports whether UpdateClient was called.
func serveExisting(
	t *testing.T,
	cr *keycloakApi.KeycloakClient,
	existing *keycloakapi.ClientRepresentation,
	wantUpdate bool,
	updateErr error,
) (*keycloakApi.KeycloakClient, error) {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	k8sClient := fake.NewClientBuilder().
		WithScheme(s).
		WithStatusSubresource(&keycloakApi.KeycloakClient{}).
		WithObjects(cr.DeepCopy()).
		Build()

	refMock := mocks.NewMockRefClient(t)
	if !cr.Spec.Public {
		refMock.On("GetSecretFromRef", testifymock.Anything, testifymock.Anything, testClientNamespace).
			Return("val", testSecretVersion, nil)
	}

	clientsMock := keycloakapiMocks.NewMockClientsClient(t)
	clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
		Return(existing, (*keycloakapi.Response)(nil), nil).
		Once()

	if wantUpdate {
		clientsMock.On("UpdateClient", testifymock.Anything, "realm", "123", testifymock.Anything).
			Return((*keycloakapi.Response)(nil), updateErr).
			Once()
	}

	kc := &keycloakapi.KeycloakClient{Clients: clientsMock}

	if cr.Spec.AuthenticationFlowBindingOverrides != nil {
		realmsMock := keycloakapiMocks.NewMockRealmClient(t)
		realmsMock.On("GetAuthenticationFlows", testifymock.Anything, "realm").
			Return([]keycloakapi.AuthenticationFlowRepresentation{
				{Alias: ptr.To("flow"), Id: ptr.To("flow-id")},
			}, (*keycloakapi.Response)(nil), nil)

		kc.Realms = realmsMock
	}

	err := NewPutClient(kc, k8sClient, refMock).
		Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), cr, "realm", &ClientContext{})

	return cr, err
}

func TestPutClient_Serve_SkipsUpdateWhenConverged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutateCR func(cr *keycloakApi.KeycloakClient)
		existing func(rep *keycloakapi.ClientRepresentation)
		wantHash string
	}{
		{
			name:     "confidential, spec-built representation",
			wantHash: clientSecretsHash(testSecretVersion),
		},
		{
			name: "confidential, GET-shaped representation",
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.Protocol = ptr.To("openid-connect")
				rep.Attributes = ptr.To(map[string]string{
					"a":                           "b",
					"realm_client":                "false",
					"client.secret.creation.time": "1700000000",
					"post.logout.redirect.uris":   "+",
				})
				rep.Access = ptr.To(map[string]bool{"view": true})
				rep.DefaultClientScopes = ptr.To([]string{"profile"})
				rep.OptionalClientScopes = ptr.To([]string{"phone"})
				rep.NodeReRegistrationTimeout = ptr.To(int32(-1))
			},
			wantHash: clientSecretsHash(testSecretVersion),
		},
		{
			name: "public",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.Public = true
				cr.Spec.Secret = ""
				cr.Status.ConfigSecretsHash = ""
			},
			wantHash: "",
		},
		{
			name: "fetched secret masked",
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.Secret = ptr.To(keycloakapi.MaskedSecretValue)
			},
			wantHash: clientSecretsHash(testSecretVersion),
		},
		{
			name: "masked fetched attribute",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.Attributes["secret.attr"] = "desired"
			},
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.Attributes = ptr.To(map[string]string{"a": "b", "secret.attr": keycloakapi.MaskedSecretValue})
			},
			wantHash: clientSecretsHash(testSecretVersion),
		},
		{
			name: "non-breaking space entry kept by Keycloak",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.RedirectUris = ptr.To([]string{"\u00a0", "https://a/*"})
			},
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.RedirectUris = ptr.To([]string{"\u00a0", "https://a/*"})
			},
			wantHash: clientSecretsHash(testSecretVersion),
		},
		{
			name: "uri order and duplicates",
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.RedirectUris = ptr.To([]string{"https://example.com/a", "https://example.com/b", "https://example.com/a"})
			},
			wantHash: clientSecretsHash(testSecretVersion),
		},
		{
			name: "extra fetched attribute",
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.Attributes = ptr.To(map[string]string{"a": "b", "extra": "x"})
			},
			wantHash: clientSecretsHash(testSecretVersion),
		},
		{
			name: "blank desired attribute absent in fetched",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.Attributes["blank"] = " "
			},
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.Attributes = ptr.To(map[string]string{"a": "b"})
			},
			wantHash: clientSecretsHash(testSecretVersion),
		},
		{
			name: "realm_client equal to computed value",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.Attributes["realm_client"] = "false"
			},
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.Attributes = ptr.To(map[string]string{"a": "b", "realm_client": "false"})
			},
			wantHash: clientSecretsHash(testSecretVersion),
		},
		{
			name: "protocol nil in body",
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.Protocol = ptr.To("openid-connect")
			},
			wantHash: clientSecretsHash(testSecretVersion),
		},
		{
			name: "public with authorizationServicesEnabled",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.Public = true
				cr.Spec.Secret = ""
				cr.Status.ConfigSecretsHash = ""
				cr.Spec.AuthorizationServicesEnabled = true
			},
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.AuthorizationServicesEnabled = ptr.To(false)
			},
			wantHash: "",
		},
		{
			name: "bearer-only with authorizationServicesEnabled",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.BearerOnly = true
				cr.Spec.AuthorizationServicesEnabled = true
			},
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.AuthorizationServicesEnabled = ptr.To(false)
			},
			wantHash: clientSecretsHash(testSecretVersion),
		},
		{
			name: "blank uri and origin entries",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.RedirectUris = ptr.To([]string{"", "https://a/*"})
				cr.Spec.WebOrigins = ptr.To([]string{""})
			},
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.RedirectUris = ptr.To([]string{"https://a/*"})
				rep.WebOrigins = ptr.To([]string{})
			},
			wantHash: clientSecretsHash(testSecretVersion),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := idempotencyCR()
			if tt.mutateCR != nil {
				tt.mutateCR(cr)
			}

			existing := existingFor(t, cr)
			if tt.existing != nil {
				tt.existing(existing)
			}

			got, err := serveExisting(t, cr, existing, false, nil)
			require.NoError(t, err)
			require.Equal(t, tt.wantHash, got.Status.ConfigSecretsHash)
		})
	}
}

func TestPutClient_Serve_UpdatesWhenNeeded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutateCR func(cr *keycloakApi.KeycloakClient)
		existing func(rep *keycloakapi.ClientRepresentation)
	}{
		{
			name: "generation differs from observed",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Status.ObservedGeneration = 1
			},
		},
		{
			name: "hash differs only",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Status.ConfigSecretsHash = "stale"
			},
		},
		{
			name: "description drift",
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.Description = ptr.To("drift")
			},
		},
		{
			name: "enabled drift",
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.Enabled = ptr.To(false)
			},
		},
		{
			name: "authorizationServicesEnabled false in Keycloak",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.AuthorizationServicesEnabled = true
			},
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.AuthorizationServicesEnabled = ptr.To(false)
			},
		},
		{
			name: "clear secret differs",
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.Secret = ptr.To("other")
			},
		},
		{
			name: "uri set differs",
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.RedirectUris = ptr.To([]string{"https://example.com/a"})
			},
		},
		{
			name: "webOrigins differ",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.WebOrigins = ptr.To([]string{"https://example.com"})
			},
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.WebOrigins = ptr.To([]string{"https://other.example.com"})
			},
		},
		{
			name: "explicit empty redirectUris against populated client",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.RedirectUris = ptr.To([]string{})
			},
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.RedirectUris = ptr.To([]string{"https://example.com/a"})
			},
		},
		{
			name: "explicit empty webOrigins against populated client",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.WebOrigins = ptr.To([]string{})
			},
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.WebOrigins = ptr.To([]string{"https://example.com"})
			},
		},
		{
			name: "masked attribute does not hide other drift",
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.Attributes = ptr.To(map[string]string{"a": keycloakapi.MaskedSecretValue, "b": "x"})
			},
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.Attributes["b"] = "y"
			},
		},
		{
			name: "attribute value differs",
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.Attributes = ptr.To(map[string]string{"a": "changed"})
			},
		},
		{
			name: "attribute missing",
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.Attributes = nil
			},
		},
		{
			name: "flow override differs",
			mutateCR: func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.AuthenticationFlowBindingOverrides = &keycloakApi.AuthenticationFlowBindingOverrides{Browser: "flow"}
			},
			existing: func(rep *keycloakapi.ClientRepresentation) {
				rep.AuthenticationFlowBindingOverrides = ptr.To(map[string]string{"browser": "other-id"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := idempotencyCR()
			if tt.mutateCR != nil {
				tt.mutateCR(cr)
			}

			flows := map[string]string(nil)
			if cr.Spec.AuthenticationFlowBindingOverrides != nil {
				flows = map[string]string{browserAuthFlow: "flow-id"}
			}

			rep := deepCopyClientRepresentation(t, convertSpecToClientRepresentation(&cr.Spec, "val", flows))
			rep.Id = ptr.To("123")

			existing := &rep
			if cr.Spec.AuthenticationFlowBindingOverrides != nil {
				existing.AuthenticationFlowBindingOverrides = ptr.To(map[string]string{browserAuthFlow: "flow-id"})
			}

			if tt.existing != nil {
				tt.existing(existing)
			}

			_, err := serveExisting(t, cr, existing, true, nil)
			require.NoError(t, err)
		})
	}
}

func TestPutClient_Serve_UpdateErrorKeepsHash(t *testing.T) {
	t.Parallel()

	cr := idempotencyCR()
	cr.Status.ObservedGeneration = 1
	cr.Status.ConfigSecretsHash = "old"

	_, err := serveExisting(t, cr, existingFor(t, cr), true, errors.New("update failed"))
	require.Error(t, err)
	require.Equal(t, "old", cr.Status.ConfigSecretsHash)
}

func TestPutClient_Serve_StampsHashOnCreate(t *testing.T) {
	t.Parallel()

	newClients := func(t *testing.T) *keycloakapiMocks.MockClientsClient {
		clientsMock := keycloakapiMocks.NewMockClientsClient(t)
		clientsMock.On("GetClientByClientID", testifymock.Anything, "realm", testClientID).
			Return((*keycloakapi.ClientRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).
			Once()
		clientsMock.On("CreateClient", testifymock.Anything, "realm", testifymock.Anything).
			Return(&keycloakapi.Response{
				HTTPResponse: &http.Response{
					Header: http.Header{"Location": []string{"http://host/admin/realms/realm/clients/123"}},
				},
			}, nil)

		return clientsMock
	}

	newK8s := func(t *testing.T, cr *keycloakApi.KeycloakClient) client.Client {
		s := runtime.NewScheme()
		require.NoError(t, keycloakApi.AddToScheme(s))
		require.NoError(t, corev1.AddToScheme(s))

		return fake.NewClientBuilder().
			WithScheme(s).
			WithStatusSubresource(&keycloakApi.KeycloakClient{}).
			WithObjects(cr.DeepCopy()).
			Build()
	}

	t.Run("create path", func(t *testing.T) {
		t.Parallel()

		cr := idempotencyCR()
		cr.Status = keycloakApi.KeycloakClientStatus{}

		refMock := mocks.NewMockRefClient(t)
		refMock.On("GetSecretFromRef", testifymock.Anything, testifymock.Anything, testClientNamespace).
			Return("val", testSecretVersion, nil)

		err := NewPutClient(&keycloakapi.KeycloakClient{Clients: newClients(t)}, newK8s(t, cr), refMock).
			Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), cr, "realm", &ClientContext{})
		require.NoError(t, err)
		require.Equal(t, clientSecretsHash(testSecretVersion), cr.Status.ConfigSecretsHash)
		require.NotEmpty(t, cr.Status.ConfigSecretsHash)
	})

	t.Run("generated secret", func(t *testing.T) {
		t.Parallel()

		cr := idempotencyCR()
		cr.Spec.Secret = ""
		cr.Status = keycloakApi.KeycloakClientStatus{}

		k8sClient := newK8s(t, cr)
		require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKeyFromObject(cr), cr))

		err := NewPutClient(&keycloakapi.KeycloakClient{Clients: newClients(t)}, k8sClient, mocks.NewMockRefClient(t)).
			Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), cr, "realm", &ClientContext{})
		require.NoError(t, err)

		var created corev1.Secret

		require.NoError(t, k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: testClientNamespace, Name: "keycloak-client-" + testClientName + "-secret"}, &created))

		want := secretref.ValuesHashSingle(map[string]string{
			"spec.secret": secretref.SecretKeyVersion(&created, keycloakApi.ClientSecretKey),
		})
		require.Equal(t, want, cr.Status.ConfigSecretsHash)
		require.NotEmpty(t, cr.Status.ConfigSecretsHash)
	})
}

// Every non-nil field of the PUT body must be compared by clientMatchesSpec.
func TestClientMatchesSpec_CoversEveryBodyField(t *testing.T) {
	t.Parallel()

	spec := keycloakApi.KeycloakClientSpec{
		ClientId:                     "cid",
		Secret:                       "ignored",
		Name:                         "name",
		Description:                  "description",
		Enabled:                      true,
		DirectAccess:                 true,
		StandardFlowEnabled:          true,
		ImplicitFlowEnabled:          true,
		AuthorizationServicesEnabled: true,
		ConsentRequired:              true,
		FullScopeAllowed:             true,
		SurrogateAuthRequired:        true,
		FrontChannelLogout:           true,
		ServiceAccount:               &keycloakApi.ServiceAccount{Enabled: true},
		ClientAuthenticatorType:      "client-secret",
		Protocol:                     ptr.To("openid-connect"),
		WebUrl:                       "https://example.com",
		HomeUrl:                      "https://example.com/home",
		AdminUrl:                     "https://example.com/admin",
		Attributes:                   map[string]string{"a": "b"},
		RedirectUris:                 ptr.To([]string{"https://example.com/*"}),
		WebOrigins:                   ptr.To([]string{"https://example.com"}),
	}

	flows := map[string]string{browserAuthFlow: "flow-id"}

	build := func() keycloakapi.ClientRepresentation {
		return convertSpecToClientRepresentation(&spec, "val", flows)
	}

	body := build()
	baseline := deepCopyClientRepresentation(t, body)
	require.True(t, clientMatchesSpec(&baseline, &body))

	bodyType := reflect.TypeOf(body)
	checked := 0

	for i := 0; i < bodyType.NumField(); i++ {
		if reflect.ValueOf(body).Field(i).IsNil() {
			continue
		}

		checked++

		field := bodyType.Field(i)

		t.Run(field.Name, func(t *testing.T) {
			t.Parallel()

			fetched := deepCopyClientRepresentation(t, build())
			perturbField(t, reflect.ValueOf(&fetched).Elem().Field(i))

			desired := build()
			require.False(t, clientMatchesSpec(&fetched, &desired), "field %s is not compared", field.Name)
		})
	}

	require.GreaterOrEqual(t, checked, 20)
}

func perturbField(t *testing.T, f reflect.Value) {
	t.Helper()

	next := reflect.New(f.Type().Elem())

	switch e := f.Elem(); e.Kind() {
	case reflect.String:
		next.Elem().SetString(e.String() + "-perturbed")
	case reflect.Bool:
		next.Elem().SetBool(!e.Bool())
	case reflect.Map:
		m := reflect.MakeMap(e.Type())

		for _, k := range e.MapKeys() {
			m.SetMapIndex(k, reflect.ValueOf(e.MapIndex(k).String()+"-perturbed"))
		}

		next.Elem().Set(m)
	case reflect.Slice:
		next.Elem().Set(reflect.ValueOf([]string{"perturbed"}))
	default:
		t.Fatalf("unsupported field kind %s: extend perturbField", e.Kind())
	}

	f.Set(next)
}
