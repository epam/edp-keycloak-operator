package chain

import (
	"context"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type Helper interface {
	InvalidateKeycloakClientTokenSecret(ctx context.Context, namespace, rootKeycloakName string) error
}

type PutRealm struct {
	next   handler.RealmHandler
	client client.Client
	hlp    Helper
}

func (h PutRealm) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start putting realm")

	rDto := dto.ConvertSpecToRealm(&realm.Spec)

	e, err := kClient.ExistRealm(rDto.Name)
	if err != nil {
		return errors.Wrap(err, "unable to check realm existence")
	}

	if e {
		rLog.Info("Realm already exists")

		return nextServeOrNil(ctx, h.next, realm, kClient)
	}

	err = kClient.CreateRealmWithDefaultConfig(rDto)
	if err != nil {
		return errors.Wrap(err, "unable to create realm with default config")
	}

	if err = h.putRealmRoles(realm, kClient); err != nil {
		return errors.Wrap(err, "unable to create realm roles on no sso scenario")
	}

	if err := h.hlp.InvalidateKeycloakClientTokenSecret(ctx, realm.Namespace, realm.Spec.KeycloakRef.Name); err != nil {
		return errors.Wrap(err, "unable invalidate keycloak client token")
	}

	rLog.Info("End putting realm!")

	return nextServeOrNil(ctx, h.next, realm, kClient)
}

func (h PutRealm) putRealmRoles(realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
	allRoles := make(map[string]string)
	// check if all user roles exists
	for _, u := range realm.Spec.Users {
		for _, rr := range u.RealmRoles {
			if _, ok := allRoles[rr]; !ok {
				allRoles[rr] = rr
			}
		}
	}

	dtoRealm := dto.ConvertSpecToRealm(&realm.Spec)

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
