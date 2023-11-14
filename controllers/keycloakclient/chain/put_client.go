package chain

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-password/password"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

const (
	passwordLength  = 36
	passwordDigits  = 9
	passwordSymbols = 0
)

// secretRef is an interface for getting secret from ref.
type secretRef interface {
	GetSecretFromRef(ctx context.Context, refVal, secretNamespace string) (string, error)
}

type PutClient struct {
	BaseElement
	SecretRef secretRef
	next      Element
}

func (el *PutClient) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client, realmName string) error {
	id, err := el.putKeycloakClient(ctx, keycloakClient, adapterClient, realmName)

	if err != nil {
		return fmt.Errorf("unable to put keycloak client: %w", err)
	}

	keycloakClient.Status.ClientID = id

	return el.NextServeOrNil(ctx, el.next, keycloakClient, adapterClient, realmName)
}

func (el *PutClient) putKeycloakClient(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client, realmName string) (string, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start creation of Keycloak client")

	clientDto, err := el.convertCrToDto(ctx, keycloakClient, realmName)
	if err != nil {
		return "", fmt.Errorf("error during convertCrToDto: %w", err)
	}

	clientID, err := adapterClient.GetClientID(clientDto.ClientId, clientDto.RealmName)
	if err != nil && !adapter.IsErrNotFound(err) {
		return "", fmt.Errorf("unable to check client id: %w", err)
	}

	if clientID != "" {
		log.Info("Client already exists")

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

	log.Info("End put keycloak client")

	id, err := adapterClient.GetClientID(clientDto.ClientId, clientDto.RealmName)
	if err != nil {
		return "", fmt.Errorf("unable to check client id: %w", err)
	}

	return id, nil
}

func (el *PutClient) convertCrToDto(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) (*dto.Client, error) {
	if keycloakClient.Spec.Public {
		res := dto.ConvertSpecToClient(&keycloakClient.Spec, "", realmName)
		return res, nil
	}

	secret, err := el.getSecret(ctx, keycloakClient)
	if err != nil {
		return nil, fmt.Errorf("unable to get secret, err: %w", err)
	}

	return dto.ConvertSpecToClient(&keycloakClient.Spec, secret, realmName), nil
}

func (el *PutClient) getSecret(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) (string, error) {
	if keycloakClient.Spec.Secret != "" {
		// We need to set secret in a new format for old clients for backward compatibility.
		// TODO: This code can be removed in the future.
		if !secretref.HasSecretRef(keycloakClient.Spec.Secret) {
			if err := el.setSecretRef(ctx, keycloakClient); err != nil {
				return "", err
			}
		}

		secretVal, err := el.SecretRef.GetSecretFromRef(ctx, keycloakClient.Spec.Secret, keycloakClient.Namespace)
		if err != nil {
			return "", fmt.Errorf("unable to get secret from ref: %w", err)
		}

		return secretVal, nil
	}

	return el.generateSecret(ctx, keycloakClient)
}

func (el *PutClient) generateSecret(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) (string, error) {
	var clientSecret corev1.Secret

	secretName := fmt.Sprintf("keycloak-client-%s-secret", keycloakClient.Name)

	secretErr := el.Client.Get(ctx, types.NamespacedName{Namespace: keycloakClient.Namespace,
		Name: secretName}, &clientSecret)
	if secretErr != nil && !k8sErrors.IsNotFound(secretErr) {
		return "", fmt.Errorf("unable to check client secret existance: %w", secretErr)
	}

	pass, err := password.Generate(passwordLength, passwordDigits, passwordSymbols, true, true)
	if err != nil {
		return "", fmt.Errorf("unable to generate password: %w", err)
	}

	if k8sErrors.IsNotFound(secretErr) {
		clientSecret = corev1.Secret{
			ObjectMeta: v1.ObjectMeta{Namespace: keycloakClient.Namespace,
				Name: secretName},
			Data: map[string][]byte{
				keycloakApi.ClientSecretKey: []byte(pass),
			},
		}

		if err := controllerutil.SetControllerReference(keycloakClient, &clientSecret, el.scheme); err != nil {
			return "", fmt.Errorf("unable to set controller ref for secret: %w", err)
		}

		if err := el.Client.Create(ctx, &clientSecret); err != nil {
			return "", fmt.Errorf("unable to create secret %+v, err: %w", clientSecret, err)
		}
	}

	keycloakClient.Spec.Secret = secretref.GenerateSecretRef(clientSecret.Name, keycloakApi.ClientSecretKey)

	if err := el.Client.Update(ctx, keycloakClient); err != nil {
		return "", fmt.Errorf("unable to update client with new secret: %s, err: %w", clientSecret.Name, err)
	}

	return string(clientSecret.Data[keycloakApi.ClientSecretKey]), nil
}

func (el *PutClient) setSecretRef(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) error {
	ref := secretref.GenerateSecretRef(keycloakClient.Spec.Secret, keycloakApi.ClientSecretKey)
	keycloakClient.Spec.Secret = ref

	if err := el.Client.Update(ctx, keycloakClient); err != nil {
		return fmt.Errorf("unable to update client with secret ref %s: %w", ref, err)
	}

	return nil
}
