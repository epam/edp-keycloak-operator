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
	keycloakclientv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
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

func TestHelper_SaveKeycloakClientTokenSecret_UpdateExisting(t *testing.T) {
	s := scheme.Scheme
	utilruntime.Must(keycloakApi.AddToScheme(s))

	existingSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-token-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			keycloakTokenSecretKey: []byte("old-token"),
		},
	}
	cl := fake.NewClientBuilder().WithRuntimeObjects(&existingSecret).Build()

	h := Helper{
		client: cl,
	}

	err := h.saveKeycloakClientTokenSecret(context.Background(), "test-token-secret", "default", []byte("new-token"))
	require.NoError(t, err)

	// Verify the secret was updated
	var updatedSecret corev1.Secret
	err = cl.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "test-token-secret"}, &updatedSecret)
	require.NoError(t, err)
	require.Equal(t, []byte("new-token"), updatedSecret.Data[keycloakTokenSecretKey])
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
			wantErr: func(t require.TestingT, err error, i ...any) {
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
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get configmap")
			},
		},
		{
			name: "successfully propagate auth spec",
			keycloak: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url: "https://test.com",
					Auth: &common.AuthSpec{
						PasswordGrant: &common.PasswordGrantConfig{
							Username: common.SourceRefOrVal{
								Value: "admin",
							},
							PasswordRef: common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-secret",
								},
								Key: "password",
							},
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			want: &KeycloakAuthData{
				Url:             "https://test.com",
				SecretNamespace: "default",
				KeycloakCRName:  "test",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							Value: "admin",
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-secret",
							},
							Key: "password",
						},
					},
				},
			},
			wantErr: require.NoError,
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
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get credentials")
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
			wantErr: func(t require.TestingT, err error, i ...any) {
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
			wantErr: func(t require.TestingT, err error, i ...any) {
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
				k8sMock.On("Get", types.NamespacedName{Namespace: "default", Name: tokenSecretName("test-keycloak")}, &corev1.Secret{}).
					Return(k8sErrors.NewNotFound(corev1.Resource("secrets"), tokenSecretName("test-keycloak")))

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
			wantErr: func(t require.TestingT, err error, i ...any) {
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

func TestHelper_CreateKeycloakClientV2FromRealm(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	// Create mock server for successful auth cases
	mockServer := fakehttp.NewServerBuilder().
		AddKeycloakAuthResponders().
		BuildAndStart()
	defer mockServer.Close()

	tests := []struct {
		name      string
		realm     *keycloakApi.KeycloakRealm
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, client *keycloakclientv2.KeycloakClient)
	}{
		{
			name: "successfully create v2 client from realm",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "successfully create v2 client with CA cert",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
						CACert: &common.SourceRef{
							SecretKeyRef: &common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "ca-cert-secret",
								},
								Key: "ca.crt",
							},
						},
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ca-cert-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"ca.crt": []byte("-----BEGIN CERTIFICATE-----\ntest-ca-cert\n-----END CERTIFICATE-----"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "successfully create v2 client with insecure skip verify",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:                mockServer.GetURL(),
						Secret:             "keycloak-secret",
						InsecureSkipVerify: true,
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "successfully create v2 client with all options",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
						CACert: &common.SourceRef{
							SecretKeyRef: &common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "ca-cert-secret",
								},
								Key: "ca.crt",
							},
						},
						InsecureSkipVerify: true,
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ca-cert-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"ca.crt": []byte("-----BEGIN CERTIFICATE-----\ntest-ca-cert\n-----END CERTIFICATE-----"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "realm not found",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get keycloak")
			},
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "keycloak not connected",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    "https://keycloak.example.com",
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: false,
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrKeycloakIsNotAvailable)
			},
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "credentials secret not found",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    "https://keycloak.example.com",
						Secret: "non-existent-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get credentials")
			},
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "keycloak CR not found",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "non-existent-keycloak",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get keycloak")
			},
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "empty username in secret should work",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "empty password in secret should work",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "cluster keycloak ref",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakAlpha.ClusterKeycloakKind,
						Name: "test-cluster-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			client, err := helper.CreateKeycloakClientV2FromRealm(ctx, tt.realm)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, client)
			}
		})
	}
}

