package chain

import (
	"fmt"
	"testing"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
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

	clientSecret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-client.test.test.secret", Namespace: ns},
		Data: map[string][]byte{"clientSecret": []byte(kServerUsr)}}

	kr := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns},
		Spec: v1alpha1.KeycloakRealmSpec{KeycloakOwner: k.Name, RealmName: fmt.Sprintf("%v.%v", ns, kRealmName)},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &k, &kr, &v1alpha1.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(&secret, &k, &kr, &clientSecret).Build()

	testRealm := dto.Realm{Name: realmName, SsoRealmEnabled: true, SsoAutoRedirectEnabled: true}
	kClient := new(adapter.Mock)
	kClient.On("DeleteRealm", "test.test").Return(nil)
	kClient.On("ExistRealm", testRealm.Name).
		Return(false, nil)
	kClient.On(
		"CreateRealmWithDefaultConfig", &dto.Realm{Name: realmName, SsoRealmEnabled: true,
			SsoAutoRedirectEnabled: true}).
		Return(nil)
	kClient.On("GetClientScope", "edp", "test.test").Return(nil,
		adapter.ErrNotFound("not found"))
	kClient.On("CreateClientScope", realmName, &adapter.ClientScope{
		Name:        "edp",
		Description: "default edp scope required for ac and nexus",
		Protocol:    "openid-connect",
		Attributes: map[string]string{
			"include.in.token.scope": "true",
		},
	}).Return("", nil)
	kClient.On("GetOpenIdConfig", &testRealm).
		Return("fooClient", nil)
	kClient.On("ExistCentralIdentityProvider", &testRealm).Return(false, nil)
	kClient.On("PutDefaultIdp", &testRealm).Return(nil)
	kClient.On("CreateCentralIdentityProvider", &testRealm, &dto.Client{ClientId: "test.test",
		ClientSecret: "test", RealmRole: dto.IncludedRealmRole{}}).
		Return(nil)

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

	kr := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns},
		Spec: v1alpha1.KeycloakRealmSpec{KeycloakOwner: k.Name, RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
			SsoRealmName: "openshift",
			Users: []v1alpha1.User{
				{RealmRoles: []string{"foo", "bar"}},
			}},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &k, &kr, &v1alpha1.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(&secret, &k, &kr).Build()

	realmUser := dto.User{RealmRoles: []string{"foo", "bar"}}
	testRealm := dto.Realm{Name: realmName, SsoRealmEnabled: true, SsoRealmName: "openshift", SsoAutoRedirectEnabled: true,
		Users: []dto.User{realmUser}}
	kClient := new(adapter.Mock)
	kClient.On("DeleteRealm", "test.test").Return(nil)
	kClient.On("ExistRealm", testRealm.Name).
		Return(false, nil)

	kClient.On(
		"CreateRealmWithDefaultConfig", &dto.Realm{Name: realmName, SsoRealmEnabled: true, SsoRealmName: "openshift",
			SsoAutoRedirectEnabled: true, Users: []dto.User{realmUser}}).
		Return(nil)
	kClient.On("GetClientScope", "edp", "test.test").Return(nil,
		adapter.ErrNotFound("not found"))
	kClient.On("CreateClientScope", realmName, &adapter.ClientScope{
		Name:        "edp",
		Description: "default edp scope required for ac and nexus",
		Protocol:    "openid-connect",
		Attributes: map[string]string{
			"include.in.token.scope": "true",
		},
	}).Return("", nil)
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

	kr := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns},
		Spec: v1alpha1.KeycloakRealmSpec{KeycloakOwner: k.Name, RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
			SsoRealmEnabled: gocloak.BoolP(false), Users: []v1alpha1.User{
				{RealmRoles: []string{"foo", "bar"}},
			}},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &k, &kr, &v1alpha1.KeycloakClient{})
	client := fake.NewClientBuilder().WithRuntimeObjects(&secret, &k, &kr).Build()

	testRealm := dto.Realm{Name: realmName, SsoRealmEnabled: false, Users: []dto.User{{}}}
	kClient := new(adapter.Mock)
	kClient.On("DeleteRealm", "test.test").Return(nil)
	kClient.On("ExistRealm", testRealm.Name).
		Return(false, nil)
	kClient.On(
		"CreateRealmWithDefaultConfig", &dto.Realm{Name: realmName, SsoRealmEnabled: false,
			SsoAutoRedirectEnabled: true, Users: []dto.User{{RealmRoles: []string{"foo", "bar"}}}}).
		Return(nil)
	kClient.On("GetClientScope", "edp", "test.test").Return(nil,
		adapter.ErrNotFound("not found"))
	kClient.On("CreateClientScope", realmName, &adapter.ClientScope{
		Name:        "edp",
		Description: "default edp scope required for ac and nexus",
		Protocol:    "openid-connect",
		Attributes: map[string]string{
			"include.in.token.scope": "true",
		},
	}).Return("", nil)
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
	kClient.On("AddRealmRoleToUser", testRealm.Name, realmUser.Username, "foo").Return(nil)

	chain := CreateDefChain(client, s)
	if err := chain.ServeRequest(&kr, kClient); err != nil {
		t.Fatal(err)
	}
}
