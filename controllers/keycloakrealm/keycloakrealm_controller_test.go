package keycloakrealm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	helpermock "github.com/epam/edp-keycloak-operator/controllers/helper/mocks"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

func TestReconcileKeycloakRealm_ReconcileWithoutOwners(t *testing.T) {
	kRealmName := "main"
	ns := "security"
	kr := &keycloakApi.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
			KeycloakRef: common.KeycloakRef{
				Kind: keycloakApi.KeycloakKind,
				Name: "kc",
			},
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, kr, &keycloakApi.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(kr).Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      kRealmName,
			Namespace: ns,
		},
	}

	r := ReconcileKeycloakRealm{
		client: client,
		helper: helper.MakeHelper(client, s, "default"),
	}

	res, err := r.Reconcile(context.TODO(), req)
	assert.Nil(t, err)

	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}

	assert.False(t, res.Requeue)

	persKr := &keycloakApi.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.Nil(t, err)
	assert.False(t, persKr.Status.Available)
}

func TestReconcileKeycloakRealm_ReconcileWithoutKeycloakOwner(t *testing.T) {
	kRealmName := "main"
	ns := "security"
	kr := &keycloakApi.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
			KeycloakRef: common.KeycloakRef{
				Kind: keycloakApi.KeycloakKind,
				Name: "kc",
			},
		},
	}
	kr.SetOwnerReferences([]metav1.OwnerReference{
		{
			Kind: "AnotherKind",
			Name: "AnotherName",
		},
	})

	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, kr, &keycloakApi.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(kr).Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      kRealmName,
			Namespace: ns,
		},
	}

	r := ReconcileKeycloakRealm{
		client: client,
		helper: helper.MakeHelper(client, s, "default"),
	}

	res, err := r.Reconcile(context.TODO(), req)
	assert.Nil(t, err)

	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}

	assert.False(t, res.Requeue)

	persKr := &keycloakApi.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.Nil(t, err)
	assert.False(t, persKr.Status.Available)
}

func TestReconcileKeycloakRealm_ReconcileNotConnectedOwner(t *testing.T) {
	kServerUrl := "http://some.security"
	kRealmName := "main"
	ns := "security"

	k := &keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-keycloak",
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakSpec{
			Url: kServerUrl,
		},
		Status: keycloakApi.KeycloakStatus{
			Connected: false,
		},
	}
	kr := &keycloakApi.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
			KeycloakRef: common.KeycloakRef{
				Kind: keycloakApi.KeycloakKind,
				Name: k.Name,
			},
		},
	}
	kr.SetOwnerReferences([]metav1.OwnerReference{
		{
			Kind: "Keycloak",
			Name: k.Name,
		},
	})

	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, k, kr, &keycloakApi.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(k, kr).Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      kRealmName,
			Namespace: ns,
		},
	}

	r := ReconcileKeycloakRealm{
		client: client,
		helper: helper.MakeHelper(client, s, "default"),
	}

	res, err := r.Reconcile(context.TODO(), req)
	assert.Nil(t, err)

	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}

	assert.False(t, res.Requeue)

	persKr := &keycloakApi.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.Nil(t, err)
	assert.False(t, persKr.Status.Available)
}

func TestReconcileKeycloakRealm_ReconcileInvalidOwnerCredentials(t *testing.T) {
	kServerUrl := "http://some.security"
	kServerUsr := "user"
	kServerPwd := "pass"
	kSecretName := "keycloak-secret"
	kRealmName := "main"
	ns := "security"
	// dependent custom resources
	k := &keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-keycloak",
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakSpec{
			Url:    kServerUrl,
			Secret: kSecretName,
		},
		Status: keycloakApi.KeycloakStatus{
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
	kr := &keycloakApi.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
			KeycloakRef: common.KeycloakRef{
				Kind: keycloakApi.KeycloakKind,
				Name: k.Name,
			},
		},
	}
	kr.SetOwnerReferences([]metav1.OwnerReference{
		{
			Kind: "Keycloak",
			Name: k.Name,
		},
	})
	// client and scheme
	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, k, kr, &keycloakApi.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(k, kr, secret).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      kRealmName,
			Namespace: ns,
		},
	}

	r := ReconcileKeycloakRealm{
		client: client,
		helper: helper.MakeHelper(client, s, "default"),
	}

	res, err := r.Reconcile(context.TODO(), req)
	assert.Nil(t, err)

	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}

	assert.False(t, res.Requeue)

	persKr := &keycloakApi.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.Nil(t, err)
	assert.False(t, persKr.Status.Available)
}

