package chain

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/keycloak-operator/v2/pkg/apis/v1/v1alpha1"
	"github.com/epam/keycloak-operator/v2/pkg/client/keycloak/adapter"
	"github.com/epam/keycloak-operator/v2/pkg/client/keycloak/dto"
	"github.com/epam/keycloak-operator/v2/pkg/client/keycloak/mock"
	"github.com/epam/keycloak-operator/v2/pkg/model"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateDefChain(t *testing.T) {
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

	clientSecret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-client.test.test.secret", Namespace: ns},
		Data: map[string][]byte{"clientSecret": []byte(kServerUsr)}}

	kr := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns},
		Spec: v1alpha1.KeycloakRealmSpec{KeycloakOwner: k.Name, RealmName: fmt.Sprintf("%v.%v", ns, kRealmName)},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &k, &kr, &v1alpha1.KeycloakClient{})
	client := fake.NewFakeClient(&secret, &k, &kr, &creatorSecret, &readerSecret, &clientSecret)

	testRealm := dto.Realm{Name: realmName, SsoRealmEnabled: true, SsoAutoRedirectEnabled: true}
	kClient := new(mock.KeycloakClient)
	kClient.On("DeleteRealm", "test.test").Return(nil)
	kClient.On("ExistRealm", testRealm.Name).
		Return(false, nil)
	kClient.On(
		"CreateRealmWithDefaultConfig", &dto.Realm{Name: realmName, SsoRealmEnabled: true,
			SsoAutoRedirectEnabled: true,
			ACCreatorPass:          "test", ACReaderPass: "test"}).
		Return(nil)
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
	kClient.On("ExistCentralIdentityProvider", &testRealm).Return(false, nil)
	kClient.On("PutDefaultIdp", &testRealm).Return(nil)
	kClient.On("CreateCentralIdentityProvider", &testRealm, &dto.Client{ClientId: "test.test",
		ClientSecret: "test", RealmRole: dto.IncludedRealmRole{}}).
		Return(nil)
	factory := new(mock.GoCloakFactory)

	factory.On("New", dto.Keycloak{User: "test", Pwd: "test"}).
		Return(kClient, nil)

	chain := CreateDefChain(client, s)
	if err := chain.ServeRequest(&kr, kClient); err != nil {
		t.Fatal(err)
	}
}

func TestCreateDefChain2(t *testing.T) {
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

	kr := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns},
		Spec: v1alpha1.KeycloakRealmSpec{KeycloakOwner: k.Name, RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
			SsoRealmName: "openshift",
			Users: []v1alpha1.User{
				{RealmRoles: []string{"foo", "bar"}},
			}},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &k, &kr, &v1alpha1.KeycloakClient{})
	client := fake.NewFakeClient(&secret, &k, &kr, &creatorSecret, &readerSecret)

	realmUser := dto.User{RealmRoles: []string{"foo", "bar"}}
	testRealm := dto.Realm{Name: realmName, SsoRealmEnabled: true, SsoRealmName: "openshift", SsoAutoRedirectEnabled: true,
		Users: []dto.User{realmUser}}
	kClient := new(mock.KeycloakClient)
	kClient.On("DeleteRealm", "test.test").Return(nil)
	kClient.On("ExistRealm", testRealm.Name).
		Return(false, nil)

	kClient.On(
		"CreateRealmWithDefaultConfig", &dto.Realm{Name: realmName, SsoRealmEnabled: true, SsoRealmName: "openshift",
			SsoAutoRedirectEnabled: true,
			ACCreatorPass:          "test", ACReaderPass: "test", Users: []dto.User{realmUser}}).
		Return(nil)
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
	kClient.On("CreateCentralIdentityProvider", &testRealm, &dto.Client{ClientId: "test.test",
		ClientSecret: "test", RealmRole: dto.IncludedRealmRole{}}).
		Return(nil)
	kClient.On("ExistRealmUser", "openshift", &realmUser).Return(true, nil)
	kClient.On("HasUserClientRole", "openshift", "test.test", &realmUser, "foo").
		Return(true, nil)
	kClient.On("HasUserClientRole", "openshift", "test.test", &realmUser, "bar").
		Return(false, nil)
	kClient.On("AddClientRoleToUser", "openshift", "test.test", &realmUser, "bar").Return(nil)
	factory := new(mock.GoCloakFactory)

	factory.On("New", dto.Keycloak{User: "test", Pwd: "test"}).
		Return(kClient, nil)

	chain := CreateDefChain(client, s)
	if err := chain.ServeRequest(&kr, kClient); err != nil {
		t.Fatal(err)
	}
}

