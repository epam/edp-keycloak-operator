package keycloakrealmgroup

import (
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/pkg/errors"
)

type terminator struct {
	kClient              keycloak.Client
	realmName, groupName string
}

func (t *terminator) DeleteResource() error {
	if err := t.kClient.DeleteGroup(t.realmName, t.groupName); err != nil {
		return errors.Wrapf(err, "unable to delete group, realm: %s, group: %s", t.realmName, t.groupName)
	}

	return nil
}

func makeTerminator(kClient keycloak.Client, realmName, groupName string) *terminator {
	return &terminator{
		kClient:   kClient,
		realmName: realmName,
		groupName: groupName,
	}
}
