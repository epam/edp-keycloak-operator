package chain

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type Element interface {
	Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client) error
}

type BaseElement struct {
	Client client.Client
	Logger logr.Logger
	scheme *runtime.Scheme
}

func (b *BaseElement) NextServeOrNil(
	ctx context.Context,
	next Element,
	keycloakClient *keycloakApi.KeycloakClient,
	adapterClient keycloak.Client,
) error {
	if next != nil {
		return next.Serve(ctx, keycloakClient, adapterClient)
	}

	return nil
}
