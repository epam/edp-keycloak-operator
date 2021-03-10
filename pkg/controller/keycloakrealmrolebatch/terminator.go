package keycloakrealmrolebatch

import (
	"context"

	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type terminator struct {
	client     client.Client
	childRoles []v1alpha1.KeycloakRealmRole
}

func (t *terminator) DeleteResource() error {
	for _, r := range t.childRoles {
		if err := t.client.Delete(context.Background(), &r); err != nil {
			return errors.Wrap(err, "unable to delete child role")
		}
	}

	return nil
}

func makeTerminator(client client.Client, childRoles []v1alpha1.KeycloakRealmRole) *terminator {
	return &terminator{
		client:     client,
		childRoles: childRoles,
	}
}
