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

// exfilSink stands in for the attacker-controlled endpoint from the advisory's proof of concept.
func exfilSink(t *testing.T, received *string) *httptest.Server {
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

func exfilKeycloakCR(url string) *keycloakApi.Keycloak {
	return &keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "exfil", Namespace: "default"},
		Spec: keycloakApi.KeycloakSpec{
			Url: url,
			Auth: &common.AuthSpec{
				PasswordGrant: &common.PasswordGrantConfig{
					Username: common.SourceRefOrVal{Value: "admin"},
					PasswordRef: common.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "victim"},
						Key:                  "adminPassword",
					},
				},
			},
		},
	}
}

func exfilHelper(t *testing.T, kc *keycloakApi.Keycloak, guard *destination.Guard) *Helper {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	victim := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "victim", Namespace: "default"},
		Data:       map[string][]byte{"adminPassword": []byte("LeakedSecret123!")},
	}

	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(victim, kc).Build()

	h, err := MakeHelper(cl, s, "operator-ns", guard)
	require.NoError(t, err)

	return h
}

// The advisory's exploit: a Keycloak CR pointing spec.url at an attacker endpoint while naming a
// Secret its author cannot read. With enforcement on, the credential must never leave the process.
func TestCreateKeycloakClientFromKeycloak_DeniesUnlistedDestination(t *testing.T) {
	t.Parallel()

	var received string

	sink := exfilSink(t, &received)
	kc := exfilKeycloakCR(sink.URL)

	guard, err := destination.New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	_, err = exfilHelper(t, kc, guard).CreateKeycloakClientFromKeycloak(context.Background(), kc)

	require.ErrorIs(t, err, destination.ErrNotAllowed)
	assert.Empty(t, received, "no request may reach an unlisted destination")
}

// The deprecated spec.secret path leaks the same way and is covered by the same check.
func TestCreateKeycloakClientFromKeycloak_DeniesUnlistedDestinationForLegacySecret(t *testing.T) {
	t.Parallel()

	var received string

	sink := exfilSink(t, &received)

	kc := &keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "exfil-legacy", Namespace: "default"},
		Spec:       keycloakApi.KeycloakSpec{Url: sink.URL, Secret: "victim"},
	}

	guard, err := destination.New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	_, err = exfilHelper(t, kc, guard).CreateKeycloakClientFromKeycloak(context.Background(), kc)

	require.ErrorIs(t, err, destination.ErrNotAllowed)
	assert.Empty(t, received, "no request may reach an unlisted destination")
}

// Warn mode must not deny; only enforce mode blocks the request.
func TestCreateKeycloakClientFromKeycloak_WarnModeStillConnects(t *testing.T) {
	t.Parallel()

	var received string

	sink := exfilSink(t, &received)
	kc := exfilKeycloakCR(sink.URL)

	guard, err := destination.New([]string{"keycloak.example.com"}, false)
	require.NoError(t, err)

	_, err = exfilHelper(t, kc, guard).CreateKeycloakClientFromKeycloak(context.Background(), kc)

	require.NoError(t, err)
	assert.Contains(t, received, "password=LeakedSecret123", "warn mode must not change behaviour")
}
