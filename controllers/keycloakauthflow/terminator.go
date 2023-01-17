package keycloakauthflow

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

type terminator struct {
	realm            *keycloakApi.KeycloakRealm
	kClient          keycloak.Client
	log              logr.Logger
	k8sClient        client.Client
	keycloakAuthFlow *adapter.KeycloakAuthFlow
}

func makeTerminator(
	realm *keycloakApi.KeycloakRealm,
	authFlow *adapter.KeycloakAuthFlow,
	k8sClient client.Client,
	kClient keycloak.Client,
	log logr.Logger,
) *terminator {
	return &terminator{
		realm:            realm,
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
	logger := t.log.WithValues("realm name", t.realm.Spec.RealmName, "flow alias", t.keycloakAuthFlow.Alias)

	var authFlowList keycloakApi.KeycloakAuthFlowList
	if err := t.k8sClient.List(ctx, &authFlowList); err != nil {
		return errors.Wrap(err, "unable to list auth flows")
	}

	for i := range authFlowList.Items {
		if authFlowList.Items[i].Spec.Realm == t.realm.Name && authFlowList.Items[i].Spec.ParentName == t.keycloakAuthFlow.Alias {
			return errors.Errorf("Unable to delete flow: %s while it has child: %s", t.keycloakAuthFlow.Alias,
				authFlowList.Items[i].Spec.Alias)
		}
	}

	logger.Info("start deleting auth flow")

	if err := t.kClient.DeleteAuthFlow(t.realm.Spec.RealmName, t.keycloakAuthFlow); err != nil {
		return errors.Wrap(err, "unable to delete auth flow")
	}

	logger.Info("deleting auth flow done")

	return nil
}
