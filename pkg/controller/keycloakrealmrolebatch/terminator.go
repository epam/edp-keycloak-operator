package keycloakrealmrolebatch

import (
	"context"

	"github.com/go-logr/logr"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type terminator struct {
	client     client.Client
	childRoles []v1alpha1.KeycloakRealmRole
	log        logr.Logger
}

func (t *terminator) DeleteResource() error {
	t.log.Info("start deleting keycloak realm role batch")
	for _, r := range t.childRoles {
		if err := t.client.Delete(context.Background(), &r); err != nil {
			return errors.Wrap(err, "unable to delete child role")
		}
	}

	t.log.Info("done deleting keycloak realm role batch")
	return nil
}

func (t *terminator) GetLogger() logr.Logger {
	return t.log
}

func makeTerminator(client client.Client, childRoles []v1alpha1.KeycloakRealmRole, log logr.Logger) *terminator {
	return &terminator{
		client:     client,
		childRoles: childRoles,
		log:        log,
	}
}
