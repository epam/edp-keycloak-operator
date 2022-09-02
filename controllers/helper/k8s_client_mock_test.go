package helper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1/v1"
)

type fakePatch string

func (f fakePatch) Type() types.PatchType {
	return types.PatchType(f)
}

func (f fakePatch) Data(obj client.Object) ([]byte, error) {
	return []byte(f), nil
}

func TestK8SClientMock_OneLiners(t *testing.T) {
	k8sMock := K8SClientMock{}

	var (
		kc              keycloakApi.Keycloak
		kList           keycloakApi.KeycloakList
		createOpts      []client.CreateOption
		deleteOpts      []client.DeleteOption
		listOpts        []client.ListOption
		updateOpts      []client.UpdateOption
		patchOpts       []client.PatchOption
		deleteAllOfOpts []client.DeleteAllOfOption
		fPatch          fakePatch
	)
	ctx := context.Background()
	k8sMock.On("Create", &kc, createOpts).Return(nil)

	err := k8sMock.Create(ctx, &kc)
	require.NoError(t, err)

	k8sMock.On("List", &kList, listOpts).Return(nil)
	err = k8sMock.List(ctx, &kList)
	require.NoError(t, err)

	k8sMock.On("Delete", &kc, deleteOpts).Return(nil)
	err = k8sMock.Delete(ctx, &kc)
	require.NoError(t, err)

	k8sMock.On("Update", &kc, updateOpts).Return(nil)
	err = k8sMock.Update(ctx, &kc)
	require.NoError(t, err)

	k8sMock.On("Patch", &kc, fPatch, patchOpts).Return(nil)
	err = k8sMock.Patch(ctx, &kc, fPatch)
	require.NoError(t, err)

	k8sMock.On("DeleteAllOf", &kc, deleteAllOfOpts).Return(nil)
	err = k8sMock.DeleteAllOf(ctx, &kc)
	require.NoError(t, err)

}

func TestK8SClientMock_Status(t *testing.T) {
	s := K8SStatusMock{}
	k8sMock := K8SClientMock{}
	sch := runtime.Scheme{}

	k8sMock.On("Scheme").Return(&sch)
	if k8sMock.Scheme() == nil {
		t.Fatal("scheme must be not nil")
	}

	k8sMock.On("Status").Return(&s)
	if k8sMock.Status() == nil {
		t.Fatal("status must be not nil")
	}

	if k8sMock.RESTMapper() != nil {
		t.Fatal("rest mapper must be nil")
	}
}

func TestK8SClientMock_Get(t *testing.T) {
	k8sMock := K8SClientMock{}
	err := keycloakApi.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	var (
		kcRequest = keycloakApi.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "kc-name1", Namespace: "kc-ns"}}
		kcResult  = keycloakApi.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "kc-name1", Namespace: "kc-ns"},
			Status: keycloakApi.KeycloakStatus{Connected: true}}
	)

	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(&kcResult).Build()
	rq := types.NamespacedName{Name: kcRequest.Name, Namespace: kcRequest.Namespace}
	k8sMock.On("Get", rq, &kcRequest).Return(fakeCl)
	err = k8sMock.Get(context.Background(), rq, &kcRequest)
	require.NoError(t, err)

	if !kcRequest.Status.Connected {
		t.Fatal("kc status is not changed")
	}
}

func TestK8SStatusMock_OneLiners(t *testing.T) {
	var (
		status     K8SStatusMock
		updateOpts []client.UpdateOption
		patchOpts  []client.PatchOption
		ctx        = context.Background()
		kc         keycloakApi.Keycloak
		fPath      fakePatch
	)

	status.On("Update", &kc, updateOpts).Return(nil)
	err := status.Update(ctx, &kc)
	require.NoError(t, err)

	status.On("Patch", &kc, fPath, patchOpts).Return(nil)
	err = status.Patch(ctx, &kc, fPath)
	require.NoError(t, err)
}
