package chain

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-password/password"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

const (
	passwordLength  = 36
	passwordDigits  = 9
	passwordSymbols = 0
)

type PutClient struct {
	BaseElement
	next Element
}

func (el *PutClient) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client) error {
	id, err := el.putKeycloakClient(ctx, keycloakClient, adapterClient)

	if err != nil {
		return fmt.Errorf("unable to put keycloak client: %w", err)
	}

	keycloakClient.Status.ClientID = id

	return el.NextServeOrNil(ctx, el.next, keycloakClient, adapterClient)
}

func (el *PutClient) putKeycloakClient(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client) (string, error) {
	reqLog := el.Logger.WithValues("keycloak client cr", keycloakClient)
	reqLog.Info("Start put keycloak client...")

	clientDto, err := el.convertCrToDto(ctx, keycloakClient)
	if err != nil {
		return "", fmt.Errorf("error during convertCrToDto: %w", err)
	}

	clientID, err := adapterClient.GetClientID(clientDto.ClientId, clientDto.RealmName)
	if err != nil && !adapter.IsErrNotFound(err) {
		return "", fmt.Errorf("unable to check client id: %w", err)
	}

	if clientID != "" {
		reqLog.Info("Client already exists")

		clientDto.ID = clientID
		if updErr := adapterClient.UpdateClient(ctx, clientDto); updErr != nil {
			return "", fmt.Errorf("unable to update keycloak client: %w", updErr)
		}

		return clientID, nil
	}

	err = adapterClient.CreateClient(ctx, clientDto)
	if err != nil {
		return "", fmt.Errorf("unable to create client: %w", err)
	}

	reqLog.Info("End put keycloak client")

	id, err := adapterClient.GetClientID(clientDto.ClientId, clientDto.RealmName)
	if err != nil {
		return "", fmt.Errorf("unable to check client id: %w", err)
	}

	return id, nil
}

func (el *PutClient) convertCrToDto(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) (*dto.Client, error) {
	if keycloakClient.Spec.Public {
		res := dto.ConvertSpecToClient(&keycloakClient.Spec, "")
		return res, nil
	}

	if keycloakClient.Spec.Secret != "" {
		secret, err := el.getSecret(ctx, keycloakClient)
		if err != nil {
			return nil, fmt.Errorf("unable to get secret, err: %w", err)
		}

		return dto.ConvertSpecToClient(&keycloakClient.Spec, secret), nil
	}

	secret, err := el.generateSecret(ctx, keycloakClient)
	if err != nil {
		return nil, fmt.Errorf("unable to generate secret: %w", err)
	}

	return dto.ConvertSpecToClient(&keycloakClient.Spec, secret), nil
}

func (el *PutClient) getSecret(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) (string, error) {
	var clientSecret coreV1.Secret

	if err := el.Client.Get(ctx, types.NamespacedName{
		Name:      keycloakClient.Spec.Secret,
		Namespace: keycloakClient.Namespace,
	}, &clientSecret); err != nil {
		return "", fmt.Errorf("unable to get client secret, secret name: %s, err: %w",
			keycloakClient.Spec.Secret, err)
	}

	return string(clientSecret.Data[keycloakApi.ClientSecretKey]), nil
}

func (el *PutClient) generateSecret(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) (string, error) {
	var clientSecret coreV1.Secret

	secretName := fmt.Sprintf("keycloak-client-%s-secret", keycloakClient.Name)

	err := el.Client.Get(ctx, types.NamespacedName{Namespace: keycloakClient.Namespace,
		Name: secretName}, &clientSecret)
	if err != nil && !k8sErrors.IsNotFound(err) {
		return "", fmt.Errorf("unable to check client secret existance: %w", err)
	}

	if k8sErrors.IsNotFound(err) {
		clientSecret = coreV1.Secret{
			ObjectMeta: v1.ObjectMeta{Namespace: keycloakClient.Namespace,
				Name: secretName},
			Data: map[string][]byte{
				keycloakApi.ClientSecretKey: []byte(
					password.MustGenerate(passwordLength, passwordDigits, passwordSymbols, true, true),
				),
			},
		}

		if err := controllerutil.SetControllerReference(keycloakClient, &clientSecret, el.scheme); err != nil {
			return "", fmt.Errorf("unable to set controller ref for secret: %w", err)
		}

		if err := el.Client.Create(ctx, &clientSecret); err != nil {
			return "", fmt.Errorf("unable to create secret %+v, err: %w", clientSecret, err)
		}
	}

	keycloakClient.Status.ClientSecretName = clientSecret.Name
	keycloakClient.Spec.Secret = clientSecret.Name

	if err := el.Client.Update(ctx, keycloakClient); err != nil {
		return "", fmt.Errorf("unable to update client with new secret: %s, err: %w", clientSecret.Name, err)
	}

	return string(clientSecret.Data[keycloakApi.ClientSecretKey]), nil
}
