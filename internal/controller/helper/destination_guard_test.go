package helper

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/destination"
)

// recordingServer records the body of every request it receives.
func recordingServer(t *testing.T, received *string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*received = string(b)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"x","refresh_token":"y","token_type":"bearer","expires_in":3600}`))
	}))
	t.Cleanup(srv.Close)

	return srv
}

// keycloakCRWithPasswordGrant names url as the destination and a Secret as the credential.
func keycloakCRWithPasswordGrant(url string) *keycloakApi.Keycloak {
	return &keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "kc", Namespace: "default"},
		Spec: keycloakApi.KeycloakSpec{
			Url: url,
			Auth: &common.AuthSpec{
				PasswordGrant: &common.PasswordGrantConfig{
					Username: common.SourceRefOrVal{Value: "admin"},
					PasswordRef: common.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "referenced"},
						Key:                  "adminPassword",
					},
				},
			},
		},
	}
}

func guardedHelper(t *testing.T, kc *keycloakApi.Keycloak, guard *destination.Guard) *Helper {
	t.Helper()

	return guardedHelperWithPassword(t, kc, guard, "S3cretValue!")
}

// The referenced Secret stands for one the custom resource author cannot read directly.
func guardedHelperWithPassword(t *testing.T, kc *keycloakApi.Keycloak, guard *destination.Guard, password string) *Helper {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	referenced := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "referenced", Namespace: "default"},
		Data:       map[string][]byte{"adminPassword": []byte(password), "username": []byte("admin")},
	}

	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(referenced, kc).Build()

	h, err := MakeHelper(cl, s, "operator-ns", guard)
	require.NoError(t, err)

	return h
}

// An unlisted spec.url is denied before the credential is resolved, so it never leaves the process.
func TestCreateKeycloakClientFromKeycloak_DeniesUnlistedDestination(t *testing.T) {
	t.Parallel()

	var received string

	server := recordingServer(t, &received)
	kc := keycloakCRWithPasswordGrant(server.URL)

	guard, err := destination.New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	_, err = guardedHelper(t, kc, guard).CreateKeycloakClientFromKeycloak(context.Background(), kc)

	require.ErrorIs(t, err, destination.ErrNotAllowed)
	assert.Empty(t, received, "nothing may reach an unlisted destination")
}

// The deprecated spec.secret path uses the same check.
func TestCreateKeycloakClientFromKeycloak_DeniesUnlistedDestinationForLegacySecret(t *testing.T) {
	t.Parallel()

	var received string

	server := recordingServer(t, &received)

	kc := &keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "kc-legacy", Namespace: "default"},
		Spec:       keycloakApi.KeycloakSpec{Url: server.URL, Secret: "referenced"},
	}

	guard, err := destination.New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	_, err = guardedHelper(t, kc, guard).CreateKeycloakClientFromKeycloak(context.Background(), kc)

	require.ErrorIs(t, err, destination.ErrNotAllowed)
	assert.Empty(t, received, "nothing may reach an unlisted destination")
}

// Warn mode records the violation and permits the request; only enforce mode denies.
func TestCreateKeycloakClientFromKeycloak_WarnModeStillConnects(t *testing.T) {
	t.Parallel()

	var received string

	server := recordingServer(t, &received)
	kc := keycloakCRWithPasswordGrant(server.URL)

	guard, err := destination.New([]string{"keycloak.example.com"}, false)
	require.NoError(t, err)

	_, err = guardedHelper(t, kc, guard).CreateKeycloakClientFromKeycloak(context.Background(), kc)

	require.NoError(t, err)
	assert.Contains(t, received, "password=S3cretValue", "warn mode must not change behaviour")
}
