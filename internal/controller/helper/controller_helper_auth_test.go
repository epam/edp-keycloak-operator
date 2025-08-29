package helper

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/fakehttp"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHelper_SaveKeycloakClientTokenSecret(t *testing.T) {
	s := scheme.Scheme
	utilruntime.Must(keycloakApi.AddToScheme(s))

	kc := keycloakApi.Keycloak{
		Spec: keycloakApi.KeycloakSpec{
			Secret: "test",
		},
	}

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: tokenSecretName(kc.Name),
		},
	}
	cl := fake.NewClientBuilder().WithRuntimeObjects(&kc, &secret).Build()

	h := Helper{
		client: cl,
	}

	err := h.saveKeycloakClientTokenSecret(context.Background(), "secret", "default", []byte("token"))
	require.NoError(t, err)
}

func TestHelper_CreateKeycloakClientFromTokenSecret(t *testing.T) {
	s := scheme.Scheme
	utilruntime.Must(keycloakApi.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddStringResponder("/auth/admin/realms/", "{}").
		BuildAndStart()
	defer mockServer.Close()

	kc := keycloakApi.Keycloak{
		Spec: keycloakApi.KeycloakSpec{
			Url:    mockServer.GetURL(),
			Secret: "test",
		},
	}

	realToken := `eyJhbGciOiJIUzI1NiJ9.eyJSb2xlIjoiQWRtaW4iLCJJc3N1ZXIiOiJJc3N1ZXIiLCJVc2VybmFtZSI6IkphdmFJblVzZSIsImV4cCI6MTYzNDAzOTA2OCwiaWF0IjoxNjM0MDM5MDY4fQ.OZJDXUqfmajSh0vpqL8VnoQGqUXH25CAVkKnoyJX3AI`
	tok := gocloak.JWT{AccessToken: realToken}

	bts, err := json.Marshal(&tok)
	require.NoError(t, err)

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: tokenSecretName(kc.Name),
		},
		Data: map[string][]byte{
			keycloakTokenSecretKey: bts,
		},
	}
	cl := fake.NewClientBuilder().WithRuntimeObjects(&kc, &secret).Build()

	h := Helper{
		client: cl,
	}

	auth, err := MakeKeycloakAuthDataFromKeycloak(context.Background(), &kc, cl)
	require.NoError(t, err)

	_, err = h.createKeycloakClientFromTokenSecret(context.Background(), auth)
	if err == nil {
		t.Fatal("no error on expired token")
	}

	if !strings.Contains(err.Error(), "token is expired") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	tokenParts := strings.Split(realToken, ".")
	rawTokenPayload, _ := base64.RawURLEncoding.DecodeString(tokenParts[1])

	var decodedTokenPayload adapter.JWTPayload

	err = json.Unmarshal(rawTokenPayload, &decodedTokenPayload)
	require.NoError(t, err)

	decodedTokenPayload.Exp = time.Now().Unix() + 1000

	rawTokenPayload, err = json.Marshal(decodedTokenPayload)
	if err != nil {
		t.Fatal("failed to marshal decoded token payload")
	}

	tokenParts[1] = base64.RawURLEncoding.EncodeToString(rawTokenPayload)
	realToken = strings.Join(tokenParts, ".")

	tok = gocloak.JWT{AccessToken: realToken}

	bts, err = json.Marshal(&tok)
	if err != nil {
		t.Fatal("failed to marshal token")
	}

	secret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: tokenSecretName(kc.Name),
		},
		Data: map[string][]byte{
			keycloakTokenSecretKey: bts,
		},
	}
	cl = fake.NewClientBuilder().WithRuntimeObjects(&kc, &secret).Build()

	h = Helper{
		client: cl,
	}

	auth, err = MakeKeycloakAuthDataFromKeycloak(context.Background(), &kc, cl)
	require.NoError(t, err)

	_, err = h.createKeycloakClientFromTokenSecret(context.Background(), auth)
	require.NoError(t, err)
}

func TestHelper_InvalidateKeycloakClientTokenSecret(t *testing.T) {
	sec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: tokenSecretName("kc-name")},
	}

	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(&sec).Build()
	h := Helper{client: fakeCl}

	err := h.InvalidateKeycloakClientTokenSecret(context.Background(), "ns", "kc-name")
	require.NoError(t, err)
}

func TestHelper_InvalidateKeycloakClientTokenSecret_FailureToGet(t *testing.T) {
	sec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: tokenSecretName("wrong-name")},
	}

	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(&sec).Build()
	h := Helper{client: fakeCl}

	err := h.InvalidateKeycloakClientTokenSecret(context.Background(), "ns", "kc-name")
	require.Error(t, err)

	if !k8sErrors.IsNotFound(err) {
		t.Fatalf("wrong error returned: %+v", err)
	}
}

