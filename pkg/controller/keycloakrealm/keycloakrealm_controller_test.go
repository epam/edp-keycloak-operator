package keycloakrealm

import (
	"context"
	"errors"
	"fmt"
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/mock"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"

	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileKeycloakRealm_ReconcileWithoutOwners(t *testing.T) {
	//prepare
	//vars
	kRealmName := "main"
	ns := "security"
	// dependent custom resources
	kr := &v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
		},
	}
	//client and scheme
	objs := []runtime.Object{kr}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, kr, &v1alpha1.KeycloakClient{})
	client := fake.NewFakeClient(objs...)

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      kRealmName,
			Namespace: ns,
		},
	}

	//reconcile
	r := ReconcileKeycloakRealm{
		client: client,
		scheme: s,
	}

	//test
	res, err := r.Reconcile(req)

	//verify
	assert.Error(t, err)
	assert.False(t, res.Requeue)

	persKr := &v1alpha1.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.False(t, persKr.Status.Available)
}

func TestReconcileKeycloakRealm_ReconcileWithoutKeycloakOwner(t *testing.T) {
	//prepare
	//vars
	kRealmName := "main"
	ns := "security"
	// dependent custom resources
	kr := &v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
		},
	}
	kr.SetOwnerReferences([]metav1.OwnerReference{
		{
			Kind: "AnotherKind",
			Name: "AnotherName",
		},
	})
	//client and scheme
	objs := []runtime.Object{
		kr,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, kr, &v1alpha1.KeycloakClient{})
	client := fake.NewFakeClient(objs...)

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      kRealmName,
			Namespace: ns,
		},
	}

	//reconcile
	r := ReconcileKeycloakRealm{
		client: client,
		scheme: s,
	}

	//test
	res, err := r.Reconcile(req)

	//verify
	assert.Error(t, err)
	assert.False(t, res.Requeue)

	persKr := &v1alpha1.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.False(t, persKr.Status.Available)
}

func TestReconcileKeycloakRealm_ReconcileNotConnectedOwner(t *testing.T) {
	//prepare
	//vars
	kServerUrl := "http://some.security"
	kRealmName := "main"
	ns := "security"
	// dependent custom resources
	k := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-keycloak",
			Namespace: ns,
		},
		Spec: v1alpha1.KeycloakSpec{
			Url: kServerUrl,
		},
		Status: v1alpha1.KeycloakStatus{
			Connected: false,
		},
	}
	kr := &v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
		},
	}
	kr.SetOwnerReferences([]metav1.OwnerReference{
		{
			Kind: "Keycloak",
			Name: k.Name,
		},
	})
	//client and scheme
	objs := []runtime.Object{
		k, kr,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, k, kr, &v1alpha1.KeycloakClient{})
	client := fake.NewFakeClient(objs...)

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      kRealmName,
			Namespace: ns,
		},
	}

	//reconcile
	r := ReconcileKeycloakRealm{
		client: client,
		scheme: s,
	}

	//test
	res, err := r.Reconcile(req)

	//verify
	assert.Error(t, err)
	assert.False(t, res.Requeue)

	persKr := &v1alpha1.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.False(t, persKr.Status.Available)
}

func TestReconcileKeycloakRealm_ReconcileInvalidOwnerCredentials(t *testing.T) {
	//prepare
	//vars
	kServerUrl := "http://some.security"
	kServerUsr := "user"
	kServerPwd := "pass"
	kSecretName := "keycloak-secret"
	kRealmName := "main"
	ns := "security"
	// dependent custom resources
	k := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-keycloak",
			Namespace: ns,
		},
		Spec: v1alpha1.KeycloakSpec{
			Url:    kServerUrl,
			Secret: kSecretName,
		},
		Status: v1alpha1.KeycloakStatus{
			Connected: true,
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kSecretName,
			Namespace: ns,
		},
		Data: map[string][]byte{
			"username": []byte(kServerUsr),
			"password": []byte(kServerPwd),
		},
	}
	kr := &v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
		},
	}
	kr.SetOwnerReferences([]metav1.OwnerReference{
		{
			Kind: "Keycloak",
			Name: k.Name,
		},
	})
	//client and scheme
	objs := []runtime.Object{
		k, kr, secret,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, k, kr, &v1alpha1.KeycloakClient{})
	client := fake.NewFakeClient(objs...)

	//keycloak client, factory and handler
	keycloakDto := dto.Keycloak{
		Url:  kServerUrl,
		User: kServerUsr,
		Pwd:  kServerPwd,
	}
	factory := new(mock.MockGoCloakFactory)
	factory.On("New", keycloakDto).
		Return(nil, errors.New("invalid credentials"))

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      kRealmName,
			Namespace: ns,
		},
	}

	//reconcile
	r := ReconcileKeycloakRealm{
		client:  client,
		scheme:  s,
		factory: factory,
	}

	//test
	res, err := r.Reconcile(req)

	//verify
	assert.Error(t, err)
	assert.False(t, res.Requeue)

	persKr := &v1alpha1.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.False(t, persKr.Status.Available)
}
