package adapter

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

var goCloakClientSupplier = func(url string) GoCloak {
	return gocloak.NewClient(url)
}

type GoCloakAdapterFactory struct {
	Log logr.Logger
}

func (a GoCloakAdapterFactory) New(keycloak dto.Keycloak) (keycloak.Client, error) {
	log := a.Log.WithValues("keycloak dto", keycloak)
	log.Info("Start creation new Keycloak Client...")

	client := goCloakClientSupplier(keycloak.Url)
	token, err := client.LoginAdmin(context.Background(), keycloak.User, keycloak.Pwd, "master")
	if err != nil {
		errMsg := fmt.Sprintf("cannot login to Keycloak server by Keycloak dto: %+v", keycloak)
		return nil, errors.Wrap(err, errMsg)
	}

	log.Info("Connection has been successfully established")
	return GoCloakAdapter{
		client:   client,
		token:    *token,
		basePath: keycloak.Url,
		log:      ctrl.Log.WithName("go-cloak-adapter"),
	}, nil
}
