package keycloakrealm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v13 "github.com/epam/edp-keycloak-operator/api/v1/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestReconcileKeycloakRealm_ReconcileWithoutOwners(t *testing.T) {
	kRealmName := "main"
	ns := "security"
	kr := &v13.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: v13.KeycloakRealmSpec{
			RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, kr, &v13.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(kr).Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      kRealmName,
			Namespace: ns,
		},
	}

	r := ReconcileKeycloakRealm{
		client: client,
		helper: helper.MakeHelper(client, s, mock.NewLogr()),
		log:    mock.NewLogr(),
	}

	res, err := r.Reconcile(context.TODO(), req)
	assert.Nil(t, err)

	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}

	assert.False(t, res.Requeue)

	persKr := &v13.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.Nil(t, err)
	assert.False(t, persKr.Status.Available)
}

func TestReconcileKeycloakRealm_ReconcileWithoutKeycloakOwner(t *testing.T) {
	kRealmName := "main"
	ns := "security"
	kr := &v13.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: v13.KeycloakRealmSpec{
			RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
		},
	}
	kr.SetOwnerReferences([]metav1.OwnerReference{
		{
			Kind: "AnotherKind",
			Name: "AnotherName",
		},
	})

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, kr, &v13.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(kr).Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      kRealmName,
			Namespace: ns,
		},
	}

	r := ReconcileKeycloakRealm{
		client: client,
		helper: helper.MakeHelper(client, s, mock.NewLogr()),
		log:    mock.NewLogr(),
	}

	res, err := r.Reconcile(context.TODO(), req)
	assert.Nil(t, err)

	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}

	assert.False(t, res.Requeue)

	persKr := &v13.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.Nil(t, err)
	assert.False(t, persKr.Status.Available)
}

func TestReconcileKeycloakRealm_ReconcileNotConnectedOwner(t *testing.T) {
	kServerUrl := "http://some.security"
	kRealmName := "main"
	ns := "security"

	k := &v13.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-keycloak",
			Namespace: ns,
		},
		Spec: v13.KeycloakSpec{
			Url: kServerUrl,
		},
		Status: v13.KeycloakStatus{
			Connected: false,
		},
	}
	kr := &v13.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: v13.KeycloakRealmSpec{
			RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
		},
	}
	kr.SetOwnerReferences([]metav1.OwnerReference{
		{
			Kind: "Keycloak",
			Name: k.Name,
		},
	})

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, k, kr, &v13.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(k, kr).Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      kRealmName,
			Namespace: ns,
		},
	}

	r := ReconcileKeycloakRealm{
		client: client,
		helper: helper.MakeHelper(client, s, mock.NewLogr()),
		log:    mock.NewLogr(),
	}

	res, err := r.Reconcile(context.TODO(), req)
	assert.Nil(t, err)

	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}

	assert.False(t, res.Requeue)

	persKr := &v13.KeycloakRealm{}
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
	k := &v13.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-keycloak",
			Namespace: ns,
		},
		Spec: v13.KeycloakSpec{
			Url:    kServerUrl,
			Secret: kSecretName,
		},
		Status: v13.KeycloakStatus{
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
	kr := &v13.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: v13.KeycloakRealmSpec{
			RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
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
	s.AddKnownTypes(v1.SchemeGroupVersion, k, kr, &v13.KeycloakClient{})
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
		helper: helper.MakeHelper(client, s, mock.NewLogr()),
		log:    mock.NewLogr(),
	}

	res, err := r.Reconcile(context.TODO(), req)
	assert.Nil(t, err)

	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}

	assert.False(t, res.Requeue)

	persKr := &v13.KeycloakRealm{}
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
	k := &v13.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-keycloak",
			Namespace: ns,
		},
		Spec: v13.KeycloakSpec{
			Url:    kServerUrl,
			Secret: kSecretName,
		},
		Status: v13.KeycloakStatus{
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
	kr := &v13.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: v13.KeycloakRealmSpec{
			KeycloakOwner: k.Name,
			RealmName:     fmt.Sprintf("%v.%v", ns, kRealmName),
		},
	}
	// client and scheme
	objs := []runtime.Object{
		k, kr, secret,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, k, kr, &v13.KeycloakClient{})

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
		helper: helper.MakeHelper(client, s, mock.NewLogr()),
		log:    mock.NewLogr(),
	}

	// test
	res, err := r.Reconcile(context.TODO(), req)
	assert.Nil(t, err)

	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}

	assert.False(t, res.Requeue)

	persKr := &v13.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.Nil(t, err)
	assert.False(t, persKr.Status.Available)
}

func TestReconcileKeycloakRealm_ReconcileDelete(t *testing.T) {
	ns, kSecretName, kServerUsr, kServerPwd, kRealmName := "test", "test", "test", "test", "test"
	k := v13.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "test-keycloak", Namespace: ns},
		Spec: v13.KeycloakSpec{Secret: kSecretName}, Status: v13.KeycloakStatus{Connected: true},
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: kSecretName, Namespace: ns}, Data: map[string][]byte{
		"username": []byte(kServerUsr), "password": []byte(kServerPwd)}}
	tNow := metav1.Time{Time: time.Now()}
	kr := v13.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns,
		DeletionTimestamp: &tNow},
		Spec: v13.KeycloakRealmSpec{KeycloakOwner: k.Name, RealmName: fmt.Sprintf("%v.%v", ns, kRealmName)},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &k, &kr, &v13.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(&secret, &k, &kr).Build()

	kClient := new(adapter.Mock)
	kClient.On("DeleteRealm", "test.test").Return(nil)

	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: kRealmName, Namespace: ns}}
	r := ReconcileKeycloakRealm{
		client: client,
		helper: helper.MakeHelper(client, s, mock.NewLogr()),
		log:    mock.NewLogr(),
	}

	_, err := r.Reconcile(context.TODO(), req)
	require.NoError(t, err)
}

func TestReconcileKeycloakRealm_Reconcile(t *testing.T) {
	ns, kRealmName := "namespace", "realm-11"
	ssoRealmMappers := []v13.SSORealmMapper{{}}

	kr := v13.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns},
		TypeMeta: metav1.TypeMeta{Kind: "KeycloakRealm", APIVersion: "apps/v1"},
		Spec: v13.KeycloakRealmSpec{KeycloakOwner: "keycloak-main", RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
			SSORealmMappers: &ssoRealmMappers},
		Status: v13.KeycloakRealmStatus{Available: true, Value: helper.StatusOK},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &kr, &v13.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(&kr).Build()

	kClient := new(adapter.Mock)
	nsName := types.NamespacedName{Name: kRealmName, Namespace: ns}
	req := reconcile.Request{NamespacedName: nsName}

	h := helper.Mock{}
	logger := mock.NewLogr()

	h.On("CreateKeycloakClientForRealm", &kr).Return(kClient, nil)
	h.On("TryToDelete", &kr,
		makeTerminator(kr.Spec.RealmName, kClient, logger),
		keyCloakRealmOperatorFinalizerName).Return(false, nil)
	h.On("UpdateStatus", &kr).Return(nil)

	ch := handler.MockRealmHandler{}
	r := ReconcileKeycloakRealm{
		client:                  client,
		helper:                  &h,
		log:                     logger,
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
