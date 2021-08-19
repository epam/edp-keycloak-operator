package helper

import (
	"context"
	"testing"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type fakePatch string

func (f fakePatch) Type() types.PatchType {
	return types.PatchType(f)
}

func (f fakePatch) Data(obj runtime.Object) ([]byte, error) {
	return []byte(f), nil
}

func TestK8SClientMock_OneLiners(t *testing.T) {
	k8sMock := K8SClientMock{}

	var (
		kc              v1alpha1.Keycloak
		kList           v1alpha1.KeycloakList
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

	if err := k8sMock.Create(ctx, &kc); err != nil {
		t.Fatal(err)
	}

	k8sMock.On("List", &kList, listOpts).Return(nil)
	if err := k8sMock.List(ctx, &kList); err != nil {
		t.Fatal(err)
	}

	k8sMock.On("Delete", &kc, deleteOpts).Return(nil)
	if err := k8sMock.Delete(ctx, &kc); err != nil {
		t.Fatal(err)
	}

	k8sMock.On("Update", &kc, updateOpts).Return(nil)
	if err := k8sMock.Update(ctx, &kc); err != nil {
		t.Fatal(err)
	}

	k8sMock.On("Patch", &kc, fPatch, patchOpts).Return(nil)
	if err := k8sMock.Patch(ctx, &kc, fPatch); err != nil {
		t.Fatal(err)
	}

	k8sMock.On("DeleteAllOf", &kc, deleteAllOfOpts).Return(nil)
	if err := k8sMock.DeleteAllOf(ctx, &kc); err != nil {
		t.Fatal(err)
	}

}

func TestK8SClientMock_Status(t *testing.T) {
	s := K8SStatusMock{}
	k8sMock := K8SClientMock{}
	k8sMock.SetStatus(&s)
	sch := runtime.Scheme{}
	k8sMock.SetScheme(&sch)

	if k8sMock.Scheme() == nil {
		t.Fatal("scheme must be not nil")
	}

	if k8sMock.Status() == nil {
		t.Fatal("status must be not nil")
	}

	if k8sMock.RESTMapper() != nil {
		t.Fatal("rest mapper must be nil")
	}
}

func TestK8SClientMock_Get(t *testing.T) {
	k8sMock := K8SClientMock{}
	var (
		kcRequest v1alpha1.Keycloak
		kcResult  = v1alpha1.Keycloak{Status: v1alpha1.KeycloakStatus{Connected: true}}
	)

	k8sMock.On("Get", types.NamespacedName{}, &kcRequest).Return(nil, &kcResult)
	if err := k8sMock.Get(context.Background(), types.NamespacedName{}, &kcRequest); err != nil {
		t.Fatal(err)
	}

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
		kc         v1alpha1.Keycloak
		fPath      fakePatch
	)

	status.On("Update", &kc, updateOpts).Return(nil)
	if err := status.Update(ctx, &kc); err != nil {
		t.Fatal(err)
	}

	status.On("Patch", &kc, fPath, patchOpts).Return(nil)
	if err := status.Patch(ctx, &kc, fPath); err != nil {
		t.Fatal(err)
	}
}
