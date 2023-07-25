package keycloakauthflow

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

type terminator struct {
	realmName        string
	realmCRName      string
	kClient          keycloak.Client
	k8sClient        client.Client
	keycloakAuthFlow *adapter.KeycloakAuthFlow
}

func makeTerminator(
	realmName string,
	realmCRName string,
	authFlow *adapter.KeycloakAuthFlow,
	k8sClient client.Client,
	kClient keycloak.Client,
) *terminator {
	return &terminator{
		realmName:        realmName,
		realmCRName:      realmCRName,
		keycloakAuthFlow: authFlow,
		kClient:          kClient,
		k8sClient:        k8sClient,
	}
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("realm name", t.realmName, "flow alias", t.keycloakAuthFlow.Alias)

	var authFlowList keycloakApi.KeycloakAuthFlowList
	if err := t.k8sClient.List(ctx, &authFlowList); err != nil {
		return fmt.Errorf("unable to get auth flow list: %w", err)
	}

	for i := range authFlowList.Items {
		if authFlowList.Items[i].Spec.RealmRef.Name == t.realmCRName && authFlowList.Items[i].Spec.ParentName == t.keycloakAuthFlow.Alias {
			return errors.Errorf("unable to delete flow: %s while it has child: %s", t.keycloakAuthFlow.Alias,
				authFlowList.Items[i].Spec.Alias)
		}
	}

	log.Info("Start deleting auth flow")

	if err := t.kClient.DeleteAuthFlow(t.realmName, t.keycloakAuthFlow); err != nil {
		return fmt.Errorf("unable to delete auth flow: %w", err)
	}

	log.Info("Deleting auth flow done")

	return nil
}