func TestHelper_CreateKeycloakClientV2FromClusterRealm(t *testing.T) {
	t.Parallel()

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	// Create mock server for successful auth cases
	mockServer := fakehttp.NewServerBuilder().
		AddKeycloakAuthResponders().
		BuildAndStart()

	t.Cleanup(func() {
		mockServer.Close()
	})

	tests := []struct {
		name      string
		realm     *keycloakAlpha.ClusterKeycloakRealm
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, client *keycloakclientv2.KeycloakClient)
	}{
		{
			name: "successfully create v2 client from cluster realm",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "test-cluster-keycloak",
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "successfully create v2 client with CA cert",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "test-cluster-keycloak",
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
						CACert: &common.SourceRef{
							SecretKeyRef: &common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "ca-cert-secret",
								},
								Key: "ca.crt",
							},
						},
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ca-cert-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"ca.crt": []byte("-----BEGIN CERTIFICATE-----\ntest-ca-cert\n-----END CERTIFICATE-----"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "successfully create v2 client with insecure skip verify",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "test-cluster-keycloak",
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:                mockServer.GetURL(),
						Secret:             "keycloak-secret",
						InsecureSkipVerify: true,
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "successfully create v2 client with all options",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "test-cluster-keycloak",
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
						CACert: &common.SourceRef{
							SecretKeyRef: &common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "ca-cert-secret",
								},
								Key: "ca.crt",
							},
						},
						InsecureSkipVerify: true,
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ca-cert-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"ca.crt": []byte("-----BEGIN CERTIFICATE-----\ntest-ca-cert\n-----END CERTIFICATE-----"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "cluster keycloak not found",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "non-existent-cluster-keycloak",
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get cluster keycloak")
			},
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "cluster keycloak not connected",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "test-cluster-keycloak",
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    "https://keycloak.example.com",
						Secret: "keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: false,
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrKeycloakIsNotAvailable)
			},
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "credentials secret not found",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "test-cluster-keycloak",
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    "https://keycloak.example.com",
						Secret: "non-existent-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get credentials")
			},
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "empty username in secret should work",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "test-cluster-keycloak",
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "empty password in secret should work",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "test-cluster-keycloak",
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			client, err := helper.CreateKeycloakClientV2FromClusterRealm(ctx, tt.realm)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, client)
			}
		})
	}
}

func TestHelper_CreateKeycloakClientV2FromKeycloak(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddKeycloakAuthResponders().
		BuildAndStart()

	t.Cleanup(func() {
		mockServer.Close()
	})

	tests := []struct {
		name      string
		kc        *keycloakApi.Keycloak
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, cl *keycloakclientv2.KeycloakClient)
	}{
		{
			name: "successfully create v2 client from keycloak",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url:    mockServer.GetURL(),
					Secret: "keycloak-secret",
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, cl *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, cl)
			},
		},
		{
			name: "successfully create v2 client with insecure skip verify",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url:                mockServer.GetURL(),
					Secret:             "keycloak-secret",
					InsecureSkipVerify: true,
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, cl *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, cl)
			},
		},
		{
			name: "successfully create v2 client with auth passwordGrant",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url: mockServer.GetURL(),
					Auth: &common.AuthSpec{
						PasswordGrant: &common.PasswordGrantConfig{
							Username: common.SourceRefOrVal{
								Value: "admin",
							},
							PasswordRef: common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "keycloak-secret",
								},
								Key: "password",
							},
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, cl *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, cl)
			},
		},
		{
			name: "successfully create v2 client with auth clientCredentials",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url: mockServer.GetURL(),
					Auth: &common.AuthSpec{
						ClientCredentials: &common.ClientCredentialsConfig{
							ClientID: common.SourceRefOrVal{
								Value: "my-admin-client",
							},
							ClientSecretRef: common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "client-secret",
								},
								Key: "secret",
							},
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "client-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"secret": []byte("client-secret-value"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, cl *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, cl)
			},
		},
		{
			name: "auth passwordGrant with username from secret ref",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url: mockServer.GetURL(),
					Auth: &common.AuthSpec{
						PasswordGrant: &common.PasswordGrantConfig{
							Username: common.SourceRefOrVal{
								SourceRef: common.SourceRef{
									SecretKeyRef: &common.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "keycloak-secret",
										},
										Key: "username",
									},
								},
							},
							PasswordRef: common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "keycloak-secret",
								},
								Key: "password",
							},
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, cl *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, cl)
			},
		},
		{
			name: "auth passwordGrant with missing password secret",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url: mockServer.GetURL(),
					Auth: &common.AuthSpec{
						PasswordGrant: &common.PasswordGrantConfig{
							Username: common.SourceRefOrVal{
								Value: "admin",
							},
							PasswordRef: common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "missing-secret",
								},
								Key: "password",
							},
						},
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to resolve password")
			},
			checkFunc: func(t *testing.T, cl *keycloakclientv2.KeycloakClient) {
				require.Nil(t, cl)
			},
		},
		{
			name: "credentials secret not found",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url:    mockServer.GetURL(),
					Secret: "non-existent-secret",
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get credentials")
			},
			checkFunc: func(t *testing.T, cl *keycloakclientv2.KeycloakClient) {
				require.Nil(t, cl)
			},
		},
		{
			name: "ca cert secret not found returns error",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url:    mockServer.GetURL(),
					Secret: "keycloak-secret",
					CACert: &common.SourceRef{
						SecretKeyRef: &common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "non-existent-ca-secret",
							},
							Key: "ca.crt",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get ca cert")
			},
			checkFunc: func(t *testing.T, cl *keycloakclientv2.KeycloakClient) {
				require.Nil(t, cl)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			h := &Helper{
				client: k8sClient,
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			cl, err := h.CreateKeycloakClientV2FromKeycloak(ctx, tt.kc)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, cl)
			}
		})
	}
}

