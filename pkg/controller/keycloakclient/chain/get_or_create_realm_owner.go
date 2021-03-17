package chain

import (
	"context"

	v1v1alpha1 "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
)

type GetOrCreateRealmOwner struct {
	BaseElement
	next Element
}

func (g *GetOrCreateRealmOwner) Serve(keycloakClient *v1v1alpha1.KeycloakClient) error {
	realm, err := g.Helper.GetOrCreateRealmOwnerRef(keycloakClient, keycloakClient.ObjectMeta)
	if err != nil {
		return errors.Wrap(err, "unable to GetOrCreateRealmOwnerRef")
	}

	if err = g.addTargetRealmIfNeed(keycloakClient, realm.Spec.RealmName); err != nil {
		return errors.Wrap(err, "unable to addTargetRealmIfNeed")
	}

	g.State.KeycloakRealm = realm

	return g.NextServeOrNil(g.next, keycloakClient)
}

func (g *GetOrCreateRealmOwner) addTargetRealmIfNeed(keycloakClient *v1v1alpha1.KeycloakClient,
	reamName string) error {
	if keycloakClient.Spec.TargetRealm != "" {
		return nil
	}

	keycloakClient.Spec.TargetRealm = reamName
	if err := g.Client.Update(context.TODO(), keycloakClient); err != nil {
		return errors.Wrap(err, "unable to set keycloak client target realm")
	}

	return nil
}
