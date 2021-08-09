package chain

import (
	"context"

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

func (h PutKeycloakClientScope) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start Keycloak Scope")
	if _, err := kClient.CreateClientScope(context.Background(), realm.Spec.RealmName, getDefClientScope()); err != nil {
		return err
	}
	rLog.Info("End of put Keycloak Scope")
	return nextServeOrNil(h.next, realm, kClient)
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