func TestHelper_CreateKeycloakClientV2FromClusterKeycloak(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddKeycloakAuthResponders().
		BuildAndStart()

	t.Cleanup(func() {
		mockServer.Close()
	})

	tests := []struct {
		name      string
		kc        *keycloakAlpha.ClusterKeycloak
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, cl *keycloakclientv2.KeycloakClient)
	}{
		{
			name: "successfully create v2 client from cluster keycloak",
			kc: &keycloakAlpha.ClusterKeycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-keycloak",
				},
				Spec: keycloakAlpha.ClusterKeycloakSpec{
					Url:    mockServer.GetURL(),
					Secret: "keycloak-secret",
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, cl *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, cl)
			},
		},
		{
			name: "successfully create v2 client with insecure skip verify",
			kc: &keycloakAlpha.ClusterKeycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-keycloak",
				},
				Spec: keycloakAlpha.ClusterKeycloakSpec{
					Url:                mockServer.GetURL(),
					Secret:             "keycloak-secret",
					InsecureSkipVerify: true,
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, cl *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, cl)
			},
		},
		{
			name: "credentials secret not found",
			kc: &keycloakAlpha.ClusterKeycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-keycloak",
				},
				Spec: keycloakAlpha.ClusterKeycloakSpec{
					Url:    mockServer.GetURL(),
					Secret: "non-existent-secret",
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get credentials")
			},
			checkFunc: func(t *testing.T, cl *keycloakclientv2.KeycloakClient) {
				require.Nil(t, cl)
			},
		},
		{
			name: "ca cert secret not found returns error",
			kc: &keycloakAlpha.ClusterKeycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-keycloak",
				},
				Spec: keycloakAlpha.ClusterKeycloakSpec{
					Url:    mockServer.GetURL(),
					Secret: "keycloak-secret",
					CACert: &common.SourceRef{
						SecretKeyRef: &common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "non-existent-ca-secret",
							},
							Key: "ca.crt",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get ca cert")
			},
			checkFunc: func(t *testing.T, cl *keycloakclientv2.KeycloakClient) {
				require.Nil(t, cl)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			h := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			cl, err := h.CreateKeycloakClientV2FromClusterKeycloak(ctx, tt.kc)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, cl)
			}
		})
	}
}

