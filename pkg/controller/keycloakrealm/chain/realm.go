package chain

import (
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PutRealm struct {
	next   handler.RealmHandler
	client client.Client
}

func (h PutRealm) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start putting realm")
	rDto := dto.ConvertSpecToRealm(realm.Spec)
	e, err := kClient.ExistRealm(rDto)
	if err != nil {
		return err
	}
	if *e {
		rLog.Info("Realm already exists")
		return nextServeOrNil(h.next, realm, kClient)
	}
	crS, err := helper.GetSecret(h.client, types.NamespacedName{
		Namespace: realm.Namespace,
		Name:      adapter.AcCreatorUsername,
	})
	if err != nil {
		return err
	}
	rDto.ACCreatorPass = string(crS.Data["password"])
	rS, err := helper.GetSecret(h.client, types.NamespacedName{
		Namespace: realm.Namespace,
		Name:      adapter.AcReaderUsername,
	})
	if err != nil {
		return err
	}
	rDto.ACReaderPass = string(rS.Data["password"])
	err = kClient.CreateRealmWithDefaultConfig(rDto)
	if err != nil {
		return err
	}
	rLog.Info("End putting realm!")
	return nextServeOrNil(h.next, realm, kClient)
}
