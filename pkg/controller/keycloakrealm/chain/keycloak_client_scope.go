package chain

import (
	"context"

	"github.com/pkg/errors"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/consts"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PutKeycloakClientScope struct {
	next   handler.RealmHandler
	client client.Client
	scheme *runtime.Scheme
}

func (h PutKeycloakClientScope) ServeRequest(ctx context.Context, realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start Keycloak Scope")
	defaultScope := getDefClientScope()

	_, err := kClient.GetClientScope(defaultScope.Name, realm.Spec.RealmName)
	if err != nil && !adapter.IsErrNotFound(err) {
		return errors.Wrap(err, "unable to get client scope")
	}

	if adapter.IsErrNotFound(err) {
		if _, err := kClient.CreateClientScope(ctx, realm.Spec.RealmName, getDefClientScope()); err != nil {
			return err
		}
	}

	rLog.Info("End of put Keycloak Scope")
	return nextServeOrNil(ctx, h.next, realm, kClient)
}

func getDefClientScope() *adapter.ClientScope {
	return &adapter.ClientScope{
		Name:        consts.DefaultClientScopeName,
		Description: "default edp scope required for ac and nexus",
		Protocol:    consts.OpenIdProtocol,
		Attributes: map[string]string{
			"include.in.token.scope": "true",
		},
	}
}
func stringP(value string) *string {
	return &value
}
