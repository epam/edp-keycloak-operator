package chain

import (
	"context"

	v1v1alpha1 "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type PutClient struct {
	BaseElement
	next Element
}

func (el *PutClient) Serve(keycloakClient *v1v1alpha1.KeycloakClient) error {
	id, err := el.putKeycloakClient(keycloakClient)
	if err != nil {
		return errors.Wrap(err, "unable to put keycloak client")
	}
	keycloakClient.Status.Id = id

	return el.NextServeOrNil(el.next, keycloakClient)
}

func (el *PutClient) putKeycloakClient(keycloakClient *v1v1alpha1.KeycloakClient) (string, error) {
	reqLog := el.Logger.WithValues("keycloak client cr", keycloakClient)
	reqLog.Info("Start put keycloak client...")

	clientDto, err := el.convertCrToDto(keycloakClient)
	if err != nil {
		return "", errors.Wrap(err, "error during convertCrToDto")
	}

	exist, err := el.State.AdapterClient.ExistClient(clientDto)
	if err != nil {
		return "", errors.Wrap(err, "error during ExistClient")
	}

	if exist {
		reqLog.Info("Client already exists")
		return el.State.AdapterClient.GetClientID(clientDto)
	}

	err = el.State.AdapterClient.CreateClient(clientDto)
	if err != nil {
		return "", errors.Wrap(err, "error during CreateClient")
	}

	reqLog.Info("End put keycloak client")
	id, err := el.State.AdapterClient.GetClientID(clientDto)
	if err != nil {
		return "", errors.Wrap(err, "unable to get client id")
	}

	return id, nil
}

func (el *PutClient) convertCrToDto(keycloakClient *v1v1alpha1.KeycloakClient) (*dto.Client, error) {
	if keycloakClient.Spec.Public {
		res := dto.ConvertSpecToClient(&keycloakClient.Spec, "")
		return res, nil
	}

	var clientSecret coreV1.Secret
	err := el.Client.Get(context.TODO(), types.NamespacedName{
		Name:      keycloakClient.Spec.Secret,
		Namespace: keycloakClient.Namespace,
	}, &clientSecret)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get client secret")
	}

	return dto.ConvertSpecToClient(&keycloakClient.Spec, string(clientSecret.Data["clientSecret"])), nil
}
