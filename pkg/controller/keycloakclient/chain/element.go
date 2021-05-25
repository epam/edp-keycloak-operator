package chain

import (
	v1v1alpha1 "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Element interface {
	Serve(keycloakClient *v1v1alpha1.KeycloakClient) error
	GetState() *State
}

type BaseElement struct {
	State  *State
	Helper *helper.Helper
	Client client.Client
	Logger logr.Logger
}

func (b *BaseElement) NextServeOrNil(next Element, keycloakClient *v1v1alpha1.KeycloakClient) error {
	if next != nil {
		return next.Serve(keycloakClient)
	}

	return nil
}

func (b *BaseElement) GetState() *State {
	return b.State
}

type State struct {
	KeycloakRealm *v1v1alpha1.KeycloakRealm
	AdapterClient keycloak.Client
}
