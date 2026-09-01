package helper

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

// Status carries the connection result; an unchanged status is not written, so the
// resourceVersion stays put and no watch event fires.
func TestUpdateConnectionStatus(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	kc := &keycloakApi.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "kc", Namespace: "default"}}
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&keycloakApi.Keycloak{}).
		WithObjects(kc).
		Build()

	ctx := ctrl.LoggerInto(context.Background(), logr.Discard())

	get := func() *keycloakApi.Keycloak {
		out := &keycloakApi.Keycloak{}
		require.NoError(t, cl.Get(ctx, client.ObjectKey{Name: "kc", Namespace: "default"}, out))

		return out
	}

	update := func(obj *keycloakApi.Keycloak, connectErr error) error {
		return UpdateConnectionStatus(ctx, cl, obj, &obj.Status.Connected, &obj.Status.Value, connectErr)
	}

	require.NoError(t, update(get(), assert.AnError))

	afterFailure := get()
	assert.False(t, afterFailure.Status.Connected)
	assert.Contains(t, afterFailure.Status.Value, assert.AnError.Error())

	require.NoError(t, update(get(), assert.AnError))
	assert.Equal(t, afterFailure.ResourceVersion, get().ResourceVersion,
		"an unchanged status must not be written")

	require.NoError(t, update(get(), nil))

	afterSuccess := get()
	assert.True(t, afterSuccess.Status.Connected)
	assert.Equal(t, common.StatusOK, afterSuccess.Status.Value)
	assert.NotEqual(t, afterFailure.ResourceVersion, afterSuccess.ResourceVersion)
}
