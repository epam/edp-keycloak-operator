package keycloakclient

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	helpermock "github.com/epam/edp-keycloak-operator/internal/controller/helper/mocks"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclient/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/destination"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

// Chain failure persists the incremented failureCount and returns a nil error with RequeueAfter from SetFailureCount.
func TestReconcile_ChainFailureRequeuesAfterBackoff(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	key := client.ObjectKey{Namespace: "default", Name: "client"}
	kc := &keycloakApi.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name},
		Spec: keycloakApi.KeycloakClientSpec{
			ClientId: "client",
			Secret:   secretref.GenerateSecretRef("missing-secret", "key"),
		},
		Status: keycloakApi.KeycloakClientStatus{FailureCount: 2},
	}
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&keycloakApi.KeycloakClient{}).
		WithObjects(kc).
		Build()

	secretRef, err := secretref.NewSecretRef(cl, destination.AllowAll())
	require.NoError(t, err)

	const backoff = 30 * time.Second

	h := helpermock.NewMockControllerHelper(t)
	h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
	h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).Return(&keycloakapi.KeycloakClient{}, nil)
	h.EXPECT().GetRealmNameFromRef(mock.Anything, mock.Anything).Return("realm", nil)
	h.EXPECT().SetFailureCount(mock.Anything).RunAndReturn(func(fc helper.FailureCountable) time.Duration {
		fc.SetFailureCount(fc.GetFailureCount() + 1)

		return backoff
	})

	ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
	r := NewReconcileKeycloakClient(cl, h, secretRef)

	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	assert.Equal(t, backoff, res.RequeueAfter)

	got := &keycloakApi.KeycloakClient{}
	require.NoError(t, cl.Get(ctx, key, got))
	assert.Equal(t, int64(3), got.Status.FailureCount)
	assert.Contains(t, got.Status.Value, "missing-secret")

	ready := meta.FindStatusCondition(got.Status.Conditions, chain.ConditionReady)
	require.NotNil(t, ready)
	assert.Equal(t, metav1.ConditionFalse, ready.Status)
	assert.Contains(t, ready.Message, "missing-secret")
}
