package chain

import (
	"github.com/epam/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/keycloak-operator/pkg/client/keycloak"
	"github.com/epam/keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/keycloak-operator/pkg/controller/helper"
	"github.com/epam/keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PutRealm struct {
	next   handler.RealmHandler
	client client.Client
}

func (h PutRealm) putRealmRoles(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	allRoles := make(map[string]string)
	//check if all user roles exists
	for _, u := range realm.Spec.Users {
		for _, rr := range u.RealmRoles {
			if _, ok := allRoles[rr]; !ok {
				allRoles[rr] = rr
			}
		}
	}

	dtoRealm := dto.ConvertSpecToRealm(realm.Spec)

	for _, r := range allRoles {
		exists, err := kClient.ExistRealmRole(dtoRealm.Name, r)
		if err != nil {
			return errors.Wrap(err, "unable to check realm role existence")
		}

		if !exists {
			if err := kClient.CreateIncludedRealmRole(dtoRealm.Name, &dto.IncludedRealmRole{Name: r}); err != nil {
				return errors.Wrap(err, "unable to create new realm role")
			}
		}
	}

	return nil
}

func (h PutRealm) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start putting realm")
	rDto := dto.ConvertSpecToRealm(realm.Spec)
	e, err := kClient.ExistRealm(rDto.Name)
	if err != nil {
		return errors.Wrap(err, "unable to check realm existence")
	}
	if e {
		rLog.Info("Realm already exists")
		return nextServeOrNil(h.next, realm, kClient)
	}
	crS, err := helper.GetSecret(h.client, types.NamespacedName{
		Namespace: realm.Namespace,
		Name:      adapter.AcCreatorUsername,
	})
	if err != nil {
		return errors.Wrap(err, "unable to get ac-creator secret")
	}
	rDto.ACCreatorPass = string(crS.Data["password"])
	rS, err := helper.GetSecret(h.client, types.NamespacedName{
		Namespace: realm.Namespace,
		Name:      adapter.AcReaderUsername,
	})
	if err != nil {
		return errors.Wrap(err, "unable to get ac-reader secret")
	}
	rDto.ACReaderPass = string(rS.Data["password"])
	err = kClient.CreateRealmWithDefaultConfig(rDto)
	if err != nil {
		return errors.Wrap(err, "unable to create realm with default config")
	}
	//put realm roles if sso realm is disabled
	if !rDto.SsoRealmEnabled {
		if err := h.putRealmRoles(realm, kClient); err != nil {
			return errors.Wrap(err, "unable to create realm roles on no sso scenario")
		}
	}
	rLog.Info("End putting realm!")
	return nextServeOrNil(h.next, realm, kClient)
}
