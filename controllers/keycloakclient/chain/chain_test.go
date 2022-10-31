package chain

import (
	"context"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v10"
	"github.com/stretchr/testify/assert"
	testifyMock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestPrivateClientSecret(t *testing.T) {
	kc := keycloakApi.KeycloakClient{ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "namespace"},
		Spec: keycloakApi.KeycloakClientSpec{TargetRealm: "namespace.main", Secret: "keycloak-secret",
			RealmRoles: &[]keycloakApi.RealmRole{{Name: "fake-client-administrators", Composite: "administrator"},
				{Name: "fake-client-users", Composite: "developer"},
			}, Public: false, ClientId: "fake-client", WebUrl: "fake-url", DirectAccess: false,
			AdvancedProtocolMappers: true, ClientRoles: nil, ProtocolMappers: &[]keycloakApi.ProtocolMapper{
				{Name: "bar", Config: map[string]string{"bar": "1"}},
				{Name: "foo", Config: map[string]string{"foo": "2"}},
			},
		},
	}

	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: "namespace"},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &kc)
	client := fake.NewClientBuilder().WithRuntimeObjects(&kc, &secret).Build()
	h := helper.MakeHelper(client, s, mock.NewLogr())

	clientDTO := dto.ConvertSpecToClient(&kc.Spec, "")

	kClient := new(adapter.Mock)
	kClient.On("ExistClient", clientDTO.ClientId, clientDTO.RealmName).Return(true, nil)
	kClient.On("GetClientID", clientDTO.ClientId, clientDTO.RealmName).Return("3333", nil)
	kClient.On("UpdateClient", testifyMock.Anything).Return(nil)

	baseElement := BaseElement{
		scheme: h.GetScheme(),
		Client: client,
		Logger: mock.NewLogr(),
	}
	putCl := PutClient{
		BaseElement: baseElement,
	}

	ctx := context.Background()

	if err := putCl.Serve(ctx, &kc, kClient); err != nil {
		t.Fatalf("%+v", err)
	}

	kc.Spec.Secret = ""

	if err := putCl.Serve(ctx, &kc, kClient); err != nil {
		t.Fatalf("%+v", err)
	}

	var (
		checkSecret corev1.Secret
		checkClient keycloakApi.KeycloakClient
	)

	err := client.Get(context.Background(), types.NamespacedName{Name: kc.Name, Namespace: kc.Namespace},
		&checkClient)
	require.NoError(t, err)

	if kc.Spec.Secret == "" || kc.Status.ClientSecretName == "" {
		t.Fatal("client secret not updated")
	}

	err = client.Get(context.Background(), types.NamespacedName{Namespace: checkClient.Namespace,
		Name: checkClient.Spec.Secret}, &checkSecret)
	require.NoError(t, err)

	if _, ok := checkSecret.Data[clientSecretKey]; !ok {
		t.Fatal("client secret key not found in secret")
	}
}

func TestMake(t *testing.T) {
	k := keycloakApi.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "test-keycloak", Namespace: "namespace"},
		Spec:   keycloakApi.KeycloakSpec{Url: "https://some", Secret: "keycloak-secret"},
		Status: keycloakApi.KeycloakStatus{Connected: true}}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: "namespace"},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}
	kr := keycloakApi.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "namespace",
		OwnerReferences: []metav1.OwnerReference{{Name: "test-keycloak", Kind: "Keycloak"}}},
		Spec: keycloakApi.KeycloakRealmSpec{RealmName: "namespace.main"},
	}
	delTime := metav1.Time{Time: time.Now()}
	kc := keycloakApi.KeycloakClient{ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "namespace",
		DeletionTimestamp: &delTime},
		Spec: keycloakApi.KeycloakClientSpec{TargetRealm: "namespace.main", Secret: "keycloak-secret",
			RealmRoles: &[]keycloakApi.RealmRole{{Name: "fake-client-administrators", Composite: "administrator"},
				{Name: "fake-client-users", Composite: "developer"},
			}, Public: true, ClientId: "fake-client", WebUrl: "fake-url", DirectAccess: false,
			AdvancedProtocolMappers: true, ClientRoles: nil, ProtocolMappers: &[]keycloakApi.ProtocolMapper{
				{Name: "bar", Config: map[string]string{"bar": "1"}},
				{Name: "foo", Config: map[string]string{"foo": "2"}},
			},
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &k, &kr, &kc, &keycloakApi.KeycloakRealm{}, &keycloakApi.KeycloakRealmList{})
	client := fake.NewClientBuilder().WithRuntimeObjects(&secret, &k, &kr, &kc).Build()
	h := helper.MakeHelper(client, s, mock.NewLogr())

	kClient := new(adapter.Mock)
	chain := Make(h.GetScheme(), client, mock.NewLogr())

	clientDTO := dto.ConvertSpecToClient(&kc.Spec, "")
	kClient.On("ExistClient", clientDTO.ClientId, clientDTO.RealmName).
		Return(false, nil)
	kClient.On("CreateClient", clientDTO).Return(nil)
	kClient.On("GetClientID", clientDTO.ClientId, clientDTO.RealmName).Return("3333", nil)
	kClient.On("UpdateClient", testifyMock.Anything).Return(nil)
	kClient.On("ExistRealmRole", kr.Spec.RealmName, "fake-client-users").
		Return(true, nil)
	kClient.On("ExistRealmRole", kr.Spec.RealmName, "fake-client-administrators").
		Return(false, nil)
	kClient.On("SyncClientProtocolMapper", clientDTO, []gocloak.ProtocolMapperRepresentation{
		{Name: gocloak.StringP("bar"), Protocol: gocloak.StringP(""), Config: &map[string]string{"bar": "1"},
			ProtocolMapper: gocloak.StringP("")},
		{Name: gocloak.StringP("foo"), Protocol: gocloak.StringP(""), Config: &map[string]string{"foo": "2"},
			ProtocolMapper: gocloak.StringP("")},
	}, false).Return(nil)

	role1DTO := dto.IncludedRealmRole{Name: "fake-client-administrators", Composite: "administrator"}
	kClient.On("CreateIncludedRealmRole", kr.Spec.RealmName, &role1DTO).Return(nil)

	err := chain.Serve(context.Background(), &kc, kClient)
	require.NoError(t, err)

	if kc.Status.ClientID != "3333" {
		t.Fatal("keycloak client status not changed")
	}
}

func TestPutClientScope_Serve(t *testing.T) {
	pcs := PutClientScope{}
	kc := keycloakApi.KeycloakClient{Spec: keycloakApi.KeycloakClientSpec{ClientId: "clid1", TargetRealm: "realm1"}}
	kClient := new(adapter.Mock)
	adapterScope := adapter.ClientScope{ID: "scope-id1"}

	ctx := context.Background()

	kClient.On("GetClientScopesByNames", ctx, adapterScope.ID, kc.Spec.TargetRealm).Return([]adapter.ClientScope{adapterScope}, nil)
	kClient.On("AddDefaultScopeToClient", ctx, kc.Spec.TargetRealm, adapterScope.ID).
		Return(nil)

	err := pcs.putClientScope(ctx, &kc, kClient)
	assert.NoError(t, err)
}