func TestHelper_resolveLegacyAuthFromSpec(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name         string
		authData     *KeycloakAuthData
		objects      []client.Object
		wantUsername string
		wantPassword string
		wantAdmin    string
		wantErr      require.ErrorAssertionFunc
	}{
		{
			name: "success with password grant using inline username",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							Value: "admin-user",
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "password-secret",
							},
							Key: "password",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "password-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"password": []byte("secret-password"),
					},
				},
			},
			wantUsername: "admin-user",
			wantPassword: "secret-password",
			wantAdmin:    keycloakApi.KeycloakAdminTypeUser,
			wantErr:      require.NoError,
		},
		{
			name: "success with password grant using secret ref for username",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							SourceRef: common.SourceRef{
								SecretKeyRef: &common.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "username-secret",
									},
									Key: "username",
								},
							},
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "password-secret",
							},
							Key: "password",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "username-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("secret-username"),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "password-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"password": []byte("secret-password"),
					},
				},
			},
			wantUsername: "secret-username",
			wantPassword: "secret-password",
			wantAdmin:    keycloakApi.KeycloakAdminTypeUser,
			wantErr:      require.NoError,
		},
		{
			name: "success with client credentials",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					ClientCredentials: &common.ClientCredentialsConfig{
						ClientID: common.SourceRefOrVal{
							Value: "my-client-id",
						},
						ClientSecretRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "client-secret",
							},
							Key: "secret",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "client-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"secret": []byte("client-secret-value"),
					},
				},
			},
			wantUsername: "my-client-id",
			wantPassword: "client-secret-value",
			wantAdmin:    keycloakApi.KeycloakAdminTypeServiceAccount,
			wantErr:      require.NoError,
		},
		{
			name: "error resolving username secret",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							SourceRef: common.SourceRef{
								SecretKeyRef: &common.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "non-existent-secret",
									},
									Key: "username",
								},
							},
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "password-secret",
							},
							Key: "password",
						},
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to resolve username")
			},
		},
		{
			name: "error resolving password secret",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							Value: "admin",
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "non-existent-secret",
							},
							Key: "password",
						},
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to resolve password")
			},
		},
		{
			name: "error resolving client id secret",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					ClientCredentials: &common.ClientCredentialsConfig{
						ClientID: common.SourceRefOrVal{
							SourceRef: common.SourceRef{
								SecretKeyRef: &common.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "non-existent-secret",
									},
									Key: "client-id",
								},
							},
						},
						ClientSecretRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "client-secret",
							},
							Key: "secret",
						},
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to resolve client id")
			},
		},
		{
			name: "error resolving client secret",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					ClientCredentials: &common.ClientCredentialsConfig{
						ClientID: common.SourceRefOrVal{
							Value: "my-client-id",
						},
						ClientSecretRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "non-existent-secret",
							},
							Key: "secret",
						},
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to resolve client secret")
			},
		},
		{
			name: "error when neither password grant nor client credentials set",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec:        &common.AuthSpec{},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "one of passwordGrant or clientCredentials must be set")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client: k8sClient,
			}

			username, password, adminType, err := helper.resolveLegacyAuthFromSpec(context.Background(), tt.authData)

			tt.wantErr(t, err)

			if err == nil {
				require.Equal(t, tt.wantUsername, username)
				require.Equal(t, tt.wantPassword, password)
				require.Equal(t, tt.wantAdmin, adminType)
			}
		})
	}
}

func TestHelper_buildV2AuthOptions(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name         string
		authData     *KeycloakAuthData
		objects      []client.Object
		wantClientID string
		wantErr      require.ErrorAssertionFunc
	}{
		{
			name: "success with password grant - returns admin-cli client ID",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							Value: "admin",
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "password-secret",
							},
							Key: "password",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "password-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"password": []byte("secret-password"),
					},
				},
			},
			wantClientID: keycloakclientv2.DefaultAdminClientID,
			wantErr:      require.NoError,
		},
		{
			name: "success with client credentials - returns custom client ID",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					ClientCredentials: &common.ClientCredentialsConfig{
						ClientID: common.SourceRefOrVal{
							Value: "my-service-client",
						},
						ClientSecretRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "client-secret",
							},
							Key: "secret",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "client-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"secret": []byte("client-secret-value"),
					},
				},
			},
			wantClientID: "my-service-client",
			wantErr:      require.NoError,
		},
		{
			name: "error resolving username in password grant",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							SourceRef: common.SourceRef{
								SecretKeyRef: &common.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "non-existent-secret",
									},
									Key: "username",
								},
							},
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "password-secret",
							},
							Key: "password",
						},
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to resolve username")
			},
		},
		{
			name: "error resolving password in password grant",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							Value: "admin",
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "non-existent-secret",
							},
							Key: "password",
						},
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to resolve password")
			},
		},
		{
			name: "error resolving client ID in client credentials",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					ClientCredentials: &common.ClientCredentialsConfig{
						ClientID: common.SourceRefOrVal{
							SourceRef: common.SourceRef{
								SecretKeyRef: &common.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "non-existent-secret",
									},
									Key: "client-id",
								},
							},
						},
						ClientSecretRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "client-secret",
							},
							Key: "secret",
						},
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to resolve client id")
			},
		},
		{
			name: "error resolving client secret in client credentials",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					ClientCredentials: &common.ClientCredentialsConfig{
						ClientID: common.SourceRefOrVal{
							Value: "my-service-client",
						},
						ClientSecretRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "non-existent-secret",
							},
							Key: "secret",
						},
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to resolve client secret")
			},
		},
		{
			name: "error when neither password grant nor client credentials set",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec:        &common.AuthSpec{},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "one of passwordGrant or clientCredentials must be set")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client: k8sClient,
			}

			clientID, _, err := helper.buildV2AuthOptions(context.Background(), tt.authData)

			tt.wantErr(t, err)

			if err == nil {
				require.Equal(t, tt.wantClientID, clientID)
			}
		})
	}
}

