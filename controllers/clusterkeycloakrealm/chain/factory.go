package chain

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MakeChain(c client.Client) RealmHandler {
	ch := &chain{}
	ch.Use(
		NewPutRealm(c),
		NewPutRealmSettings(),
		NewUserProfile(),
	)

	return ch
}
