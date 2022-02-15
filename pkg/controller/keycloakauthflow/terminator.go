package keycloakauthflow

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

type terminator struct {
	realmName        string
	kClient          keycloak.Client
	log              logr.Logger
	k8sClient        client.Client
	keycloakAuthFlow *adapter.KeycloakAuthFlow
}

func makeTerminator(realmName string, authFlow *adapter.KeycloakAuthFlow, k8sClient client.Client,
	kClient keycloak.Client, log logr.Logger) *terminator {

	return &terminator{
		realmName:        realmName,
		keycloakAuthFlow: authFlow,
		kClient:          kClient,
		log:              log,
		k8sClient:        k8sClient,
	}
}

func (t *terminator) GetLogger() logr.Logger {
	return t.log
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	logger := t.log.WithValues("realm name", t.realmName, "flow alias", t.keycloakAuthFlow.Alias)

	var authFlowList v1alpha1.KeycloakAuthFlowList
	if err := t.k8sClient.List(ctx, &authFlowList); err != nil {
		return errors.Wrap(err, "unable to list auth flows")
	}

	for _, af := range authFlowList.Items {
		if af.Spec.ParentName == t.keycloakAuthFlow.Alias {
			return errors.Errorf("Unable to delete flow: %s while it has child: %s", t.keycloakAuthFlow.Alias,
				af.Spec.Alias)
		}
	}

	logger.Info("start deleting auth flow")
	if err := t.kClient.DeleteAuthFlow(t.realmName, t.keycloakAuthFlow); err != nil {
		return errors.Wrap(err, "unable to delete auth flow")
	}

	logger.Info("deleting auth flow done")
	return nil
}
