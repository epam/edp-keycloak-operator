package keycloakclient

import (
	"context"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
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

	kclient := new(mock.KeycloakClient)
	c := dto.ConvertSpecToClient(&kc.Spec, "")
	kclient.On("ExistClient", c).Return(
		false, nil)
	kclient.On("CreateClient", c).Return(
		nil)
	kclient.On("GetClientId", c).Return(
		"uuid", nil)
	rm := dto.ConvertSpecToRealm(kr.Spec)
	ar := dto.IncludedRealmRole{
		Name:      "fake-client-administrators",
		Composite: "administrator",
	}
	kclient.On("ExistRealmRole", rm.Name, ar.Name).Return(
		false, nil)
	kclient.On("CreateRealmRole", rm, ar).Return(
		nil)
	dr := dto.IncludedRealmRole{
		Name:      "fake-client-users",
		Composite: "developer",
	}
	kclient.On("ExistRealmRole", rm.Name, dr.Name).Return(
		false, nil)
	kclient.On("CreateRealmRole", rm, dr).Return(
		nil)

	keycloakDto := dto.Keycloak{
		Url:  "https://some",
		User: "user",
		Pwd:  "pass",
	}
	factory := new(mock.GoCloakFactory)
	factory.On("New", keycloakDto).
		Return(kclient, nil)

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "main",
			Namespace: "namespace",
		},
	}

	h := helper.MakeHelper(client, s)
	//reconcile
	r := ReconcileKeycloakClient{
		Client: client,
		Helper: h,
		Log:    &mock.Logger{},
	}

	//test
	res, err := r.Reconcile(context.TODO(), req)

	//verify
	assert.Nil(t, err)
	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}
	assert.False(t, res.Requeue)

	persKc := &v1alpha1.KeycloakClient{}
	err = client.Get(context.TODO(), req.NamespacedName, persKc)
	assert.Equal(t, "FAIL", persKc.Status.Value)
	assert.Empty(t, persKc.Status.ClientID)
}

func TestReconcileKeycloakClient_ReconcileWithMappers(t *testing.T) {
	k := v1alpha1.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "test-keycloak", Namespace: "namespace"},
		Spec:   v1alpha1.KeycloakSpec{Url: "https://some", Secret: "keycloak-secret"},
		Status: v1alpha1.KeycloakStatus{Connected: true}}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: "namespace"},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}
	kr := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "namespace",
		OwnerReferences: []metav1.OwnerReference{{Name: "test-keycloak", Kind: "Keycloak"}}},
		Spec: v1alpha1.KeycloakRealmSpec{RealmName: "namespace.main"},
	}
	delTime := metav1.Time{Time: time.Now()}
	kc := v1alpha1.KeycloakClient{ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "namespace",
		DeletionTimestamp: &delTime},
		Spec: v1alpha1.KeycloakClientSpec{TargetRealm: "namespace.main", Secret: "keycloak-secret",
			RealmRoles: &[]v1alpha1.RealmRole{{Name: "fake-client-administrators", Composite: "administrator"},
				{Name: "fake-client-users", Composite: "developer"},
			}, Public: true, ClientId: "fake-client", WebUrl: "fake-url", DirectAccess: false,
			AdvancedProtocolMappers: true, ClientRoles: nil, ProtocolMappers: &[]v1alpha1.ProtocolMapper{
				{Name: "bar", Config: map[string]string{"bar": "1"}},
				{Name: "foo", Config: map[string]string{"foo": "2"}},
			},
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &k, &kr, &kc)
	client := fake.NewFakeClient(&secret, &k, &kr, &kc)
	kclient := new(mock.KeycloakClient)

	clientDTO := dto.ConvertSpecToClient(&kc.Spec, "")
	realmDTO := dto.ConvertSpecToRealm(kr.Spec)
	role1DTO := dto.PrimaryRealmRole{Name: "fake-client-administrators", Composites: []string{"administrator"},
		IsComposite: true}

	kclient.On("ExistClient", clientDTO.ClientId, clientDTO.RealmName).
		Return(true, nil)
	kclient.On("GetClientID", clientDTO.ClientId, clientDTO.RealmName).
		Return("321", nil)
	kclient.On("ExistRealmRole", realmDTO.Name, role1DTO.Name).Return(true, nil)
	kclient.On("SyncClientProtocolMapper", clientDTO, []gocloak.ProtocolMapperRepresentation{
		{Name: gocloak.StringP("bar"), Protocol: gocloak.StringP(""), Config: &map[string]string{"bar": "1"},
			ProtocolMapper: gocloak.StringP("")},
		{Name: gocloak.StringP("foo"), Protocol: gocloak.StringP(""), Config: &map[string]string{"foo": "2"},
			ProtocolMapper: gocloak.StringP("")},
	}).Return(nil)
	kclient.On("DeleteClient", "321", "namespace.main").Return(nil)

	keycloakDto := dto.Keycloak{Url: "https://some", User: "user", Pwd: "pass"}
	factory := new(mock.GoCloakFactory)
	factory.On("New", keycloakDto).
		Return(kclient, nil)

	h := helper.MakeHelper(client, s)
	r := ReconcileKeycloakClient{
		Client: client,
		Helper: h,
		Log:    &mock.Logger{},
	}

	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: "main", Namespace: "namespace"}}
	_, err := r.Reconcile(context.TODO(), req)
	if err != nil {
		t.Fatal(err)
	}
}
