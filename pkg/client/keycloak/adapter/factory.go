package adapter

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/pkg/errors"
)

var goCloakClientSupplier = func(url string) GoCloak {
	return gocloak.NewClient(url)
}

type GoCloakAdapterFactory struct {
}

func (GoCloakAdapterFactory) New(keycloak dto.Keycloak) (keycloak.Client, error) {
	reqLog := log.WithValues("keycloak dto", keycloak)
	reqLog.Info("Start creation new Keycloak Client...")

	client := goCloakClientSupplier(keycloak.Url)
	token, err := client.LoginAdmin(context.Background(), keycloak.User, keycloak.Pwd, "master")
	if err != nil {
		errMsg := fmt.Sprintf("cannot login to Keycloak server by Keycloak dto: %+v", keycloak)
		return nil, errors.Wrap(err, errMsg)
	}

	reqLog.Info("Connection has been successfully established")
	return GoCloakAdapter{
		client:   client,
		token:    *token,
		basePath: keycloak.Url,
	}, nil
}
