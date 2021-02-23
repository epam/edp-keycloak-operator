package chain

import (
	"context"
	"fmt"

	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"github.com/pkg/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var clientSecretName = "keycloak-client.%s.secret"

type PutKeycloakClientCR struct {
	next   handler.RealmHandler
	client client.Client
	scheme *runtime.Scheme
}

func (h PutKeycloakClientCR) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start creation of Keycloak client CR")
	kc, err := helper.GetKeycloakClientCR(h.client, types.NamespacedName{
		Namespace: realm.Namespace,
		Name:      realm.Spec.RealmName,
	})
	if err != nil {
		return errors.Wrap(err, "unable to get kc client cr")
	}
	if kc != nil {
		rLog.Info("Required Keycloak client CR already exists")
		return nextServeOrNil(h.next, realm, kClient)
	}

	kc = &v1alpha1.KeycloakClient{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      realm.Spec.RealmName,
			Namespace: realm.Namespace,
		},
		Spec: v1alpha1.KeycloakClientSpec{
			Secret:      fmt.Sprintf(clientSecretName, realm.Spec.RealmName),
			TargetRealm: realm.Spec.RealmName,
			ClientId:    realm.Spec.RealmName,
			ClientRoles: []string{"administrator", "developer"},
		},
	}
	if realm.Spec.SsoRealmEnabled == nil || *realm.Spec.SsoRealmEnabled {
		kc.Spec.TargetRealm = realm.Spec.SsoRealmName
	}

	err = controllerutil.SetControllerReference(realm, kc, h.scheme)
	if err != nil {
		return errors.Wrap(err, "cannot set owner ref for keycloak client CR")
	}
	err = h.client.Create(context.TODO(), kc)
	if err != nil {
		return errors.Wrap(err, "cannot create keycloak client cr")
	}
	rLog.Info("Keycloak client has been successfully created", "keycloak client", kc)
	return nextServeOrNil(h.next, realm, kClient)
}
