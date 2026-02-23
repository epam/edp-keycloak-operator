package chain

import (
	"context"
	"fmt"
	"testing"

	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestCreateDefChain(t *testing.T) {
	ns, kSecretName, kServerUsr, kServerPwd, kRealmName, realmName := "test", "test-secret", "test-usr", "test-pwd", "test", "test.test"
	k := keycloakApi.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "test-keycloak", Namespace: ns},
		Spec: keycloakApi.KeycloakSpec{Secret: kSecretName}, Status: keycloakApi.KeycloakStatus{Connected: true},
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: kSecretName, Namespace: ns}, Data: map[string][]byte{
		"username": []byte(kServerUsr), "password": []byte(kServerPwd)}}

	clientSecret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-client.test.test.secret", Namespace: ns},
		Data: map[string][]byte{keycloakApi.ClientSecretKey: []byte(kServerUsr)}}

	kr := keycloakApi.KeycloakRealm{
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

	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))
	require.NoError(t, keycloakApi.AddToScheme(s))
	client := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(&secret, &k, &kr, &clientSecret).Build()

	mockRealm := v2mocks.NewMockRealmClient(t)
	// PutRealm: realm already exists
	mockRealm.EXPECT().GetRealm(testifymock.Anything, kr.Spec.RealmName).
		Return(&keycloakv2.RealmRepresentation{}, nil, nil).Once()
	// RealmSettings: GetRealm + UpdateRealm
	mockRealm.EXPECT().GetRealm(testifymock.Anything, kr.Spec.RealmName).
		Return(&keycloakv2.RealmRepresentation{}, nil, nil).Once()
	mockRealm.EXPECT().UpdateRealm(testifymock.Anything, kr.Spec.RealmName, testifymock.Anything).
		Return(nil, nil)

	_ = realmName // kept for local variable consistency
	chain := CreateDefChain(client, s)
	kClientV2 := &keycloakv2.KeycloakClient{Realms: mockRealm}
	err := chain.ServeRequest(context.Background(), &kr, kClientV2)
	require.NoError(t, err)
}
