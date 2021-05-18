package chain

import (
	v1v1alpha1 "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Element interface {
	Serve(keycloakClient *v1v1alpha1.KeycloakClient, adapterClient keycloak.Client) error
}

type BaseElement struct {
	Client client.Client
	Logger logr.Logger
	scheme *runtime.Scheme
}

func (b *BaseElement) NextServeOrNil(next Element, keycloakClient *v1v1alpha1.KeycloakClient,
	adapterClient keycloak.Client) error {
	if next != nil {
		return next.Serve(keycloakClient, adapterClient)
	}

	return nil
}
