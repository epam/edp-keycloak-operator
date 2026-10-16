package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

func TestFindOwnerCR_SharedURLIsConflict(t *testing.T) {
	self := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameB, Namespace: "ns-a"},
	}
	self.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	self.Spec.RealmRef.Name = testRealmRefName

	owner := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: "ns-b"},
		Status:     keycloakApi.KeycloakRealmGroupStatus{ID: "gid"},
	}
	owner.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	owner.Spec.RealmRef.Name = testRealmRefName

	objs := make([]client.Object, 0, 5)
	objs = append(objs, owner)
	objs = append(objs, namespacedKeycloakStack("ns-a", testRealmRefName, "http://shared.example")...)
	objs = append(objs, namespacedKeycloakStack("ns-b", testRealmRefName, "http://shared.example/")...)

	got, err := findOwnerCR(context.Background(), newFakeK8sClient(t, objs...), self, "gid")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "ns-b", got.Namespace)
	assert.Equal(t, testCRNameA, got.Name)
}

func TestFindOwnerCR_CloneDifferentURLIsNotConflict(t *testing.T) {
	self := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameB, Namespace: "ns-a"},
	}
	self.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	self.Spec.RealmRef.Name = testRealmRefName

	owner := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: "ns-b"},
		Status:     keycloakApi.KeycloakRealmGroupStatus{ID: "gid"},
	}
	owner.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	owner.Spec.RealmRef.Name = testRealmRefName

	objs := make([]client.Object, 0, 5)
	objs = append(objs, owner)
	objs = append(objs, namespacedKeycloakStack("ns-a", testRealmRefName, "http://kc-a.example")...)
	objs = append(objs, namespacedKeycloakStack("ns-b", testRealmRefName, "http://kc-b.example")...)

	got, err := findOwnerCR(context.Background(), newFakeK8sClient(t, objs...), self, "gid")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFindOwnerCR_ResolutionFailure(t *testing.T) {
	self := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameB, Namespace: "ns-a"},
	}
	self.Spec.RealmRef.Name = testRealmRefName

	owner := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: "ns-b"},
		Status:     keycloakApi.KeycloakRealmGroupStatus{ID: "gid"},
	}
	owner.Spec.RealmRef.Name = testRealmRefName

	_, err := findOwnerCR(context.Background(), newFakeK8sClient(t, owner), self, "gid")
	assert.ErrorContains(t, err, "unable to resolve realm target for ownership check")
}
