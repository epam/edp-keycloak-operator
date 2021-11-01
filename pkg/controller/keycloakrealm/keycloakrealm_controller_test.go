package keycloakrealm

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile_SetupWithManager(t *testing.T) {
	l := mock.Logger{}
	h := helper.MakeHelper(nil, scheme.Scheme, &l)

	r := NewReconcileKeycloakRealm(nil, nil, &l, h)
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{MetricsBindAddress: "0"})
	if err != nil {
		t.Fatal(err)
	}

	err = r.SetupWithManager(mgr, time.Second)
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "no kind is registered for the type") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	if r.successReconcileTimeout != time.Second {
		t.Fatal("success reconcile timeout is not set")
	}
}

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
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, kr, &v1alpha1.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(kr).Build()

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
		helper: helper.MakeHelper(client, s, nil),
		log:    &mock.Logger{},
	}

	//test
	res, err := r.Reconcile(context.TODO(), req)

	//verify
	assert.Nil(t, err)
	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}
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
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, kr, &v1alpha1.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(kr).Build()

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
		helper: helper.MakeHelper(client, s, nil),
		log:    &mock.Logger{},
	}

	//test
	res, err := r.Reconcile(context.TODO(), req)

	//verify
	assert.Nil(t, err)
	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}
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
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, k, kr, &v1alpha1.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(k, kr).Build()

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
		helper: helper.MakeHelper(client, s, nil),
		log:    &mock.Logger{},
	}

	//test
	res, err := r.Reconcile(context.TODO(), req)

	//verify
	assert.Nil(t, err)
	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}
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
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, k, kr, &v1alpha1.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(k, kr, secret).Build()

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
		helper: helper.MakeHelper(client, s, nil),
		log:    &mock.Logger{},
	}

	//test
	res, err := r.Reconcile(context.TODO(), req)

	//verify
	assert.Nil(t, err)
	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}
	assert.False(t, res.Requeue)

	persKr := &v1alpha1.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.False(t, persKr.Status.Available)
}

func TestReconcileKeycloakRealm_ReconcileWithKeycloakOwnerAndInvalidCreds(t *testing.T) {
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
			KeycloakOwner: k.Name,
			RealmName:     fmt.Sprintf("%v.%v", ns, kRealmName),
		},
	}
	//client and scheme
	objs := []runtime.Object{
		k, kr, secret,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, k, kr, &v1alpha1.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(objs...).Build()

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
		helper: helper.MakeHelper(client, s, nil),
		log:    &mock.Logger{},
	}

	//test
	res, err := r.Reconcile(context.TODO(), req)

	//verify
	assert.Nil(t, err)
	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}
	assert.False(t, res.Requeue)

	persKr := &v1alpha1.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.False(t, persKr.Status.Available)
}

func TestReconcileKeycloakRealm_ReconcileDelete(t *testing.T) {
	ns, kSecretName, kServerUsr, kServerPwd, kRealmName := "test", "test", "test", "test", "test"
	k := v1alpha1.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "test-keycloak", Namespace: ns},
		Spec: v1alpha1.KeycloakSpec{Secret: kSecretName}, Status: v1alpha1.KeycloakStatus{Connected: true},
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: kSecretName, Namespace: ns}, Data: map[string][]byte{
		"username": []byte(kServerUsr), "password": []byte(kServerPwd)}}
	tNow := metav1.Time{Time: time.Now()}
	kr := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns,
		DeletionTimestamp: &tNow},
		Spec: v1alpha1.KeycloakRealmSpec{KeycloakOwner: k.Name, RealmName: fmt.Sprintf("%v.%v", ns, kRealmName)},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &k, &kr, &v1alpha1.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(&secret, &k, &kr).Build()

	kClient := new(adapter.Mock)
	kClient.On("DeleteRealm", "test.test").Return(nil)

	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: kRealmName, Namespace: ns}}
	r := ReconcileKeycloakRealm{
		client: client,
		helper: helper.MakeHelper(client, s, nil),
		log:    &mock.Logger{},
	}

	if _, err := r.Reconcile(context.TODO(), req); err != nil {
		t.Fatal(err)
	}
}

func TestReconcileKeycloakRealm_Reconcile(t *testing.T) {
	ns, kRealmName := "namespace", "realm-11"
	ssoRealmMappers := []v1alpha1.SSORealmMapper{{}}

	kr := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns},
		TypeMeta: metav1.TypeMeta{Kind: "KeycloakRealm", APIVersion: "apps/v1"},
		Spec: v1alpha1.KeycloakRealmSpec{KeycloakOwner: "keycloak-main", RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
			SSORealmMappers: &ssoRealmMappers},
		Status: v1alpha1.KeycloakRealmStatus{Available: true, Value: helper.StatusOK},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &kr, &v1alpha1.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(&kr).Build()

	kClient := new(adapter.Mock)
	nsName := types.NamespacedName{Name: kRealmName, Namespace: ns}
	req := reconcile.Request{NamespacedName: nsName}

	h := helper.Mock{}
	logger := mock.Logger{}
	h.On("CreateKeycloakClientForRealm", &kr).Return(kClient, nil)
	h.On("TryToDelete", &kr,
		makeTerminator(kr.Spec.RealmName, kClient, &logger),
		keyCloakRealmOperatorFinalizerName).Return(false, nil)
	h.On("UpdateStatus", &kr).Return(nil)
	ch := handler.MockRealmHandler{}
	r := ReconcileKeycloakRealm{
		client:                  client,
		scheme:                  s,
		helper:                  &h,
		log:                     &logger,
		chain:                   &ch,
		successReconcileTimeout: time.Hour,
	}

	ch.On("ServeRequest", &kr, kClient).Return(nil)

	res, err := r.Reconcile(context.TODO(), req)
	if err != nil {
		t.Fatal(err)
	}

	if res.RequeueAfter != r.successReconcileTimeout {
		t.Fatal("success reconcile timeout is not set")
	}
}