func TestHelper_createKeycloakClientFromLoginPassword_withAuthSpec(t *testing.T) {
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
			name: "successful client creation with password grant auth spec",
			authData: &KeycloakAuthData{
				Url:             "https://test.keycloak.com",
				SecretNamespace: "default",
				KeycloakCRName:  "test-keycloak",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							Value: "admin-user",
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "password-secret",
							},
							Key: "password",
						},
					},
				},
			},
			setupHelper: func(t *testing.T) *Helper {
				mockClient := mocks.NewMockClient(t)
				mockClient.On("ExportToken").Return([]byte(`{"access_token":"mock-token"}`), nil)

				return &Helper{
					client: fake.NewClientBuilder().WithRuntimeObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "password-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"password": []byte("secret-password"),
							},
						},
					).Build(),
					adapterBuilder: func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger, restyClient *resty.Client) (keycloak.Client, error) {
						require.Equal(t, "admin-user", conf.User)
						require.Equal(t, "secret-password", conf.Password)
						require.Equal(t, keycloakApi.KeycloakAdminTypeUser, adminType)
						return mockClient, nil
					},
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "successful client creation with client credentials auth spec",
			authData: &KeycloakAuthData{
				Url:             "https://test.keycloak.com",
				SecretNamespace: "default",
				KeycloakCRName:  "test-keycloak",
				AuthSpec: &common.AuthSpec{
					ClientCredentials: &common.ClientCredentialsConfig{
						ClientID: common.SourceRefOrVal{
							Value: "service-client",
						},
						ClientSecretRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "client-secret",
							},
							Key: "secret",
						},
					},
				},
			},
			setupHelper: func(t *testing.T) *Helper {
				mockClient := mocks.NewMockClient(t)
				mockClient.On("ExportToken").Return([]byte(`{"access_token":"mock-token"}`), nil)

				return &Helper{
					client: fake.NewClientBuilder().WithRuntimeObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "client-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"secret": []byte("client-secret-value"),
							},
						},
					).Build(),
					adapterBuilder: func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger, restyClient *resty.Client) (keycloak.Client, error) {
						require.Equal(t, "service-client", conf.User)
						require.Equal(t, "client-secret-value", conf.Password)
						require.Equal(t, keycloakApi.KeycloakAdminTypeServiceAccount, adminType)
						return mockClient, nil
					},
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "error resolving auth spec credentials",
			authData: &KeycloakAuthData{
				Url:             "https://test.keycloak.com",
				SecretNamespace: "default",
				KeycloakCRName:  "test-keycloak",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							SourceRef: common.SourceRef{
								SecretKeyRef: &common.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "non-existent-secret",
									},
									Key: "username",
								},
							},
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "password-secret",
							},
							Key: "password",
						},
					},
				},
			},
			setupHelper: func(t *testing.T) *Helper {
				return &Helper{
					client:          fake.NewClientBuilder().Build(),
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to resolve username")
			},
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

func TestHelper_createKeycloakClientV2FromAuthData_withServiceAccount(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	// Create mock server for successful auth cases
	mockServer := fakehttp.NewServerBuilder().
		AddKeycloakAuthResponders().
		BuildAndStart()
	defer mockServer.Close()

	tests := []struct {
		name      string
		authData  *KeycloakAuthData
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, client *keycloakclientv2.KeycloakClient)
	}{
		{
			name: "successfully create v2 client with service account admin type",
			authData: &KeycloakAuthData{
				Url:             mockServer.GetURL(),
				SecretName:      "keycloak-secret",
				SecretNamespace: "default",
				AdminType:       keycloakApi.KeycloakAdminTypeServiceAccount,
				KeycloakCRName:  "test-keycloak",
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("service-account-client"),
						"password": []byte("client-secret"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "successfully create v2 client with password grant auth spec",
			authData: &KeycloakAuthData{
				Url:             mockServer.GetURL(),
				SecretNamespace: "default",
				KeycloakCRName:  "test-keycloak",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							Value: "admin",
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "password-secret",
							},
							Key: "password",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "password-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "successfully create v2 client with client credentials auth spec",
			authData: &KeycloakAuthData{
				Url:             mockServer.GetURL(),
				SecretNamespace: "default",
				KeycloakCRName:  "test-keycloak",
				AuthSpec: &common.AuthSpec{
					ClientCredentials: &common.ClientCredentialsConfig{
						ClientID: common.SourceRefOrVal{
							Value: "admin-cli",
						},
						ClientSecretRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "client-secret",
							},
							Key: "secret",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "client-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"secret": []byte("client-secret-value"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "error when auth spec is invalid",
			authData: &KeycloakAuthData{
				Url:             mockServer.GetURL(),
				SecretNamespace: "default",
				KeycloakCRName:  "test-keycloak",
				AuthSpec:        &common.AuthSpec{},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "one of passwordGrant or clientCredentials must be set")
			},
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.Nil(t, client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			client, err := helper.createKeycloakClientV2FromAuthData(context.Background(), tt.authData)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, client)
			}
		})
	}
}

func TestHelper_CreateKeycloakClientV2FromRealmRef(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	// Create mock server for successful auth cases
	mockServer := fakehttp.NewServerBuilder().
		AddKeycloakAuthResponders().
		BuildAndStart()
	defer mockServer.Close()

	tests := []struct {
		name      string
		object    ObjectWithRealmRef
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, client *keycloakclientv2.KeycloakClient)
	}{
		{
			name: "successfully create v2 client from KeycloakRealm ref",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "test-realm",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test",
						KeycloakRef: common.KeycloakRef{
							Kind: keycloakApi.KeycloakKind,
							Name: "test-keycloak",
						},
					},
				},
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "successfully create v2 client from ClusterKeycloakRealm ref",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakAlpha.ClusterKeycloakRealmKind,
						Name: "test-cluster-realm",
					},
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-realm",
					},
					Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
						RealmName:          "test",
						ClusterKeycloakRef: "test-cluster-keycloak",
					},
				},
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "error when KeycloakRealm not found",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "non-existent-realm",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get realm")
			},
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "error when ClusterKeycloakRealm not found",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakAlpha.ClusterKeycloakRealmKind,
						Name: "non-existent-cluster-realm",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get cluster realm")
			},
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "error with unknown realm kind",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: "UnknownKind",
						Name: "test-realm",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unknown realm kind")
			},
			checkFunc: func(t *testing.T, client *keycloakclientv2.KeycloakClient) {
				require.Nil(t, client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			client, err := helper.CreateKeycloakClientV2FromRealmRef(ctx, tt.object)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, client)
			}
		})
	}
}

