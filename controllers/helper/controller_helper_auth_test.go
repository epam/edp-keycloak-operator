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
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/fakehttp"
)

func TestHelper_CreateKeycloakClientForRealm(t *testing.T) {
	mc := K8SClientMock{}

	utilruntime.Must(keycloakApi.AddToScheme(scheme.Scheme))
	helper := MakeHelper(&mc, scheme.Scheme, mock.NewLogr())
	realm := keycloakApi.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "testOwnerReference",
					Kind: "Keycloak",
				},
			},
		},
	}

	kc := keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "testOwnerReference"},
		Status:     keycloakApi.KeycloakStatus{Connected: true},
		Spec:       keycloakApi.KeycloakSpec{Secret: "ss1"},
	}

	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(&kc).Build()

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "testOwnerReference",
	}, &keycloakApi.Keycloak{}).Return(fakeCl)

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "kc-token-testOwnerReference",
	}, &v1.Secret{}).Return(errors.New("FATAL"))

	_, err := helper.CreateKeycloakClientForRealm(context.Background(), &realm)
	require.Error(t, err)

	if !strings.Contains(err.Error(), "FATAL") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

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

	helper := MakeHelper(cl, s, mock.NewLogr())
	adapterMock := adapter.Mock{
		ExportTokenErr: errors.New("export token fatal"),
	}
	helper.adapterBuilder = func(ctx context.Context, url, user, password, adminType string, log logr.Logger,
		restyClient *resty.Client) (keycloak.Client, error) {
		return &adapterMock, nil
	}

	_, err := helper.CreateKeycloakClientFromLoginPassword(context.Background(), &kc)
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

	helper := MakeHelper(cl, s, mock.NewLogr())
	helper.restyClient = resty.New()

	_, err := helper.CreateKeycloakClientFromLoginPassword(context.Background(), &kc)
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

	err := h.SaveKeycloakClientTokenSecret(context.Background(), &kc, []byte("token"))
	require.NoError(t, err)
}

func TestHelper_SaveKeycloakClientTokenSecret_Failures(t *testing.T) {
	s := scheme.Scheme
	utilruntime.Must(keycloakApi.AddToScheme(s))

	kc := keycloakApi.Keycloak{
		Spec: keycloakApi.KeycloakSpec{
			Secret: "test",
		},
	}

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: tokenSecretName("test2"),
		},
	}
	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(&kc, &secret).Build()
	mc := K8SClientMock{}
	mc.On("Get", types.NamespacedName{Name: tokenSecretName(kc.Name)}, &corev1.Secret{}).
		Return(errors.New("fatal secret"))

	h := Helper{
		client: &mc,
	}

	err := h.SaveKeycloakClientTokenSecret(context.Background(), &kc, []byte("token"))
	require.Error(t, err)

	if !strings.Contains(err.Error(), "fatal secret") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	kc.Name = "test2"

	mc.On("Get", types.NamespacedName{Name: tokenSecretName(kc.Name)}, &corev1.Secret{}).Return(fakeCl)

	secret.Data = map[string][]byte{
		keycloakTokenSecretKey: []byte("token"),
	}

	var updateOpts []client.UpdateOption

	mc.On("Update", &secret, updateOpts).Return(errors.New("secret update fatal"))

	err = h.SaveKeycloakClientTokenSecret(context.Background(), &kc, []byte("token"))
	require.Error(t, err)

	if !strings.Contains(err.Error(), "secret update fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
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

	_, err = h.CreateKeycloakClientFromTokenSecret(context.Background(), &kc)
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

	_, err = h.CreateKeycloakClientFromTokenSecret(context.Background(), &kc)
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
