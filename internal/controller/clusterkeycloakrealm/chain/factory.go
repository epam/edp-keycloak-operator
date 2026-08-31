package chain

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/pkg/destination"
)

func MakeChain(c client.Client, operatorNs string, guard *destination.Guard) RealmHandler {
	ch := &chain{}
	ch.Use(
		NewPutRealm(c),
		NewPutRealmSettings(),
		NewPutRealmLocalizationTexts(),
		NewUserProfile(),
		NewConfigureEmail(c, operatorNs, guard),
		NewAuthFlow(),
	)

	return ch
}
