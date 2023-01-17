package keycloakrealmrolebatch

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

type terminator struct {
	client     client.Client
	childRoles []keycloakApi.KeycloakRealmRole
	log        logr.Logger
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	t.log.Info("start deleting keycloak realm role batch")

	for i := range t.childRoles {
		if err := t.client.Delete(ctx, &t.childRoles[i]); err != nil {
			return errors.Wrap(err, "unable to delete child role")
		}
	}

	t.log.Info("done deleting keycloak realm role batch")

	return nil
}

func (t *terminator) GetLogger() logr.Logger {
	return t.log
}

func makeTerminator(client client.Client, childRoles []keycloakApi.KeycloakRealmRole, log logr.Logger) *terminator {
	return &terminator{
		client:     client,
		childRoles: childRoles,
		log:        log,
	}
}
