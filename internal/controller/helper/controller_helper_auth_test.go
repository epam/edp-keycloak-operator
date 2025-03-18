package helper

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
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

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakApiAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/fakehttp"
)

func TestCreateKeycloakClientFromLoginPassword_FailureExportToken(t *testing.T) {
	s := scheme.Scheme
	utilruntime.Must(keycloakApi.AddToScheme(s))

	kc := keycloakApi.Keycloak{
		Spec: keycloakApi.KeycloakSpec{
			Secret: "test",
		},
	}
	lpSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: kc.Spec.Secret,
		},
		Data: map[string][]byte{
			"username": []byte("username"),
			"password": []byte("password"),
		},
	}

	cl := fake.NewClientBuilder().WithRuntimeObjects(&kc, &lpSecret).Build()

	helper := MakeHelper(cl, s, "default")
	adapterMock := mocks.NewMockClient(t)
	adapterMock.On("ExportToken").Return(nil, errors.New("export token fatal"))

	helper.adapterBuilder = func(ctx context.Context, conf adapter.GoCloakConfig, adminType string, log logr.Logger,
		restyClient *resty.Client) (keycloak.Client, error) {
		return adapterMock, nil
	}

	auth, err := MakeKeycloakAuthDataFromKeycloak(context.Background(), &kc, cl)
	require.NoError(t, err)

	_, err = helper.createKeycloakClientFromLoginPassword(context.Background(), auth)
	if err == nil {
		t.Fatal("no error on token export")
	}

	if !strings.Contains(err.Error(), "export token fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCreateKeycloakClientFromLoginPassword(t *testing.T) {
	s := scheme.Scheme
	utilruntime.Must(keycloakApi.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddStringResponder("/auth/admin/realms/", "{}").
		AddStringResponder("/auth/realms/master/protocol/openid-connect/token", "{}").
		BuildAndStart()
	defer mockServer.Close()

	kc := keycloakApi.Keycloak{
		Spec: keycloakApi.KeycloakSpec{
			Url:    mockServer.GetURL(),
			Secret: "test",
		},
	}

	lpSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: kc.Spec.Secret,
		},
		Data: map[string][]byte{
			"username": []byte("username"),
			"password": []byte("password"),
		},
	}

	cl := fake.NewClientBuilder().WithRuntimeObjects(&kc, &lpSecret).Build()

	helper := MakeHelper(cl, s, "default")
	helper.restyClient = resty.New()

	auth, err := MakeKeycloakAuthDataFromKeycloak(context.Background(), &kc, cl)
	require.NoError(t, err)

	_, err = helper.createKeycloakClientFromLoginPassword(context.Background(), auth)
	if err != nil {
		t.Fatalf("%+v", err)
	}
}

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

	if !k8sErrors.IsNotFound(errors.Cause(err)) {
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
	t.Parallel()

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
			t.Parallel()

			got, err := MakeKeycloakAuthDataFromKeycloak(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.keycloak, tt.k8sClient(t))

			tt.wantErr(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMakeKeycloakAuthDataFromClusterKeycloak(t *testing.T) {
	t.Parallel()

	s := runtime.NewScheme()
	require.NoError(t, keycloakApiAlpha.AddToScheme(s))
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
			t.Parallel()

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
