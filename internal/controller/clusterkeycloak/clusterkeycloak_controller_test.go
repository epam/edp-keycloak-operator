package clusterkeycloak

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
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type stubHelper struct{ err error }

func (s stubHelper) CreateKeycloakClientFromClusterKeycloak(
	context.Context,
	*keycloakAlpha.ClusterKeycloak,
) (*keycloakapi.KeycloakClient, error) {
	return nil, s.err
}

// Status carries the connection result; an unchanged status is not written, so the
// resourceVersion stays put and no watch event fires.
func TestUpdateConnectionStatusToKeycloak(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakAlpha.AddToScheme(scheme))

	kc := &keycloakAlpha.ClusterKeycloak{ObjectMeta: metav1.ObjectMeta{Name: "kc"}}
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&keycloakAlpha.ClusterKeycloak{}).
		WithObjects(kc).
		Build()

	ctx := ctrl.LoggerInto(context.Background(), logr.Discard())

	get := func() *keycloakAlpha.ClusterKeycloak {
		out := &keycloakAlpha.ClusterKeycloak{}
		require.NoError(t, cl.Get(ctx, client.ObjectKey{Name: "kc"}, out))

		return out
	}

	failing := NewReconcile(cl, scheme, stubHelper{err: assert.AnError})
	require.NoError(t, failing.updateConnectionStatusToKeycloak(ctx, get()))

	afterFailure := get()
	assert.False(t, afterFailure.Status.Connected)
	assert.Contains(t, afterFailure.Status.Value, assert.AnError.Error())

	require.NoError(t, failing.updateConnectionStatusToKeycloak(ctx, get()))
	assert.Equal(t, afterFailure.ResourceVersion, get().ResourceVersion,
		"an unchanged status must not be written")

	succeeding := NewReconcile(cl, scheme, stubHelper{})
	require.NoError(t, succeeding.updateConnectionStatusToKeycloak(ctx, get()))

	afterSuccess := get()
	assert.True(t, afterSuccess.Status.Connected)
	assert.Equal(t, common.StatusOK, afterSuccess.Status.Value)
	assert.NotEqual(t, afterFailure.ResourceVersion, afterSuccess.ResourceVersion)
}
