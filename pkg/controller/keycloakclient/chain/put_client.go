package chain

import (
	"context"
	"fmt"

	v1v1alpha1 "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const clientSecretKey = "clientSecret"

type PutClient struct {
	BaseElement
	next Element
}

func (el *PutClient) Serve(keycloakClient *v1v1alpha1.KeycloakClient) error {
	id, err := el.putKeycloakClient(keycloakClient)
	if err != nil {
		return errors.Wrap(err, "unable to put keycloak client")
	}
	keycloakClient.Status.ClientID = id

	return el.NextServeOrNil(el.next, keycloakClient)
}

func (el *PutClient) putKeycloakClient(keycloakClient *v1v1alpha1.KeycloakClient) (string, error) {
	reqLog := el.Logger.WithValues("keycloak client cr", keycloakClient)
	reqLog.Info("Start put keycloak client...")

	clientDto, err := el.convertCrToDto(keycloakClient)
	if err != nil {
		return "", errors.Wrap(err, "error during convertCrToDto")
	}

	exist, err := el.State.AdapterClient.ExistClient(clientDto.ClientId, clientDto.RealmName)
	if err != nil {
		return "", errors.Wrap(err, "error during ExistClient")
	}

	if exist {
		reqLog.Info("Client already exists")
		return el.State.AdapterClient.GetClientID(clientDto.ClientId, clientDto.RealmName)
	}

	err = el.State.AdapterClient.CreateClient(clientDto)
	if err != nil {
		return "", errors.Wrap(err, "error during CreateClient")
	}

	reqLog.Info("End put keycloak client")
	id, err := el.State.AdapterClient.GetClientID(clientDto.ClientId, clientDto.RealmName)
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

	if keycloakClient.Spec.Secret != "" {
		secret, err := el.getSecret(keycloakClient)
		if err != nil {
			return nil, errors.Wrap(err, "unable to get secret")
		}

		return dto.ConvertSpecToClient(&keycloakClient.Spec, secret), nil
	}

	secret, err := el.generateSecret(keycloakClient)
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate secret")
	}

	return dto.ConvertSpecToClient(&keycloakClient.Spec, secret), nil
}

func (el *PutClient) getSecret(keycloakClient *v1v1alpha1.KeycloakClient) (string, error) {
	var clientSecret coreV1.Secret

	if err := el.Client.Get(context.TODO(), types.NamespacedName{
		Name:      keycloakClient.Spec.Secret,
		Namespace: keycloakClient.Namespace,
	}, &clientSecret); err != nil {
		return "", errors.Wrapf(err, "unable to get client secret, secret name: %s",
			keycloakClient.Spec.Secret)
	}

	return string(clientSecret.Data["clientSecret"]), nil
}

func (el *PutClient) generateSecret(keycloakClient *v1v1alpha1.KeycloakClient) (string, error) {
	clientSecret := coreV1.Secret{
		ObjectMeta: v1.ObjectMeta{Namespace: keycloakClient.Namespace,
			Name: fmt.Sprintf("keycloak-client-%s-secret", keycloakClient.Name)},
		Data: map[string][]byte{clientSecretKey: []byte(password.MustGenerate(32, 7, 4,
			false, true))},
	}

	if err := controllerutil.SetControllerReference(keycloakClient, &clientSecret, el.Helper.GetScheme()); err != nil {
		return "", errors.Wrap(err, "unable to set controller ref for secret")
	}

	if err := el.Client.Create(context.Background(), &clientSecret); err != nil {
		return "", errors.Wrapf(err, "unable to create secret %+v", clientSecret)
	}

	keycloakClient.Status.ClientSecretName = clientSecret.Name
	keycloakClient.Spec.Secret = clientSecret.Name

	if err := el.Client.Update(context.Background(), keycloakClient); err != nil {
		return "", errors.Wrapf(err, "unable to update client with new secret: %s", clientSecret.Name)
	}

	return string(clientSecret.Data["clientSecret"]), nil
}