func TestHelper_getKeycloakAuthDataFromRealmRef(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name      string
		object    ObjectWithRealmRef
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, authData *KeycloakAuthData)
	}{
		{
			name: "successfully get auth data from KeycloakRealm ref",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "test-realm",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test",
						KeycloakRef: common.KeycloakRef{
							Kind: keycloakApi.KeycloakKind,
							Name: "test-keycloak",
						},
					},
				},
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    "https://keycloak.example.com",
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Equal(t, "https://keycloak.example.com", authData.Url)
				require.Equal(t, "keycloak-secret", authData.SecretName)
				require.Equal(t, "default", authData.SecretNamespace)
			},
		},
		{
			name: "successfully get auth data from ClusterKeycloakRealm ref",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakAlpha.ClusterKeycloakRealmKind,
						Name: "test-cluster-realm",
					},
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-realm",
					},
					Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
						RealmName:          "test",
						ClusterKeycloakRef: "test-cluster-keycloak",
					},
				},
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    "https://cluster-keycloak.example.com",
						Secret: "cluster-keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Equal(t, "https://cluster-keycloak.example.com", authData.Url)
				require.Equal(t, "cluster-keycloak-secret", authData.SecretName)
			},
		},
		{
			name: "error when realm not found and object being deleted returns ErrKeycloakRealmNotFound",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-group",
					Namespace:         "default",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{"test-finalizer"},
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "non-existent-realm",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.ErrorIs(t, err, ErrKeycloakRealmNotFound)
			},
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Nil(t, authData)
			},
		},
		{
			name: "error when cluster realm not found and object being deleted returns ErrKeycloakRealmNotFound",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-group",
					Namespace:         "default",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{"test-finalizer"},
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakAlpha.ClusterKeycloakRealmKind,
						Name: "non-existent-cluster-realm",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.ErrorIs(t, err, ErrKeycloakRealmNotFound)
			},
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Nil(t, authData)
			},
		},
		{
			name: "error with unknown realm kind",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: "UnknownKind",
						Name: "test-realm",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unknown realm kind")
			},
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Nil(t, authData)
			},
		},
		{
			name: "error when keycloak not connected",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "test-realm",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test",
						KeycloakRef: common.KeycloakRef{
							Kind: keycloakApi.KeycloakKind,
							Name: "test-keycloak",
						},
					},
				},
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    "https://keycloak.example.com",
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: false,
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.ErrorIs(t, err, ErrKeycloakIsNotAvailable)
			},
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Nil(t, authData)
			},
		},
		{
			name: "error when cluster keycloak not connected",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakAlpha.ClusterKeycloakRealmKind,
						Name: "test-cluster-realm",
					},
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-realm",
					},
					Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
						RealmName:          "test",
						ClusterKeycloakRef: "test-cluster-keycloak",
					},
				},
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    "https://cluster-keycloak.example.com",
						Secret: "cluster-keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: false,
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.ErrorIs(t, err, ErrKeycloakIsNotAvailable)
			},
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Nil(t, authData)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			authData, err := helper.getKeycloakAuthDataFromRealmRef(ctx, tt.object)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, authData)
			}
		})
	}
}

