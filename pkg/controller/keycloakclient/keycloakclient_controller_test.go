package keycloakclient

import (
	"context"
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/mock"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func TestReconcileKeycloakClient_WithoutOwnerReference(t *testing.T) {
	//prepare
	//client & scheme
	k := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-keycloak",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakSpec{
			Url:    "https://some",
			Secret: "keycloak-secret",
		},
		Status: v1alpha1.KeycloakStatus{
			Connected: true,
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-secret",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}
	kr := &v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: "namespace.main",
		},
	}
	kc := &v1alpha1.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakClientSpec{
			TargetRealm: "main",
			Secret:      "keycloak-secret",
			RealmRoles: &[]v1alpha1.RealmRole{
				{
					Name:      "fake-client-administrators",
					Composite: "administrator",
				},
				{
					Name:      "fake-client-users",
					Composite: "developer",
				},
			},
			Public:                  true,
			ClientId:                "fake-client",
			WebUrl:                  "fake-url",
			DirectAccess:            false,
			AdvancedProtocolMappers: true,
			ClientRoles:             nil,
		},
	}
	objs := []runtime.Object{
		k, kr, secret, kc,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, k, kr, kc)
	client := fake.NewFakeClient(objs...)

	//keycloak client and factory

	kclient := new(mock.MockKeycloakClient)
	c := dto.ConvertSpecToClient(kc.Spec, "")
	kclient.On("ExistClient", c).Return(
		false, nil)
	kclient.On("CreateClient", c).Return(
		nil)
	kclient.On("GetClientId", c).Return(
		"uuid", nil)
	rm := dto.ConvertSpecToRealm(kr.Spec)
	ar := dto.RealmRole{
		Name:      "fake-client-administrators",
		Composite: "administrator",
	}
	kclient.On("ExistRealmRole", rm, ar).Return(
		false, nil)
	kclient.On("CreateRealmRole", rm, ar).Return(
		nil)
	dr := dto.RealmRole{
		Name:      "fake-client-users",
		Composite: "developer",
	}
	kclient.On("ExistRealmRole", rm, dr).Return(
		false, nil)
	kclient.On("CreateRealmRole", rm, dr).Return(
		nil)

	keycloakDto := dto.Keycloak{
		Url:  "https://some",
		User: "user",
		Pwd:  "pass",
	}
	factory := new(mock.MockGoCloakFactory)
	factory.On("New", keycloakDto).
		Return(kclient, nil)

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "main",
			Namespace: "namespace",
		},
	}

	//reconcile
	r := ReconcileKeycloakClient{
		client:  client,
		scheme:  s,
		factory: factory,
	}

	//test
	res, err := r.Reconcile(req)

	//verify
	assert.Error(t, err)
	assert.False(t, res.Requeue)

	persKc := &v1alpha1.KeycloakClient{}
	err = client.Get(context.TODO(), req.NamespacedName, persKc)
	assert.Equal(t, "FAIL", persKc.Status.Value)
	assert.Empty(t, persKc.Status.Id)
}
