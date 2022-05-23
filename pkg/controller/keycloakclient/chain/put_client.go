package chain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1v1alpha1 "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

const clientSecretKey = "clientSecret"

type PutClient struct {
	BaseElement
	next Element
}

func (el *PutClient) Serve(ctx context.Context, keycloakClient *v1v1alpha1.KeycloakClient, adapterClient keycloak.Client) error {
	id, err := el.putKeycloakClient(keycloakClient, adapterClient)
	if err != nil {
		return errors.Wrap(err, "unable to put keycloak client")
	}
	keycloakClient.Status.ClientID = id

	return el.NextServeOrNil(ctx, el.next, keycloakClient, adapterClient)
}

func (el *PutClient) putKeycloakClient(keycloakClient *v1v1alpha1.KeycloakClient, adapterClient keycloak.Client) (string, error) {
	reqLog := el.Logger.WithValues("keycloak client cr", keycloakClient)
	reqLog.Info("Start put keycloak client...")

	clientDto, err := el.convertCrToDto(keycloakClient)
	if err != nil {
		return "", errors.Wrap(err, "error during convertCrToDto")
	}

	exist, err := adapterClient.ExistClient(clientDto.ClientId, clientDto.RealmName)
	if err != nil {
		return "", errors.Wrap(err, "error during ExistClient")
	}

	if exist {
		reqLog.Info("Client already exists")
		return adapterClient.GetClientID(clientDto.ClientId, clientDto.RealmName)
	}

	err = adapterClient.CreateClient(clientDto)
	if err != nil {
		return "", errors.Wrap(err, "error during CreateClient")
	}

	reqLog.Info("End put keycloak client")
	id, err := adapterClient.GetClientID(clientDto.ClientId, clientDto.RealmName)
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
	var clientSecret coreV1.Secret
	secretName := fmt.Sprintf("keycloak-client-%s-secret", keycloakClient.Name)
	//TODO: get context from controller
	err := el.Client.Get(context.Background(), types.NamespacedName{Namespace: keycloakClient.Namespace,
		Name: secretName}, &clientSecret)
	if err != nil && !k8sErrors.IsNotFound(err) {
		return "", errors.Wrap(err, "unable to check client secret existance")
	}

	if k8sErrors.IsNotFound(err) {
		clientSecret = coreV1.Secret{
			ObjectMeta: v1.ObjectMeta{Namespace: keycloakClient.Namespace,
				Name: secretName},
			Data: map[string][]byte{clientSecretKey: []byte(password.MustGenerate(36, 9, 0,
				true, true))},
		}

		if err := controllerutil.SetControllerReference(keycloakClient, &clientSecret, el.scheme); err != nil {
			return "", errors.Wrap(err, "unable to set controller ref for secret")
		}
		//TODO: get context from controller
		if err := el.Client.Create(context.Background(), &clientSecret); err != nil {
			return "", errors.Wrapf(err, "unable to create secret %+v", clientSecret)
		}
	}

	keycloakClient.Status.ClientSecretName = clientSecret.Name
	keycloakClient.Spec.Secret = clientSecret.Name
	//TODO: get context from controller
	if err := el.Client.Update(context.Background(), keycloakClient); err != nil {
		return "", errors.Wrapf(err, "unable to update client with new secret: %s", clientSecret.Name)
	}

	return string(clientSecret.Data["clientSecret"]), nil
}
