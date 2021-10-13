package helper

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHelper_CreateKeycloakClientForRealm(t *testing.T) {
	mc := Client{}

	utilruntime.Must(v1alpha1.AddToScheme(scheme.Scheme))
	helper := MakeHelper(&mc, scheme.Scheme, nil)
	realm := v1alpha1.KeycloakRealm{
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

	kc := v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "testOwnerReference"},
		Status:     v1alpha1.KeycloakStatus{Connected: true},
		Spec:       v1alpha1.KeycloakSpec{Secret: "ss1"},
	}

	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(&kc).Build()

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "testOwnerReference",
	}, &v1alpha1.Keycloak{}).Return(fakeCl)

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "kc-token-testOwnerReference",
	}, &v1.Secret{}).Return(errors.New("FATAL"))

	_, err := helper.CreateKeycloakClientForRealm(context.Background(), &realm)
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "FATAL") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCreateKeycloakClientFromLoginPassword_FailureExportToken(t *testing.T) {
	s := scheme.Scheme
	utilruntime.Must(v1alpha1.AddToScheme(s))

	kc := v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
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

	helper := MakeHelper(cl, s, nil)
	adapterMock := adapter.Mock{
		ExportTokenErr: errors.New("export token fatal"),
	}
	helper.adapterBuilder = func(ctx context.Context, url, user, password string, log logr.Logger,
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
	utilruntime.Must(v1alpha1.AddToScheme(s))

	kc := v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
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
	httpmock.Activate()
	httpmock.RegisterResponder("POST", "/auth/realms/master/protocol/openid-connect/token",
		httpmock.NewStringResponder(200, `{}`))
	cl := fake.NewClientBuilder().WithRuntimeObjects(&kc, &lpSecret).Build()

	helper := MakeHelper(cl, s, nil)
	helper.restyClient = resty.New()
	httpmock.ActivateNonDefault(helper.restyClient.GetClient())

	_, err := helper.CreateKeycloakClientFromLoginPassword(context.Background(), &kc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestHelper_SaveKeycloakClientTokenSecret(t *testing.T) {
	s := scheme.Scheme
	utilruntime.Must(v1alpha1.AddToScheme(s))
	kc := v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
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

	if err := h.SaveKeycloakClientTokenSecret(context.Background(), &kc, []byte("token")); err != nil {
		t.Fatal(err)
	}
}

func TestHelper_SaveKeycloakClientTokenSecret_Failures(t *testing.T) {
	s := scheme.Scheme
	utilruntime.Must(v1alpha1.AddToScheme(s))
	kc := v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
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
	mc := Client{}
	mc.On("Get", types.NamespacedName{Name: tokenSecretName(kc.Name)}, &corev1.Secret{}).
		Return(errors.New("fatal secret"))

	h := Helper{
		client: &mc,
	}

	err := h.SaveKeycloakClientTokenSecret(context.Background(), &kc, []byte("token"))
	if err == nil {
		t.Fatal(err)
	}
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
	if err == nil {
		t.Fatal(err)
	}
	if !strings.Contains(err.Error(), "secret update fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestHelper_CreateKeycloakClientFromTokenSecret(t *testing.T) {
	s := scheme.Scheme
	utilruntime.Must(v1alpha1.AddToScheme(s))
	kc := v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			Secret: "test",
		},
	}

	realToken := `eyJhbGciOiJIUzI1NiJ9.eyJSb2xlIjoiQWRtaW4iLCJJc3N1ZXIiOiJJc3N1ZXIiLCJVc2VybmFtZSI6IkphdmFJblVzZSIsImV4cCI6MTYzNDAzOTA2OCwiaWF0IjoxNjM0MDM5MDY4fQ.OZJDXUqfmajSh0vpqL8VnoQGqUXH25CAVkKnoyJX3AI`
	tok := gocloak.JWT{AccessToken: realToken}
	bts, _ := json.Marshal(&tok)

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

	_, err := h.CreateKeycloakClientFromTokenSecret(context.Background(), &kc)
	if err == nil {
		t.Fatal("no error on expired token")
	}
	if !strings.Contains(err.Error(), "token is expired") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	tokenParts := strings.Split(realToken, ".")
	rawTokenPayload, _ := base64.RawURLEncoding.DecodeString(tokenParts[1])
	var decodedTokenPayload adapter.JWTPayload
	_ = json.Unmarshal(rawTokenPayload, &decodedTokenPayload)
	decodedTokenPayload.Exp = time.Now().Unix() + 1000
	rawTokenPayload, _ = json.Marshal(decodedTokenPayload)
	tokenParts[1] = base64.RawURLEncoding.EncodeToString(rawTokenPayload)
	realToken = strings.Join(tokenParts, ".")

	tok = gocloak.JWT{AccessToken: realToken}
	bts, _ = json.Marshal(&tok)
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

	if _, err := h.CreateKeycloakClientFromTokenSecret(context.Background(), &kc); err != nil {
		t.Fatal(err)
	}
}
