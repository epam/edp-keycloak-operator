package keycloak

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/nerzal/gocloak.v2"
	v1v1alpha1 "keycloak-operator/pkg/apis/v1/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("gocloak_adapter")

// GoCloakConnection is the abstraction that holds both Gocloak client and Gocloak token
type GoCloakConnection struct {
	client gocloak.GoCloak
	token  gocloak.JWT
}

type IGoCloakAdapter interface {
	GetConnection(cr v1v1alpha1.Keycloak) (*GoCloakConnection, error)
}

// GoCloakAdapter it the struct that holds ClientSup - supplier that returns the implementation of gocloak client
type GoCloakAdapter struct {
	ClientSup func(url string) gocloak.GoCloak
}

// GetConnection is the method of the adapter that establish connection to the Keycloak server according to the
// input Keycloak CR and follows the next rules:
// status of CR should be connected
// login according to the spec of the CR
// uses supplier from the GoCloakAdapter struct
func (a GoCloakAdapter) GetConnection(cr v1v1alpha1.Keycloak) (*GoCloakConnection, error) {
	reqLog := log.WithValues("Keycloak Cr spec", cr.Spec)
	reqLog.Info("Start getting connection...")
	if a.ClientSup == nil {
		return nil, errors.New("gocloak adapter does not have required client supplier")
	}
	client := a.ClientSup(cr.Spec.Url)
	token, err := client.LoginAdmin(cr.Spec.User, cr.Spec.Pwd, "master")
	if err != nil {
		errMsg := fmt.Sprintf("cannot login to Keycloak server by cr: %+v", cr)
		return nil, errors.Wrap(err, errMsg)
	}
	reqLog.Info("Connection has been established", "token", token)
	return &GoCloakConnection{
		client: client,
		token:  *token,
	}, nil
}
