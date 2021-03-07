package helper

import (
	"testing"

	"github.com/epmd-edp/keycloak-operator/pkg/apis"
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func TestHelper_GetOrCreateRealmOwnerRef(t *testing.T) {
	mc := Client{}

	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	helper := MakeHelper(&mc, scheme)

	kcClient := v1alpha1.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "foo",
					Kind: "KeycloakRealm",
				},
			},
		},
	}

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "foo",
	}, &v1alpha1.KeycloakRealm{}).Return(nil)

	_, err := helper.GetOrCreateRealmOwnerRef(&kcClient, kcClient.ObjectMeta)
	if err != nil {
		t.Fatal(err)
	}

	kcClient = v1alpha1.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
		},
	}

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "main",
	}, &v1alpha1.KeycloakRealm{}).Return(nil)

	_, err = helper.GetOrCreateRealmOwnerRef(&kcClient, kcClient.ObjectMeta)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHelper_GetOrCreateRealmOwnerRef_Failure(t *testing.T) {
	mc := Client{}

	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	helper := MakeHelper(&mc, scheme)

	kcClient := v1alpha1.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "foo",
					Kind: "KeycloakRealm",
				},
			},
		},
	}

	mockErr := errors.New("mock error")

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "foo",
	}, &v1alpha1.KeycloakRealm{}).Return(mockErr)

	_, err := helper.GetOrCreateRealmOwnerRef(&kcClient, kcClient.ObjectMeta)
	if err == nil {
		t.Fatal("no error on k8s client get fatal")
	}

	if errors.Cause(err) != mockErr {
		t.Fatal("wrong error returned")
	}

	kcClient = v1alpha1.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
		},
	}

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "main",
	}, &v1alpha1.KeycloakRealm{}).Return(mockErr)

	_, err = helper.GetOrCreateRealmOwnerRef(&kcClient, kcClient.ObjectMeta)
	if err == nil {
		t.Fatal("no error on k8s client get fatal")
	}

	if errors.Cause(err) != mockErr {
		t.Fatal("wrong error returned")
	}
}

func TestHelper_GetOrCreateKeycloakOwnerRef(t *testing.T) {
	mc := Client{}

	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	helper := MakeHelper(&mc, scheme)

	realm := v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "foo",
					Kind: "Keycloak",
				},
			},
		},
	}

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "foo",
	}, &v1alpha1.Keycloak{}).Return(nil)

	_, err := helper.GetOrCreateKeycloakOwnerRef(&realm)
	if err != nil {
		t.Fatal(err)
	}

	realm = v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
		},

		Spec: v1alpha1.KeycloakRealmSpec{
			KeycloakOwner: "test321",
		},
	}

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "test321",
	}, &v1alpha1.Keycloak{}).Return(nil)

	_, err = helper.GetOrCreateKeycloakOwnerRef(&realm)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHelper_GetOrCreateKeycloakOwnerRef_Failure(t *testing.T) {
	mc := Client{}

	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	helper := MakeHelper(&mc, scheme)

	realm := v1alpha1.KeycloakRealm{}

	_, err := helper.GetOrCreateKeycloakOwnerRef(&realm)
	if err == nil {
		t.Fatal("no error on empty owner reference and spec")
	}

	if errors.Cause(err).Error() != "keycloak owner is not specified neither in ownerReference nor in spec for realm " {
		t.Log(errors.Cause(err).Error())
		t.Fatal("wrong error message returned")
	}

	realm = v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "foo",
					Kind: "Deployment",
				},
			},
		},
	}

	_, err = helper.GetOrCreateKeycloakOwnerRef(&realm)
	if err == nil {
		t.Fatal("no error on empty owner reference and spec")
	}

	if errors.Cause(err).Error() != "keycloak owner is not specified neither in ownerReference nor in spec for realm " {
		t.Log(errors.Cause(err).Error())
		t.Fatal("wrong error message returned")
	}

	realm = v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "foo",
					Kind: "Deployment",
				},
			},
		},
		Spec: v1alpha1.KeycloakRealmSpec{
			KeycloakOwner: "testSpec",
		},
	}

	mockErr := errors.New("fatal")
	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "testSpec",
	}, &v1alpha1.Keycloak{}).Return(mockErr)

	_, err = helper.GetOrCreateKeycloakOwnerRef(&realm)
	if err == nil {
		t.Fatal("no error on k8s client get fatal")
	}

	if errors.Cause(err) != mockErr {
		t.Fatal("wrong error returned")
	}

	realm = v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "testOwnerReference",
					Kind: "Keycloak",
				},
			},
		},
		Spec: v1alpha1.KeycloakRealmSpec{
			KeycloakOwner: "testSpec",
		},
	}

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "testOwnerReference",
	}, &v1alpha1.Keycloak{}).Return(mockErr)

	_, err = helper.GetOrCreateKeycloakOwnerRef(&realm)
	if err == nil {
		t.Fatal("no error on k8s client get fatal")
	}

	if errors.Cause(err) != mockErr {
		t.Fatal("wrong error returned")
	}
}

func TestHelper_CreateKeycloakClient(t *testing.T) {
	mc := Client{}

	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	helper := MakeHelper(&mc, scheme)
	realm := v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "testOwnerReference",
					Kind: "Keycloak",
				},
			},
		},
	}

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "testOwnerReference",
	}, &v1alpha1.Keycloak{}).Return(nil)

	mc.On("Get", types.NamespacedName{
		Namespace: "",
		Name:      "",
	}, &v1.Secret{}).Return(nil)

	clientFactory := ClientFactory{}
	clientFactory.On("New", dto.Keycloak{}).Return(adapter.GoCloakAdapter{}, nil)

	_, err := helper.CreateKeycloakClient(&realm, &clientFactory)
	if err != nil {
		t.Fatal(err)
	}
}
