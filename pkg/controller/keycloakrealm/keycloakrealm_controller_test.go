package keycloakrealm

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain"
	"github.com/epam/edp-keycloak-operator/pkg/model"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
		helper: helper.MakeHelper(client, s),
	}

	//test
	res, err := r.Reconcile(req)

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
		helper: helper.MakeHelper(client, s),
	}

	//test
	res, err := r.Reconcile(req)

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
		helper: helper.MakeHelper(client, s),
	}

	//test
	res, err := r.Reconcile(req)

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
	factory := new(mock.GoCloakFactory)
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
		factory: factory,
		helper:  helper.MakeHelper(client, s),
	}

	//test
	res, err := r.Reconcile(req)

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
	client := fake.NewFakeClient(objs...)

	//keycloak client, factory and handler
	keycloakDto := dto.Keycloak{
		Url:  kServerUrl,
		User: kServerUsr,
		Pwd:  kServerPwd,
	}
	factory := new(mock.GoCloakFactory)
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
		factory: factory,
		helper:  helper.MakeHelper(client, s),
	}

	//test
	res, err := r.Reconcile(req)

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
	client := fake.NewFakeClient(&secret, &k, &kr)

	kClient := new(mock.KeycloakClient)
	kClient.On("DeleteRealm", "test.test").Return(nil)
	factory := new(mock.GoCloakFactory)

	factory.On("New", dto.Keycloak{User: "test", Pwd: "test"}).
		Return(kClient, nil)

	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: kRealmName, Namespace: ns}}
	r := ReconcileKeycloakRealm{client: client, factory: factory, helper: helper.MakeHelper(client, s)}

	if _, err := r.Reconcile(req); err != nil {
		t.Fatal(err)
	}
}

func TestReconcileKeycloakRealm_Reconcile(t *testing.T) {
	ns, kSecretName, kServerUsr, kServerPwd, kRealmName, realmName := "test", "test", "test", "test", "test", "test.test"
	k := v1alpha1.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "test-keycloak", Namespace: ns},
		Spec: v1alpha1.KeycloakSpec{Secret: kSecretName}, Status: v1alpha1.KeycloakStatus{Connected: true},
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: kSecretName, Namespace: ns}, Data: map[string][]byte{
		"username": []byte(kServerUsr), "password": []byte(kServerPwd)}}

	creatorSecret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: adapter.AcCreatorUsername, Namespace: ns}, Data: map[string][]byte{
		"username": []byte(kServerUsr), "password": []byte(kServerPwd)}}

	readerSecret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: adapter.AcReaderUsername, Namespace: ns}, Data: map[string][]byte{
		"username": []byte(kServerUsr), "password": []byte(kServerPwd)}}

	ssoRealmMappers := []v1alpha1.SSORealmMapper{{}}

	kr := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns},
		Spec: v1alpha1.KeycloakRealmSpec{KeycloakOwner: k.Name, RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
			SSORealmMappers: &ssoRealmMappers},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &k, &kr, &v1alpha1.KeycloakClient{})
	client := fake.NewFakeClient(&secret, &k, &kr, &creatorSecret, &readerSecret)

	testRealm := dto.Realm{Name: realmName, SsoRealmEnabled: true, SsoAutoRedirectEnabled: true}
	kClient := new(mock.KeycloakClient)
	kClient.On("DeleteRealm", "test.test").Return(nil)
	kClient.On("ExistRealm", testRealm.Name).
		Return(false, nil)
	kClient.On(
		"CreateRealmWithDefaultConfig", &dto.Realm{Name: realmName, SsoRealmEnabled: true, SsoAutoRedirectEnabled: true,
			ACCreatorPass: "test", ACReaderPass: "test"}).Return(nil)
	kClient.On("CreateClientScope", realmName, model.ClientScope{
		Name:        gocloak.StringP("edp"),
		Description: gocloak.StringP("default edp scope required for ac and nexus"),
		Protocol:    gocloak.StringP("openid-connect"),
		ClientScopeAttributes: &model.ClientScopeAttributes{
			IncludeInTokenScope: gocloak.StringP("true"),
		},
	}).Return(nil)
	kClient.On("GetOpenIdConfig", &testRealm).
		Return("fooClient", nil)
	kClient.On("ExistCentralIdentityProvider", &testRealm).Return(true, nil)
	kClient.On("PutDefaultIdp", &testRealm).Return(nil)
	kClient.On("SyncRealmIdentityProviderMappers", kr.Spec.RealmName,
		dto.ConvertSSOMappersToIdentityProviderMappers(kr.Spec.SsoRealmName, ssoRealmMappers)).Return(nil)
	factory := new(mock.GoCloakFactory)

	factory.On("New", dto.Keycloak{User: "test", Pwd: "test"}).
		Return(kClient, nil)

	nsName := types.NamespacedName{Name: kRealmName, Namespace: ns}
	req := reconcile.Request{NamespacedName: nsName}
	r := ReconcileKeycloakRealm{client: client, factory: factory, helper: helper.MakeHelper(client, s),
		handler: chain.CreateDefChain(client, s)}

	if _, err := r.Reconcile(req); err != nil {
		t.Fatal(err)
	}

	var checkRealm v1alpha1.KeycloakRealm
	if err := client.Get(context.Background(), nsName, &checkRealm); err != nil {
		t.Fatal(err)
	}

	if label, ok := checkRealm.Labels[chain.TargetRealmLabel]; !ok || label == "" || label != checkRealm.Spec.RealmName {
		t.Fatal("target realm label is not set")
	}
}
