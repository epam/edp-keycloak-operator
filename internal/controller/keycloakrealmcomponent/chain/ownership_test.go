package chain

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestFindOwnerCR_SameRealmRefIsConflict(t *testing.T) {
	self := ownerComponent(testNamespace, "self", "")
	owner := ownerComponent(testNamespace, "owner", testComponentID)

	got, err := findOwnerCR(context.Background(), newFakeK8sClient(t, owner), self, testComponentID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "owner", got.Name)
}

func TestFindOwnerCR_SelfIsSkipped(t *testing.T) {
	self := ownerComponent(testNamespace, "self", testComponentID)

	got, err := findOwnerCR(context.Background(), newFakeK8sClient(t, self.DeepCopy()), self, testComponentID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFindOwnerCR_EmptyIDIsNoOwner(t *testing.T) {
	self := ownerComponent(testNamespace, "self", "")
	other := ownerComponent(testNamespace, "other", "")

	got, err := findOwnerCR(context.Background(), newFakeK8sClient(t, other), self, "")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFindOwnerCR_SharedURLIsConflict(t *testing.T) {
	self := ownerComponent("ns-a", "self", "")
	owner := ownerComponent("ns-b", "owner", testComponentID)

	objs := slices.Concat(
		[]client.Object{owner},
		namespacedKeycloakStack("ns-a", "http://shared.example"),
		namespacedKeycloakStack("ns-b", "http://shared.example/"),
	)

	got, err := findOwnerCR(context.Background(), newFakeK8sClient(t, objs...), self, testComponentID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "ns-b", got.Namespace)
	assert.Equal(t, "owner", got.Name)
}

func TestFindOwnerCR_CloneDifferentURLIsNotConflict(t *testing.T) {
	self := ownerComponent("ns-a", "self", "")
	owner := ownerComponent("ns-b", "owner", testComponentID)

	objs := slices.Concat(
		[]client.Object{owner},
		namespacedKeycloakStack("ns-a", "http://kc-a.example"),
		namespacedKeycloakStack("ns-b", "http://kc-b.example"),
	)

	got, err := findOwnerCR(context.Background(), newFakeK8sClient(t, objs...), self, testComponentID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFindOwnerCR_CandidateResolutionFailureNamesCandidate(t *testing.T) {
	self := ownerComponent("ns-a", "self", "")
	owner := ownerComponent("ns-b", "owner", testComponentID)

	_, err := findOwnerCR(context.Background(), newFakeK8sClient(t, owner), self, testComponentID)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unable to resolve realm target of KeycloakRealmComponent ns-b/owner for ownership check")
}

func TestFindOwnerCR_SelfResolutionFailureWithResolvableCandidate(t *testing.T) {
	self := ownerComponent("ns-a", "self", "")
	owner := ownerComponent("ns-b", "owner", testComponentID)

	objs := slices.Concat([]client.Object{owner}, namespacedKeycloakStack("ns-b", "http://kc-b.example"))

	got, err := findOwnerCR(context.Background(), newFakeK8sClient(t, objs...), self, testComponentID)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.ErrorContains(t, err, "unable to resolve realm target for ownership check")
}

func TestFindOwnerCR_ListError(t *testing.T) {
	self := ownerComponent(testNamespace, "self", "")

	k8sClient := fake.NewClientBuilder().
		WithScheme(newScheme(t)).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				return errors.New("list failed")
			},
		}).
		Build()

	_, err := findOwnerCR(context.Background(), k8sClient, self, testComponentID)
	assert.ErrorContains(t, err, "unable to list KeycloakRealmComponent resources")
}