func TestCreateDefChainNoSSO(t *testing.T) {
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

	kr := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns},
		Spec: v1alpha1.KeycloakRealmSpec{KeycloakOwner: k.Name, RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
			SsoRealmEnabled: gocloak.BoolP(false), Users: []v1alpha1.User{
				{RealmRoles: []string{"foo", "bar"}},
			}},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &k, &kr, &v1alpha1.KeycloakClient{})
	client := fake.NewFakeClient(&secret, &k, &kr, &creatorSecret, &readerSecret)

	testRealm := dto.Realm{Name: realmName, SsoRealmEnabled: false, Users: []dto.User{{}}}
	kClient := new(mock.KeycloakClient)
	kClient.On("DeleteRealm", "test.test").Return(nil)
	kClient.On("ExistRealm", testRealm.Name).
		Return(false, nil)
	kClient.On(
		"CreateRealmWithDefaultConfig", &dto.Realm{Name: realmName, SsoRealmEnabled: false, SsoAutoRedirectEnabled: true,
			ACCreatorPass: "test", ACReaderPass: "test", Users: []dto.User{{RealmRoles: []string{"foo", "bar"}}}}).
		Return(nil)
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
	kClient.On("CreateCentralIdentityProvider", &testRealm, &dto.Client{ClientId: "test.test",
		ClientSecret: "test", RealmRole: dto.IncludedRealmRole{}}).
		Return(nil)
	realmUser := dto.User{RealmRoles: []string{"foo", "bar"}}
	kClient.On("ExistRealmUser", testRealm.Name, &realmUser).
		Return(false, nil)
	kClient.On("CreateRealmUser", testRealm.Name, &realmUser).Return(nil)
	kClient.On("ExistRealmRole", testRealm.Name, "foo").Return(false, nil)
	kClient.On("ExistRealmRole", testRealm.Name, "bar").Return(true, nil)
	kClient.On("CreateIncludedRealmRole", testRealm.Name, &dto.IncludedRealmRole{Name: "foo"}).Return(nil)
	kClient.On("HasUserRealmRole", testRealm.Name, &realmUser, "foo").Return(false, nil)
	kClient.On("HasUserRealmRole", testRealm.Name, &realmUser, "bar").Return(true, nil)
	kClient.On("AddRealmRoleToUser", testRealm.Name, &realmUser, "foo").Return(nil)
	factory := new(mock.GoCloakFactory)

	factory.On("New", dto.Keycloak{User: "test", Pwd: "test"}).
		Return(kClient, nil)

	chain := CreateDefChain(client, s)
	if err := chain.ServeRequest(&kr, kClient); err != nil {
		t.Fatal(err)
	}
}

func TestPutAcSecret_ServeRequest(t *testing.T) {
	kr := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: v1alpha1.KeycloakRealmSpec{KeycloakOwner: "test", RealmName: "test",
			SsoRealmEnabled: gocloak.BoolP(false), Users: []v1alpha1.User{
				{RealmRoles: []string{"foo", "bar"}},
			}},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &kr)
	client := fake.NewFakeClient(&kr)

	acSecret := PutAcSecret{
		client: client,
	}

	kClient := new(mock.KeycloakClient)

	if err := acSecret.ServeRequest(&kr, kClient); err != nil {
		t.Fatal(err)
	}

	var k8sAcSecret corev1.Secret
	if err := client.Get(context.Background(), types.NamespacedName{
		Name:      adapter.AcReaderUsername,
		Namespace: "test",
	}, &k8sAcSecret); err != nil {
		t.Fatal("creator secret not found")
	}
}
