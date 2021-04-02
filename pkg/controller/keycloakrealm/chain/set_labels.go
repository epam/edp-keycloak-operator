package chain

import (
	"context"

	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const TargetRealmLabel = "targetRealm"

type SetLabels struct {
	next   handler.RealmHandler
	client client.Client
}

func (s SetLabels) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	if realm.Labels == nil {
		realm.Labels = make(map[string]string)
	}

	if tr, ok := realm.Labels[TargetRealmLabel]; !ok || tr != realm.Spec.RealmName {
		realm.Labels[TargetRealmLabel] = realm.Spec.RealmName
	}

	if err := s.client.Update(context.Background(), realm); err != nil {
		return errors.Wrapf(err, "unable to update realm with new labels, realm: %+v", realm)
	}

	return nextServeOrNil(s.next, realm, kClient)
}
