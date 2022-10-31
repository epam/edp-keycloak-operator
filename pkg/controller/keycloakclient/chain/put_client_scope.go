package chain

import (
	"context"
	"fmt"

	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"

	v1v1alpha1 "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/pkg/errors"
)

type PutClientScope struct {
	BaseElement
	next Element
}

func (el *PutClientScope) Serve(ctx context.Context, keycloakClient *v1v1alpha1.KeycloakClient, adapterClient keycloak.Client) error {
	if err := el.putClientScope(ctx, keycloakClient, adapterClient); err != nil {
		return errors.Wrap(err, "error during putClientScope")
	}

	return el.NextServeOrNil(ctx, el.next, keycloakClient, adapterClient)
}

func (el *PutClientScope) putClientScope(ctx context.Context, keycloakClient *v1v1alpha1.KeycloakClient, adapterClient keycloak.Client) error {
	if !keycloakClient.Spec.AudRequired {
		return nil
	}

	scope, err := adapterClient.GetClientScope(helper.DefaultClientScopeName, keycloakClient.Spec.TargetRealm)
	if err != nil {
		return errors.Wrap(err, "error during GetClientScope")
	}

	if err := el.putClientScopeMapper(ctx, keycloakClient, adapterClient, scope); err != nil {
		return errors.Wrap(err, "unable to put scope mapper")
	}

	if err := adapterClient.LinkClientScopeToClient(
		keycloakClient.Spec.ClientId, scope.ID, keycloakClient.Spec.TargetRealm); err != nil {
		return errors.Wrap(err, "error during LinkClientScopeToClient")
	}

	return nil
}

func (el *PutClientScope) putClientScopeMapper(ctx context.Context, keycloakClient *v1v1alpha1.KeycloakClient, adapterClient keycloak.Client,
	scope *adapter.ClientScope) error {
	mappers, err := adapterClient.GetClientScopeMappers(ctx, keycloakClient.Spec.TargetRealm, scope.ID)
	if err != nil {
		return errors.Wrap(err, "unable to get client scope mappers")
	}

	audienceProtocolMapper := getAudienceProtocolMapper(keycloakClient.Spec.ClientId)
	for _, pm := range mappers {
		if pm.Name == audienceProtocolMapper.Name {
			return nil
		}
	}

	if err := adapterClient.PutClientScopeMapper(
		keycloakClient.Spec.TargetRealm, scope.ID, audienceProtocolMapper); err != nil {
		return errors.Wrap(err, "error during PutClientScopeMapper")
	}

	return nil
}

func getAudienceProtocolMapper(clientId string) *adapter.ProtocolMapper {
	return &adapter.ProtocolMapper{
		Name:           fmt.Sprintf("%v-%v", clientId, "audience"),
		Protocol:       adapter.OpenIdProtocol,
		ProtocolMapper: adapter.OIDCAudienceMapper,
		Config: map[string]string{
			"access.token.claim":       "true",
			"included.client.audience": clientId,
		},
	}
}
