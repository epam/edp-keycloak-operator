package chain

import (
	"context"
	"fmt"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"github.com/google/uuid"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type PutKeycloakClientSecret struct {
	next   handler.RealmHandler
	client client.Client
	scheme *runtime.Scheme
}

func (h PutKeycloakClientSecret) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start creation of Keycloak client secret")
	if !realm.Spec.SSOEnabled() {
		rLog.Info("sso realm disabled skip creation of Keycloak client secret")
		return nextServeOrNil(h.next, realm, kClient)
	}

	sn := fmt.Sprintf(clientSecretName, realm.Spec.RealmName)
	s, err := helper.GetSecret(h.client, types.NamespacedName{
		Name:      sn,
		Namespace: realm.Namespace,
	})
	if err != nil {
		return err
	}
	if s != nil {
		rLog.Info("Keycloak client secret already exist")
		return nextServeOrNil(h.next, realm, kClient)
	}
	s = &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sn,
			Namespace: realm.Namespace,
		},
		Data: map[string][]byte{
			"clientSecret": []byte(uuid.New().String()),
		},
	}
	cl := &v1alpha1.KeycloakClient{}
	err = h.client.Get(context.TODO(), types.NamespacedName{
		Namespace: realm.Namespace,
		Name:      realm.Spec.RealmName,
	}, cl)
	if err != nil {
		return err
	}
	err = controllerutil.SetControllerReference(cl, s, h.scheme)
	if err != nil {
		return err
	}
	err = h.client.Create(context.TODO(), s)
	if err != nil {
		return err
	}
	rLog.Info("End of put Keycloak client secret")
	return nextServeOrNil(h.next, realm, kClient)
}
