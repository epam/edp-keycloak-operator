package chain

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"github.com/sethvargo/go-password/password"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PutAcSecret struct {
	client client.Client
	next   handler.RealmHandler
}

func (h PutAcSecret) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start putting Admin Console secrets...")
	err := h.putRandomSecret(types.NamespacedName{
		Namespace: realm.Namespace,
		Name:      adapter.AcCreatorUsername,
	})
	if err != nil {
		return err
	}
	err = h.putRandomSecret(types.NamespacedName{
		Namespace: realm.Namespace,
		Name:      adapter.AcReaderUsername,
	})
	if err != nil {
		return err
	}
	rLog.Info("End putting Admin Console secrets!")
	return nextServeOrNil(h.next, realm, kClient)
}

func (h PutAcSecret) putRandomSecret(nsn types.NamespacedName) error {
	s, err := helper.GetSecret(h.client, nsn)
	if err != nil {
		return err
	}
	if s != nil {
		log.Info("Secret already exists. Skip adding", "namespace", nsn.Namespace, "secret name", nsn.Name)
		return nil
	}
	pwd, err := password.Generate(13, 7, 0, false, false)
	if err != nil {
		return err
	}
	s = &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: nsn.Namespace,
			Name:      nsn.Name,
		}, Data: map[string][]byte{
			"username": []byte(nsn.Name),
			"password": []byte(pwd),
		},
	}
	return h.client.Create(context.TODO(), s)
}
