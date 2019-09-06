package adapter

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/nerzal/gocloak.v2"
	"keycloak-operator/pkg/apis/v1/v1alpha1"
	"keycloak-operator/pkg/client/keycloak"
)

var goCloakClientSupplier = func(url string) gocloak.GoCloak {
	return gocloak.NewClient(url)
}

type GoCloakAdapterFactory struct {
}

func (GoCloakAdapterFactory) New(spec v1alpha1.KeycloakSpec) (keycloak.Client, error) {
	reqLog := log.WithValues("keycloak spec", spec)
	reqLog.Info("Start creation new Keycloak Client...")

	client := goCloakClientSupplier(spec.Url)
	token, err := client.LoginAdmin(spec.User, spec.Pwd, "master")
	if err != nil {
		errMsg := fmt.Sprintf("cannot login to Keycloak server by Keycloak spec: %+v", spec)
		return nil, errors.Wrap(err, errMsg)
	}

	reqLog.Info("Connection has been successfully established")
	return GoCloakAdapter{
		client:   client,
		token:    *token,
		basePath: spec.Url,
	}, nil
}
