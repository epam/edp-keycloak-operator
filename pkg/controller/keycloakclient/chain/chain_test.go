package chain

import (
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestMake(t *testing.T) {
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
	h := helper.MakeHelper(client, s)

	keycloakDto := dto.Keycloak{
		Url:  "https://some",
		User: "user",
		Pwd:  "pass",
	}
	kClient := new(mock.KeycloakClient)
	factory := new(mock.GoCloakFactory)
	factory.On("New", keycloakDto).
		Return(kClient, nil)

	chain := Make(h, client, logf.Log.WithName("controller_keycloakclient"), factory)

	clientDTO := dto.ConvertSpecToClient(&kc.Spec, "")
	kClient.On("ExistClient", clientDTO).
		Return(false, nil)
	kClient.On("CreateClient", clientDTO).Return(nil)
	kClient.On("GetClientID", clientDTO).Return("3333", nil)
	kClient.On("ExistRealmRole", kr.Spec.RealmName, "fake-client-users").
		Return(true, nil)
	kClient.On("ExistRealmRole", kr.Spec.RealmName, "fake-client-administrators").
		Return(false, nil)
	kClient.On("SyncClientProtocolMapper", clientDTO, []gocloak.ProtocolMapperRepresentation{
		{Name: gocloak.StringP("bar"), Protocol: gocloak.StringP(""), Config: &map[string]string{"bar": "1"},
			ProtocolMapper: gocloak.StringP("")},
		{Name: gocloak.StringP("foo"), Protocol: gocloak.StringP(""), Config: &map[string]string{"foo": "2"},
			ProtocolMapper: gocloak.StringP("")},
	}).Return(nil)

	role1DTO := dto.IncludedRealmRole{Name: "fake-client-administrators", Composite: "administrator"}
	kClient.On("CreateIncludedRealmRole", kr.Spec.RealmName, &role1DTO).Return(nil)

	if err := chain.Serve(&kc); err != nil {
		t.Fatal(err)
	}

	if kc.Status.Id != "3333" {
		t.Fatal("keycloak client status not changed")
	}
}
