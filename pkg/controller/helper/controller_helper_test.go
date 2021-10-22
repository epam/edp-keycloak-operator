package helper

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHelper_GetOrCreateRealmOwnerRef(t *testing.T) {
	mc := Client{}

	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	helper := MakeHelper(&mc, scheme, nil)

	kcGroup := v1alpha1.KeycloakRealmGroup{
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

	_, err := helper.GetOrCreateRealmOwnerRef(&kcGroup, kcGroup.ObjectMeta)
	if err != nil {
		t.Fatal(err)
	}

	kcGroup = v1alpha1.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
		},
		Spec: v1alpha1.KeycloakRealmGroupSpec{
			Realm: "foo13",
		},
	}

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "foo13",
	}, &v1alpha1.KeycloakRealm{}).Return(nil)

	_, err = helper.GetOrCreateRealmOwnerRef(&kcGroup, kcGroup.ObjectMeta)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHelper_GetOrCreateRealmOwnerRef_Failure(t *testing.T) {
	mc := Client{}

	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	helper := MakeHelper(&mc, scheme, nil)

	kcGroup := v1alpha1.KeycloakRealmGroup{
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

	_, err := helper.GetOrCreateRealmOwnerRef(&kcGroup, kcGroup.ObjectMeta)
	if err == nil {
		t.Fatal("no error on k8s client get fatal")
	}

	if errors.Cause(err) != mockErr {
		t.Fatal("wrong error returned")
	}

	kcGroup = v1alpha1.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
		},
		Spec: v1alpha1.KeycloakRealmGroupSpec{Realm: "main123"},
	}

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "main123",
	}, &v1alpha1.KeycloakRealm{}).Return(mockErr)

	_, err = helper.GetOrCreateRealmOwnerRef(&kcGroup, kcGroup.ObjectMeta)
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
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	helper := MakeHelper(&mc, scheme, nil)

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
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	helper := MakeHelper(&mc, scheme, nil)

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

func TestMakeHelper(t *testing.T) {
	rCl := resty.New()
	httpmock.ActivateNonDefault(rCl.GetClient())
	httpmock.RegisterResponder("POST", "/k-url/auth/realms/master/protocol/openid-connect/token",
		httpmock.NewStringResponder(200, "{}"))

	h := MakeHelper(nil, nil, nil)
	_, err := h.adapterBuilder(context.Background(), "k-url", "foo", "bar",
		v1alpha1.KeycloakAdminTypeServiceAccount, nil, rCl)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHelper_CreateKeycloakClient(t *testing.T) {
	mc := Client{}

	utilruntime.Must(v1alpha1.AddToScheme(scheme.Scheme))
	helper := MakeHelper(&mc, scheme.Scheme, nil)
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

	kc := v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "testOwnerReference"},
		Status:     v1alpha1.KeycloakStatus{Connected: true},
		Spec:       v1alpha1.KeycloakSpec{Secret: "ss1"},
	}

	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(&kc).Build()

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "testOwnerReference",
	}, &v1alpha1.Keycloak{}).Return(fakeCl)

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      kc.Spec.Secret,
	}, &v1.Secret{}).Return(nil)

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "kc-token-testOwnerReference",
	}, &v1.Secret{}).Return(&k8sErrors.StatusError{ErrStatus: metav1.Status{
		Status:  metav1.StatusFailure,
		Code:    http.StatusNotFound,
		Reason:  metav1.StatusReasonNotFound,
		Message: "not found",
	}})

	_, err := helper.CreateKeycloakClientForRealm(context.Background(), &realm)
	if err == nil {
		t.Fatal("no error on trying to connect to keycloak")
	}

	if !strings.Contains(err.Error(), "could not get token") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
