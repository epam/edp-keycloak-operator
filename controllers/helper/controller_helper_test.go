package helper

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/fakehttp"
)

func TestMakeHelper(t *testing.T) {
	rCl := resty.New()

	mockServer := fakehttp.NewServerBuilder().
		AddStringResponder("/auth/realms/master/protocol/openid-connect/token/", "{}").
		BuildAndStart()
	defer mockServer.Close()

	logger := mock.NewLogr()
	h := MakeHelper(nil, nil, "default")
	_, err := h.adapterBuilder(
		context.Background(),
		adapter.GoCloakConfig{
			Url:      mockServer.GetURL(),
			User:     "foo",
			Password: "bar",
		},
		keycloakApi.KeycloakAdminTypeServiceAccount,
		logger,
		rCl,
	)
	require.NoError(t, err)
}

type testTerminator struct {
	err error
	log logr.Logger
}

func (t *testTerminator) DeleteResource(ctx context.Context) error {
	return t.err
}
func (t *testTerminator) GetLogger() logr.Logger {
	return t.log
}

func TestHelper_TryToDelete(t *testing.T) {
	logger := mock.NewLogr()

	term := testTerminator{
		log: logger,
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-secret1"}}
	fakeClient := fake.NewClientBuilder().WithRuntimeObjects(&secret).Build()
	h := Helper{client: fakeClient}

	_, err := h.TryToDelete(context.Background(), &secret, &term, "fin")
	require.NoError(t, err)

	term.err = errors.New("delete resource fatal")
	secret.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	_, err = h.TryToDelete(context.Background(), &secret, &term, "fin")
	require.Error(t, err)

	if err.Error() != "error during keycloak resource deletion: delete resource fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