func TestHelper_getKeycloakAuthDataFromRealm_withClusterKeycloakKind(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name      string
		realm     *keycloakApi.KeycloakRealm
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, authData *KeycloakAuthData)
	}{
		{
			name: "successfully get auth data with ClusterKeycloak kind",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakAlpha.ClusterKeycloakKind,
						Name: "test-cluster-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    "https://cluster-keycloak.example.com",
						Secret: "cluster-keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Equal(t, "https://cluster-keycloak.example.com", authData.Url)
				require.Equal(t, "cluster-keycloak-secret", authData.SecretName)
			},
		},
		{
			name: "error when ClusterKeycloak not found",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakAlpha.ClusterKeycloakKind,
						Name: "non-existent-cluster-keycloak",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get cluster keycloak")
			},
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Nil(t, authData)
			},
		},
		{
			name: "error when ClusterKeycloak not connected",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakAlpha.ClusterKeycloakKind,
						Name: "test-cluster-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    "https://cluster-keycloak.example.com",
						Secret: "cluster-keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: false,
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.ErrorIs(t, err, ErrKeycloakIsNotAvailable)
			},
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Nil(t, authData)
			},
		},
		{
			name: "error with unknown keycloak kind",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: "UnknownKind",
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unknown keycloak kind")
			},
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Nil(t, authData)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			authData, err := helper.getKeycloakAuthDataFromRealm(ctx, tt.realm)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, authData)
			}
		})
	}
}

func TestHelper_CreateKeycloakClientFomAuthData(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddStringResponder("/auth/admin/realms/", "{}").
		BuildAndStart()
	defer mockServer.Close()

	tests := []struct {
		name        string
		authData    *KeycloakAuthData
		setupHelper func(t *testing.T) *Helper
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name: "successful creation from login password when no token secret exists",
			authData: &KeycloakAuthData{
				Url:             mockServer.GetURL(),
				SecretName:      "keycloak-admin-secret",
				SecretNamespace: "default",
				AdminType:       keycloakApi.KeycloakAdminTypeUser,
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
			name: "login password creation failure",
			authData: &KeycloakAuthData{
				Url:             mockServer.GetURL(),
				SecretName:      "non-existent-secret",
				SecretNamespace: "default",
				AdminType:       keycloakApi.KeycloakAdminTypeUser,
				KeycloakCRName:  "test-keycloak",
			},
			setupHelper: func(t *testing.T) *Helper {
				return &Helper{
					client:          fake.NewClientBuilder().Build(),
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to create kc client from login password")
			},
		},
		{
			name: "token secret exists but contains invalid token",
			authData: &KeycloakAuthData{
				Url:             mockServer.GetURL(),
				SecretName:      "keycloak-admin-secret",
				SecretNamespace: "default",
				AdminType:       keycloakApi.KeycloakAdminTypeUser,
				KeycloakCRName:  "test-keycloak",
			},
			setupHelper: func(t *testing.T) *Helper {
				return &Helper{
					client: fake.NewClientBuilder().WithRuntimeObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      tokenSecretName("test-keycloak"),
								Namespace: "default",
							},
							Data: map[string][]byte{
								keycloakTokenSecretKey: []byte("invalid-json-token"),
							},
						},
					).Build(),
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to create kc client from token secret")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := tt.setupHelper(t)
			ctx := context.Background()

			_, err := helper.CreateKeycloakClientFomAuthData(ctx, tt.authData)
			tt.wantErr(t, err)
		})
	}
}

