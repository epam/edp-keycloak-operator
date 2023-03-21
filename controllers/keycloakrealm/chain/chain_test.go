package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

func TestCreateDefChain(t *testing.T) {
	ns, kSecretName, kServerUsr, kServerPwd, kRealmName, realmName := "test", "test", "test", "test", "test", "test.test"
	k := keycloakApi.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "test-keycloak", Namespace: ns},
		Spec: keycloakApi.KeycloakSpec{Secret: kSecretName}, Status: keycloakApi.KeycloakStatus{Connected: true},
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: kSecretName, Namespace: ns}, Data: map[string][]byte{
		"username": []byte(kServerUsr), "password": []byte(kServerPwd)}}

	clientSecret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-client.test.test.secret", Namespace: ns},
		Data: map[string][]byte{keycloakApi.ClientSecretKey: []byte(kServerUsr)}}

	ssoEnable := true

	kr := keycloakApi.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kRealmName,
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakRealmSpec{
			KeycloakOwner:   k.Name,
			RealmName:       fmt.Sprintf("%v.%v", ns, kRealmName),
			SsoRealmEnabled: &ssoEnable,
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &k, &kr, &keycloakApi.KeycloakClient{})
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
		adapter.NotFoundError("not found"))
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

	hm := helper.Mock{}

	hm.On("InvalidateKeycloakClientTokenSecret", k.Namespace, k.Name).Return(nil)
	chain := CreateDefChain(client, s, &hm)
	err := chain.ServeRequest(context.Background(), &kr, kClient)
	require.NoError(t, err)
}

func TestCreateDefChain_SSORealm(t *testing.T) {
	ns, kSecretName, kServerUsr, kServerPwd, kRealmName, realmName := "test", "test", "test", "test", "test", "test.test"
	k := keycloakApi.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "test-keycloak", Namespace: ns},
		Spec: keycloakApi.KeycloakSpec{Secret: kSecretName}, Status: keycloakApi.KeycloakStatus{Connected: true},
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: kSecretName, Namespace: ns}, Data: map[string][]byte{
		"username": []byte(kServerUsr), "password": []byte(kServerPwd)}}

	ssoEnabled := true

	kr := keycloakApi.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns},
		Spec: keycloakApi.KeycloakRealmSpec{
			KeycloakOwner:   k.Name,
			RealmName:       fmt.Sprintf("%v.%v", ns, kRealmName),
			SsoRealmName:    "openshift",
			SsoRealmEnabled: &ssoEnabled,
			Users: []keycloakApi.User{
				{RealmRoles: []string{"foo", "bar"}},
			}},
	}

	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &k, &kr, &keycloakApi.KeycloakClient{})
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
		adapter.NotFoundError("not found"))
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

	hm := helper.Mock{}
	hm.On("InvalidateKeycloakClientTokenSecret", k.Namespace, k.Name).Return(nil)
	chain := CreateDefChain(client, s, &hm)
	err := chain.ServeRequest(context.Background(), &kr, kClient)
	require.NoError(t, err)
}

func TestCreateDefChainNoSSO(t *testing.T) {
	ns, kSecretName, kServerUsr, kServerPwd, kRealmName, realmName := "test", "test", "test", "test", "test", "test.test"
	k := keycloakApi.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "test-keycloak", Namespace: ns},
		Spec: keycloakApi.KeycloakSpec{Secret: kSecretName}, Status: keycloakApi.KeycloakStatus{Connected: true},
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: kSecretName, Namespace: ns}, Data: map[string][]byte{
		"username": []byte(kServerUsr), "password": []byte(kServerPwd)}}

	kr := keycloakApi.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: kRealmName, Namespace: ns},
		Spec: keycloakApi.KeycloakRealmSpec{KeycloakOwner: k.Name, RealmName: fmt.Sprintf("%v.%v", ns, kRealmName),
			SsoRealmEnabled: gocloak.BoolP(false), Users: []keycloakApi.User{
				{RealmRoles: []string{"foo", "bar"}},
			}},
	}

	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &k, &kr, &keycloakApi.KeycloakClient{})
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
		adapter.NotFoundError("not found"))
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

	hm := helper.Mock{}

	hm.On("InvalidateKeycloakClientTokenSecret", k.Namespace, k.Name).Return(nil)
	chain := CreateDefChain(client, s, &hm)
	err := chain.ServeRequest(context.Background(), &kr, kClient)
	require.NoError(t, err)
}
