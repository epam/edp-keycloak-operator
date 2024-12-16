package chain

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MakeChain(c client.Client, operatorNs string) RealmHandler {
	ch := &chain{}
	ch.Use(
		NewPutRealm(c),
		NewPutRealmSettings(),
		NewUserProfile(),
		NewConfigureEmail(c, operatorNs),
	)

	return ch
}