func TestHelper_InvalidateKeycloakClientTokenSecret_FailureToDelete(t *testing.T) {
	sec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: tokenSecretName("kc-name")},
		TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
	}

	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(&sec).Build()
	k8sMock := K8SClientMock{}
	k8sMock.On("Get", types.NamespacedName{Namespace: sec.Namespace, Name: sec.Name}, &corev1.Secret{}).
		Return(fakeCl)

	var dOptions []client.DeleteOption

	k8sMock.On("Delete", &sec, dOptions).Return(errors.New("deletion error"))

	h := Helper{client: &k8sMock}
	err := h.InvalidateKeycloakClientTokenSecret(context.Background(), "ns", "kc-name")
	require.Error(t, err)

	if !strings.Contains(err.Error(), "deletion error") {
		t.Fatalf("wrong error returned: %+v", err)
	}
}

func TestMakeKeycloakAuthDataFromKeycloak(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name      string
		keycloak  *keycloakApi.Keycloak
		k8sClient func(t *testing.T) client.Client
		want      *KeycloakAuthData
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "successfully create keycloak auth data with caCert secret",
			keycloak: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url:    "https://test.com",
					Secret: "admin-secret",
					CACert: &common.SourceRef{
						SecretKeyRef: &common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-secret",
							},
							Key: "ca.crt",
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithRuntimeObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"ca.crt": []byte("test-ca-cert"),
						},
					}).Build()
			},
			want: &KeycloakAuthData{
				Url:             "https://test.com",
				SecretName:      "admin-secret",
				SecretNamespace: "default",
				KeycloakCRName:  "test",
				CACert:          "test-ca-cert",
			},
			wantErr: require.NoError,
		},
		{
			name: "successfully create keycloak auth data with caCert configmap",
			keycloak: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url:    "https://test.com",
					Secret: "admin-secret",
					CACert: &common.SourceRef{
						ConfigMapKeyRef: &common.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-configmap",
							},
							Key: "ca.crt",
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithRuntimeObjects(
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-configmap",
							Namespace: "default",
						},
						Data: map[string]string{
							"ca.crt": "test-ca-cert",
						},
					}).Build()
			},
			want: &KeycloakAuthData{
				Url:             "https://test.com",
				SecretName:      "admin-secret",
				SecretNamespace: "default",
				KeycloakCRName:  "test",
				CACert:          "test-ca-cert",
			},
			wantErr: require.NoError,
		},
		{
			name: "caCert secret not found",
			keycloak: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					CACert: &common.SourceRef{
						SecretKeyRef: &common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-secret",
							},
							Key: "ca.crt",
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get secret")
			},
		},
		{
			name: "caCert configmap not found",
			keycloak: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					CACert: &common.SourceRef{
						ConfigMapKeyRef: &common.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-configmap",
							},
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get configmap")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeKeycloakAuthDataFromKeycloak(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.keycloak, tt.k8sClient(t))

			tt.wantErr(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMakeKeycloakAuthDataFromClusterKeycloak(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name      string
		keycloak  *keycloakAlpha.ClusterKeycloak
		k8sClient func(t *testing.T) client.Client
		want      *KeycloakAuthData
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "successfully create keycloak auth data with caCert secret",
			keycloak: &keycloakAlpha.ClusterKeycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: keycloakAlpha.ClusterKeycloakSpec{
					Url:    "https://test.com",
					Secret: "admin-secret",
					CACert: &common.SourceRef{
						SecretKeyRef: &common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-secret",
							},
							Key: "ca.crt",
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithRuntimeObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: "ns-with-secrets",
						},
						Data: map[string][]byte{
							"ca.crt": []byte("test-ca-cert"),
						},
					},
				).Build()
			},
			want: &KeycloakAuthData{
				Url:             "https://test.com",
				SecretName:      "admin-secret",
				SecretNamespace: "ns-with-secrets",
				KeycloakCRName:  "test",
				CACert:          "test-ca-cert",
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeKeycloakAuthDataFromClusterKeycloak(
				ctrl.LoggerInto(context.Background(),
					logr.Discard()),
				tt.keycloak,
				"ns-with-secrets",
				tt.k8sClient(t),
			)

			tt.wantErr(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestHelper_createKeycloakClientFromLoginPassword(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name        string
		authData    *KeycloakAuthData
		setupHelper func(t *testing.T) *Helper
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name: "successful client creation from login password",
			authData: &KeycloakAuthData{
				Url:                "https://test.keycloak.com",
				SecretName:         "keycloak-admin-secret",
				SecretNamespace:    "default",
				AdminType:          keycloakApi.KeycloakAdminTypeServiceAccount,
				KeycloakCRName:     "test-keycloak",
				CACert:             "test-ca-cert",
				InsecureSkipVerify: false,
			},
			setupHelper: func(t *testing.T) *Helper {
				mockClient := mocks.NewMockClient(t)
				mockClient.On("ExportToken").Return([]byte(`{"access_token":"mock-token","refresh_token":"mock-refresh"}`), nil)

				return &Helper{
					client: fake.NewClientBuilder().WithRuntimeObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "keycloak-admin-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"username": []byte("admin"),
								"password": []byte("password123"),
							},
						},
					).Build(),
					adapterBuilder: func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger, restyClient *resty.Client) (keycloak.Client, error) {
						return mockClient, nil
					},
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "secret not found error",
			authData: &KeycloakAuthData{
				Url:             "https://test.keycloak.com",
				SecretName:      "non-existent-secret",
				SecretNamespace: "default",
				AdminType:       keycloakApi.KeycloakAdminTypeServiceAccount,
				KeycloakCRName:  "test-keycloak",
			},
			setupHelper: func(t *testing.T) *Helper {
				return &Helper{
					client:          fake.NewClientBuilder().Build(),
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "authData login password secret not found")
			},
		},
		{
			name: "CreateKeycloakClient failure",
			authData: &KeycloakAuthData{
				Url:             "https://test.keycloak.com",
				SecretName:      "keycloak-admin-secret",
				SecretNamespace: "default",
				AdminType:       keycloakApi.KeycloakAdminTypeServiceAccount,
				KeycloakCRName:  "test-keycloak",
			},
			setupHelper: func(t *testing.T) *Helper {
				return &Helper{
					client: fake.NewClientBuilder().WithRuntimeObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "keycloak-admin-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"username": []byte("admin"),
								"password": []byte("password123"),
							},
						},
					).Build(),
					adapterBuilder: func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger, restyClient *resty.Client) (keycloak.Client, error) {
						return nil, errors.New("failed to create keycloak client")
					},
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to init authData client adapter")
			},
		},
		{
			name: "token export failure",
			authData: &KeycloakAuthData{
				Url:             "https://test.keycloak.com",
				SecretName:      "keycloak-admin-secret",
				SecretNamespace: "default",
				AdminType:       keycloakApi.KeycloakAdminTypeServiceAccount,
				KeycloakCRName:  "test-keycloak",
			},
			setupHelper: func(t *testing.T) *Helper {
				mockClient := mocks.NewMockClient(t)
				mockClient.On("ExportToken").Return(nil, errors.New("failed to export token"))

				return &Helper{
					client: fake.NewClientBuilder().WithRuntimeObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "keycloak-admin-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"username": []byte("admin"),
								"password": []byte("password123"),
							},
						},
					).Build(),
					adapterBuilder: func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger, restyClient *resty.Client) (keycloak.Client, error) {
						return mockClient, nil
					},
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to export authData client token")
			},
		},
		{
			name: "missing username in secret",
			authData: &KeycloakAuthData{
				Url:             "https://test.keycloak.com",
				SecretName:      "keycloak-admin-secret",
				SecretNamespace: "default",
				AdminType:       keycloakApi.KeycloakAdminTypeServiceAccount,
				KeycloakCRName:  "test-keycloak",
			},
			setupHelper: func(t *testing.T) *Helper {
				mockClient := mocks.NewMockClient(t)
				mockClient.On("ExportToken").Return([]byte(`{"access_token":"mock-token"}`), nil)

				return &Helper{
					client: fake.NewClientBuilder().WithRuntimeObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "keycloak-admin-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"password": []byte("password123"),
							},
						},
					).Build(),
					adapterBuilder: func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger, restyClient *resty.Client) (keycloak.Client, error) {
						// Verify empty username is passed
						require.Equal(t, "", conf.User)
						require.Equal(t, "password123", conf.Password)
						return mockClient, nil
					},
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: require.NoError, // Empty username should still work
		},
		{
			name: "missing password in secret",
			authData: &KeycloakAuthData{
				Url:             "https://test.keycloak.com",
				SecretName:      "keycloak-admin-secret",
				SecretNamespace: "default",
				AdminType:       keycloakApi.KeycloakAdminTypeServiceAccount,
				KeycloakCRName:  "test-keycloak",
			},
			setupHelper: func(t *testing.T) *Helper {
				mockClient := mocks.NewMockClient(t)
				mockClient.On("ExportToken").Return([]byte(`{"access_token":"mock-token"}`), nil)

				return &Helper{
					client: fake.NewClientBuilder().WithRuntimeObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "keycloak-admin-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"username": []byte("admin"),
							},
						},
					).Build(),
					adapterBuilder: func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger, restyClient *resty.Client) (keycloak.Client, error) {
						// Verify empty password is passed
						require.Equal(t, "admin", conf.User)
						require.Equal(t, "", conf.Password)
						return mockClient, nil
					},
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: require.NoError, // Empty password should still work
		},
		{
			name: "token save failure - create secret error",
			authData: &KeycloakAuthData{
				Url:             "https://test.keycloak.com",
				SecretName:      "keycloak-admin-secret",
				SecretNamespace: "default",
				AdminType:       keycloakApi.KeycloakAdminTypeServiceAccount,
				KeycloakCRName:  "test-keycloak",
			},
			setupHelper: func(t *testing.T) *Helper {
				mockClient := mocks.NewMockClient(t)
				mockClient.On("ExportToken").Return([]byte(`{"access_token":"mock-token"}`), nil)

				k8sMock := &K8SClientMock{}
				// Mock the Get call for the admin secret
				k8sMock.On("Get", types.NamespacedName{Namespace: "default", Name: "keycloak-admin-secret"}, &corev1.Secret{}).
					Return(fake.NewClientBuilder().WithRuntimeObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "keycloak-admin-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"username": []byte("admin"),
								"password": []byte("password123"),
							},
						},
					).Build())

				// Mock the Get call for the token secret to return not found
				k8sMock.On("Get", types.NamespacedName{Namespace: "default", Name: tokenSecretName("keycloak-admin-secret")}, &corev1.Secret{}).
					Return(k8sErrors.NewNotFound(corev1.Resource("secrets"), tokenSecretName("keycloak-admin-secret")))

				// Mock the Create call to return an error
				var createOptions []client.CreateOption
				k8sMock.On("Create", mock.AnythingOfType("*v1.Secret"), createOptions).Return(errors.New("failed to create secret"))

				return &Helper{
					client: k8sMock,
					adapterBuilder: func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger, restyClient *resty.Client) (keycloak.Client, error) {
						return mockClient, nil
					},
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to save authData token to secret")
			},
		},
		{
			name: "successful client creation with insecure skip verify",
			authData: &KeycloakAuthData{
				Url:                "https://test.keycloak.com",
				SecretName:         "keycloak-admin-secret",
				SecretNamespace:    "default",
				AdminType:          keycloakApi.KeycloakAdminTypeUser,
				KeycloakCRName:     "test-keycloak",
				InsecureSkipVerify: true,
			},
			setupHelper: func(t *testing.T) *Helper {
				mockClient := mocks.NewMockClient(t)
				mockClient.On("ExportToken").Return([]byte(`{"access_token":"mock-token"}`), nil)

				return &Helper{
					client: fake.NewClientBuilder().WithRuntimeObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "keycloak-admin-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"username": []byte("admin"),
								"password": []byte("password123"),
							},
						},
					).Build(),
					adapterBuilder: func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger, restyClient *resty.Client) (keycloak.Client, error) {
						// Verify the config parameters
						require.Equal(t, "https://test.keycloak.com", conf.Url)
						require.Equal(t, "admin", conf.User)
						require.Equal(t, "password123", conf.Password)
						require.Equal(t, "", conf.RootCertificate)
						require.True(t, conf.InsecureSkipVerify)
						require.Equal(t, keycloakApi.KeycloakAdminTypeUser, adminType)
						return mockClient, nil
					},
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "successful client creation with CA certificate",
			authData: &KeycloakAuthData{
				Url:                "https://test.keycloak.com",
				SecretName:         "keycloak-admin-secret",
				SecretNamespace:    "default",
				AdminType:          keycloakApi.KeycloakAdminTypeServiceAccount,
				KeycloakCRName:     "test-keycloak",
				CACert:             "-----BEGIN CERTIFICATE-----\ntest-ca-cert\n-----END CERTIFICATE-----",
				InsecureSkipVerify: false,
			},
			setupHelper: func(t *testing.T) *Helper {
				mockClient := mocks.NewMockClient(t)
				mockClient.On("ExportToken").Return([]byte(`{"access_token":"mock-token"}`), nil)

				return &Helper{
					client: fake.NewClientBuilder().WithRuntimeObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "keycloak-admin-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"username": []byte("admin"),
								"password": []byte("password123"),
							},
						},
					).Build(),
					adapterBuilder: func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger, restyClient *resty.Client) (keycloak.Client, error) {
						// Verify the config parameters
						require.Equal(t, "https://test.keycloak.com", conf.Url)
						require.Equal(t, "admin", conf.User)
						require.Equal(t, "password123", conf.Password)
						require.Equal(t, "-----BEGIN CERTIFICATE-----\ntest-ca-cert\n-----END CERTIFICATE-----", conf.RootCertificate)
						require.False(t, conf.InsecureSkipVerify)
						require.Equal(t, keycloakApi.KeycloakAdminTypeServiceAccount, adminType)
						return mockClient, nil
					},
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := tt.setupHelper(t)
			ctx := context.Background()

			_, err := helper.createKeycloakClientFromLoginPassword(ctx, tt.authData)
			tt.wantErr(t, err)
		})
	}
}
