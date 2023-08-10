package chain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

var clientSecretName = "keycloak-client.%s.secret"

type PutKeycloakClientCR struct {
	next   handler.RealmHandler
	client client.Client
	scheme *runtime.Scheme
}

func (h PutKeycloakClientCR) getClientRoles(realm *keycloakApi.KeycloakRealm) []string {
	userRoles := make(map[string]string)

	for _, u := range realm.Spec.Users {
		for _, r := range u.RealmRoles {
			userRoles[r] = r
		}
	}

	clientRoles := make([]string, 0, len(userRoles))
	for _, v := range userRoles {
		clientRoles = append(clientRoles, v)
	}

	return clientRoles
}

func (h PutKeycloakClientCR) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start creation of Keycloak client CR")

	if !realm.Spec.SSOEnabled() {
		rLog.Info("sso realm disabled skip creation of Keycloak client CR")
		return nextServeOrNil(ctx, h.next, realm, kClient)
	}

	kc := &keycloakApi.KeycloakClient{}
	err := h.client.Get(ctx, types.NamespacedName{
		Namespace: realm.Namespace,
		Name:      realm.Spec.RealmName,
	}, kc)

	if err == nil {
		rLog.Info("Required Keycloak client CR already exists")
		return nextServeOrNil(ctx, h.next, realm, kClient)
	}

	if !k8sErrors.IsNotFound(err) {
		return errors.Wrap(err, "unable to get kc client cr")
	}

	kc = &keycloakApi.KeycloakClient{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      realm.Spec.RealmName,
			Namespace: realm.Namespace,
		},
		Spec: keycloakApi.KeycloakClientSpec{
			Secret:      fmt.Sprintf(clientSecretName, realm.Spec.RealmName),
			TargetRealm: realm.Spec.SsoRealmName,
			ClientId:    realm.Spec.RealmName,
			ClientRoles: h.getClientRoles(realm),
		},
	}

	err = controllerutil.SetControllerReference(realm, kc, h.scheme)
	if err != nil {
		return errors.Wrap(err, "cannot set owner ref for keycloak client CR")
	}

	err = h.client.Create(ctx, kc)
	if err != nil {
		return errors.Wrap(err, "cannot create keycloak client cr")
	}

	rLog.Info("Keycloak client has been successfully created", "keycloak client", kc)

	return nextServeOrNil(ctx, h.next, realm, kClient)
}