func TestReconcileKeycloakRealm_ReconcileWithKeycloakOwnerAndInvalidCreds(t *testing.T) {
	// prepare
	// vars
	kServerUrl := "http://some.security"
	kServerUsr := "user"
	kServerPwd := "pass"
	kSecretName := "keycloak-secret"
	kRealmName := "main"
	ns := "security"
	// dependent custom resources
	k := &keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-keycloak",
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakSpec{
			Url:    kServerUrl,
			Secret: kSecretName,
		},
		Status: keycloakApi.KeycloakStatus{
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
	kr := &keycloakApi.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakRealmSpec{
			KeycloakRef: common.KeycloakRef{
				Kind: keycloakApi.KeycloakKind,
				Name: k.Name,
			},
			RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
		},
	}
	// client and scheme
	objs := []runtime.Object{
		k, kr, secret,
	}
	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, k, kr, &keycloakApi.KeycloakClient{})

	client := fake.NewClientBuilder().WithRuntimeObjects(objs...).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      kRealmName,
			Namespace: ns,
		},
	}

	// reconcile
	r := ReconcileKeycloakRealm{
		client: client,
		helper: helper.MakeHelper(client, s, "default"),
	}

	// test
	res, err := r.Reconcile(context.TODO(), req)
	assert.Nil(t, err)

	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}

	assert.False(t, res.Requeue)

	persKr := &keycloakApi.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.Nil(t, err)
	assert.False(t, persKr.Status.Available)
}

func TestReconcileKeycloakRealm_ReconcileDelete(t *testing.T) {
	ns, kSecretName, kServerUsr, kServerPwd, kRealmName := "test", "test", "test", "test", "test"
	k := keycloakApi.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "test-keycloak", Namespace: ns},
		Spec: keycloakApi.KeycloakSpec{Secret: kSecretName}, Status: keycloakApi.KeycloakStatus{Connected: true},
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: kSecretName, Namespace: ns}, Data: map[string][]byte{
		"username": []byte(kServerUsr), "password": []byte(kServerPwd)}}
	tNow := metav1.Time{Time: time.Now()}
	kr := keycloakApi.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns,
		DeletionTimestamp: &tNow},
		Spec: keycloakApi.KeycloakRealmSpec{KeycloakOwner: k.Name, RealmName: fmt.Sprintf("%v.%v", ns, kRealmName)},
	}

	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &k, &kr, &keycloakApi.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(&secret, &k, &kr).Build()

	kClient := new(adapter.Mock)
	kClient.On("DeleteRealm", "test.test").Return(nil)

	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: kRealmName, Namespace: ns}}
	r := ReconcileKeycloakRealm{
		client: client,
		helper: helper.MakeHelper(client, s, "default"),
	}

	_, err := r.Reconcile(context.TODO(), req)
	require.NoError(t, err)
}

func TestReconcileKeycloakRealm_Reconcile(t *testing.T) {
	ns, kRealmName := "namespace", "realm-11"
	ssoRealmMappers := []keycloakApi.SSORealmMapper{{}}

	kr := keycloakApi.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KeycloakRealm",
			APIVersion: "apps/v1",
		},
		Spec: keycloakApi.KeycloakRealmSpec{
			KeycloakRef: common.KeycloakRef{
				Kind: keycloakApi.KeycloakKind,
				Name: "realm",
			},
			RealmName:       fmt.Sprintf("%v.%v", ns, kRealmName),
			SSORealmMappers: &ssoRealmMappers,
		},
		Status: keycloakApi.KeycloakRealmStatus{Available: true, Value: helper.StatusOK},
	}

	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &kr, &keycloakApi.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(&kr).Build()

	kClient := new(adapter.Mock)
	nsName := types.NamespacedName{Name: kRealmName, Namespace: ns}
	req := reconcile.Request{NamespacedName: nsName}

	h := helpermock.NewControllerHelper(t)

	h.On("SetKeycloakOwnerRef", testifymock.Anything, &kr).Return(nil)
	h.On("CreateKeycloakClientFromRealm", testifymock.Anything, &kr).Return(kClient, nil)
	h.On("TryToDelete", testifymock.Anything, testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(false, nil)

	ch := handler.MockRealmHandler{}
	r := ReconcileKeycloakRealm{
		client:                  client,
		helper:                  h,
		chain:                   &ch,
		successReconcileTimeout: time.Hour,
	}

	ch.On("ServeRequest", &kr, kClient).Return(nil)

	res, err := r.Reconcile(context.TODO(), req)
	require.NoError(t, err)

	if res.RequeueAfter != r.successReconcileTimeout {
		t.Fatal("success reconcile timeout is not set")
	}
}