func TestHelper_CreateKeycloakClientFromRealmRef(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddStringResponder("/auth/admin/realms/", "{}").
		BuildAndStart()
	defer mockServer.Close()

	tests := []struct {
		name        string
		object      ObjectWithRealmRef
		setupHelper func(t *testing.T, objects []client.Object) *Helper
		objects     []client.Object
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name: "successful creation from KeycloakRealm ref",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "test-realm",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test",
						KeycloakRef: common.KeycloakRef{
							Kind: keycloakApi.KeycloakKind,
							Name: "test-keycloak",
						},
					},
				},
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			setupHelper: func(t *testing.T, objects []client.Object) *Helper {
				mockClient := mocks.NewMockClient(t)
				mockClient.On("ExportToken").Return([]byte(`{"access_token":"mock-token"}`), nil)

				return &Helper{
					client:            fake.NewClientBuilder().WithScheme(s).WithObjects(objects...).Build(),
					operatorNamespace: "default",
					adapterBuilder: func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger, restyClient *resty.Client) (keycloak.Client, error) {
						return mockClient, nil
					},
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "error when realm not found",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "non-existent-realm",
					},
				},
			},
			objects: []client.Object{},
			setupHelper: func(t *testing.T, objects []client.Object) *Helper {
				return &Helper{
					client:            fake.NewClientBuilder().WithScheme(s).WithObjects(objects...).Build(),
					operatorNamespace: "default",
					tokenSecretLock:   &sync.Mutex{},
				}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := tt.setupHelper(t, tt.objects)
			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())

			_, err := helper.CreateKeycloakClientFromRealmRef(ctx, tt.object)
			tt.wantErr(t, err)
		})
	}
}

func TestHelper_CreateKeycloakClientFromRealm(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddStringResponder("/auth/admin/realms/", "{}").
		BuildAndStart()
	defer mockServer.Close()

	tests := []struct {
		name        string
		realm       *keycloakApi.KeycloakRealm
		setupHelper func(t *testing.T, objects []client.Object) *Helper
		objects     []client.Object
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name: "successful creation from realm",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			setupHelper: func(t *testing.T, objects []client.Object) *Helper {
				mockClient := mocks.NewMockClient(t)
				mockClient.On("ExportToken").Return([]byte(`{"access_token":"mock-token"}`), nil)

				return &Helper{
					client:            fake.NewClientBuilder().WithScheme(s).WithObjects(objects...).Build(),
					operatorNamespace: "default",
					adapterBuilder: func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger, restyClient *resty.Client) (keycloak.Client, error) {
						return mockClient, nil
					},
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "error when keycloak not found",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "non-existent-keycloak",
					},
				},
			},
			objects: []client.Object{},
			setupHelper: func(t *testing.T, objects []client.Object) *Helper {
				return &Helper{
					client:            fake.NewClientBuilder().WithScheme(s).WithObjects(objects...).Build(),
					operatorNamespace: "default",
					tokenSecretLock:   &sync.Mutex{},
				}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := tt.setupHelper(t, tt.objects)
			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())

			_, err := helper.CreateKeycloakClientFromRealm(ctx, tt.realm)
			tt.wantErr(t, err)
		})
	}
}

func TestHelper_CreateKeycloakClientFromClusterRealm(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddStringResponder("/auth/admin/realms/", "{}").
		BuildAndStart()
	defer mockServer.Close()

	tests := []struct {
		name        string
		realm       *keycloakAlpha.ClusterKeycloakRealm
		setupHelper func(t *testing.T, objects []client.Object) *Helper
		objects     []client.Object
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name: "successful creation from cluster realm",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "test-cluster-keycloak",
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			setupHelper: func(t *testing.T, objects []client.Object) *Helper {
				mockClient := mocks.NewMockClient(t)
				mockClient.On("ExportToken").Return([]byte(`{"access_token":"mock-token"}`), nil)

				return &Helper{
					client:            fake.NewClientBuilder().WithScheme(s).WithObjects(objects...).Build(),
					operatorNamespace: "default",
					adapterBuilder: func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger, restyClient *resty.Client) (keycloak.Client, error) {
						return mockClient, nil
					},
					tokenSecretLock: &sync.Mutex{},
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "error when cluster keycloak not found",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "non-existent-cluster-keycloak",
				},
			},
			objects: []client.Object{},
			setupHelper: func(t *testing.T, objects []client.Object) *Helper {
				return &Helper{
					client:            fake.NewClientBuilder().WithScheme(s).WithObjects(objects...).Build(),
					operatorNamespace: "default",
					tokenSecretLock:   &sync.Mutex{},
				}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := tt.setupHelper(t, tt.objects)
			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())

			_, err := helper.CreateKeycloakClientFromClusterRealm(ctx, tt.realm)
			tt.wantErr(t, err)
		})
	}
}
