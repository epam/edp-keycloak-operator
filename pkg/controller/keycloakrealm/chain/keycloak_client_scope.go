package chain

import (
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/consts"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/model"
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
	if err := kClient.CreateClientScope(realm.Spec.RealmName, getDefClientScope()); err != nil {
		return err
	}
	rLog.Info("End of put Keycloak Scope")
	return nextServeOrNil(h.next, realm, kClient)
}

func getDefClientScope() model.ClientScope {
	return model.ClientScope{
		Name:        stringP(consts.DefaultClientScopeName),
		Description: stringP("default edp scope required for ac and nexus"),
		Protocol:    stringP(consts.OpenIdProtocol),
		ClientScopeAttributes: &model.ClientScopeAttributes{
			IncludeInTokenScope: stringP("true"),
		},
	}
}
func stringP(value string) *string {
	return &value
}
