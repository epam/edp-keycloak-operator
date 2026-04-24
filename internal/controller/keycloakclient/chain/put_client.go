package chain

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

const (
	secretLength = 32 // 32 bytes = 256 bits of entropy

	browserAuthFlow     = "browser"
	directGrantAuthFlow = "direct_grant"
)

// secretRef is an interface for getting secret from ref.
type secretRef interface {
	GetSecretFromRef(ctx context.Context, refVal, secretNamespace string) (string, error)
}

type PutClient struct {
	kClient   *keycloakapi.KeycloakClient
	k8sClient client.Client
	secretRef secretRef
}

func NewPutClient(kClient *keycloakapi.KeycloakClient, k8sClient client.Client, secretRef secretRef) *PutClient {
	return &PutClient{kClient: kClient, k8sClient: k8sClient, secretRef: secretRef}
}

func (h *PutClient) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string, clientCtx *ClientContext) error {
	id, err := h.putKeycloakClient(ctx, keycloakClient, realmName)
	if err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync client: %s", err.Error()))

		return fmt.Errorf("unable to put keycloak client: %w", err)
	}

	keycloakClient.Status.ClientID = id
	clientCtx.ClientUUID = id

	h.setSuccessCondition(ctx, keycloakClient, "Client synchronized with Keycloak")

	return nil
}

func (h *PutClient) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionClientSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (h *PutClient) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionClientSynced,
		metav1.ConditionTrue,
		ReasonClientUpdated,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (h *PutClient) putKeycloakClient(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) (string, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start creation of Keycloak client")

	var (
		authFlowOverrides map[string]string
		err               error
	)

	if keycloakClient.Spec.AuthenticationFlowBindingOverrides != nil {
		authFlowOverrides, err = h.getAuthFlows(ctx, keycloakClient, realmName)
		if err != nil {
			return "", fmt.Errorf("unable to get auth flows: %w", err)
		}
	}

	clientSecret, err := h.getClientSecret(ctx, keycloakClient)
	if err != nil {
		return "", fmt.Errorf("error getting client secret: %w", err)
	}

	clientRep := convertSpecToClientRepresentation(&keycloakClient.Spec, clientSecret, authFlowOverrides)

	existingClient, _, err := h.kClient.Clients.GetClientByClientID(ctx, realmName, keycloakClient.Spec.ClientId)
	if err != nil && !keycloakapi.IsNotFound(err) {
		return "", fmt.Errorf("unable to check client id: %w", err)
	}

	if existingClient != nil && existingClient.Id != nil {
		log.Info("Client already exists")

		clientUUID := *existingClient.Id
		if _, updErr := h.kClient.Clients.UpdateClient(ctx, realmName, clientUUID, clientRep); updErr != nil {
			return "", fmt.Errorf("unable to update keycloak client: %w", updErr)
		}

		return clientUUID, nil
	}

	resp, err := h.kClient.Clients.CreateClient(ctx, realmName, clientRep)
	if err != nil {
		return "", fmt.Errorf("unable to create client: %w", err)
	}

	log.Info("End put keycloak client")

	id := keycloakapi.GetResourceIDFromResponse(resp)
	if id == "" {
		// Fallback: look up the client to get the UUID
		created, _, lookupErr := h.kClient.Clients.GetClientByClientID(ctx, realmName, keycloakClient.Spec.ClientId)
		if lookupErr != nil {
			return "", fmt.Errorf("unable to get client id after creation: %w", lookupErr)
		}

		if created == nil || created.Id == nil {
			return "", fmt.Errorf("created client has no ID")
		}

		id = *created.Id
	}

	return id, nil
}

func (h *PutClient) getClientSecret(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) (string, error) {
	if keycloakClient.Spec.Public {
		return "", nil
	}

	return h.getSecret(ctx, keycloakClient)
}

func (h *PutClient) getSecret(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) (string, error) {
	if keycloakClient.Spec.Secret != "" {
		// We need to set secret in a new format for old clients for backward compatibility.
		// TODO: This code can be removed in the future.
		if !secretref.HasSecretRef(keycloakClient.Spec.Secret) {
			if err := h.setSecretRef(ctx, keycloakClient); err != nil {
				return "", err
			}
		}

		secretVal, err := h.secretRef.GetSecretFromRef(ctx, keycloakClient.Spec.Secret, keycloakClient.Namespace)
		if err != nil {
			return "", fmt.Errorf("unable to get secret from ref: %w", err)
		}

		return secretVal, nil
	}

	return h.generateSecret(ctx, keycloakClient)
}

func (h *PutClient) generateSecret(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) (string, error) {
	var clientSecret corev1.Secret

	secretName := fmt.Sprintf("keycloak-client-%s-secret", keycloakClient.Name)

	secretErr := h.k8sClient.Get(ctx, types.NamespacedName{Namespace: keycloakClient.Namespace,
		Name: secretName}, &clientSecret)
	if secretErr != nil && !k8sErrors.IsNotFound(secretErr) {
		return "", fmt.Errorf("unable to check client secret existence: %w", secretErr)
	}

	pass, err := generateSecureSecret()
	if err != nil {
		return "", fmt.Errorf("unable to generate secret: %w", err)
	}

	if k8sErrors.IsNotFound(secretErr) {
		clientSecret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: keycloakClient.Namespace,
				Name: secretName},
			Data: map[string][]byte{
				keycloakApi.ClientSecretKey: []byte(pass),
			},
		}

		if err := controllerutil.SetControllerReference(keycloakClient, &clientSecret, h.k8sClient.Scheme()); err != nil {
			return "", fmt.Errorf("unable to set controller ref for secret: %w", err)
		}

		if err := h.k8sClient.Create(ctx, &clientSecret); err != nil {
			return "", fmt.Errorf("unable to create secret %+v, err: %w", clientSecret, err)
		}
	}

	keycloakClient.Spec.Secret = secretref.GenerateSecretRef(clientSecret.Name, keycloakApi.ClientSecretKey)

	if err := h.k8sClient.Update(ctx, keycloakClient); err != nil {
		return "", fmt.Errorf("unable to update client with new secret: %s, err: %w", clientSecret.Name, err)
	}

	return string(clientSecret.Data[keycloakApi.ClientSecretKey]), nil
}

func (h *PutClient) setSecretRef(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) error {
	ref := secretref.GenerateSecretRef(keycloakClient.Spec.Secret, keycloakApi.ClientSecretKey)
	keycloakClient.Spec.Secret = ref

	if err := h.k8sClient.Update(ctx, keycloakClient); err != nil {
		return fmt.Errorf("unable to update client with secret ref %s: %w", ref, err)
	}

	return nil
}

func (h *PutClient) getAuthFlows(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) (map[string]string, error) {
	clientAuthFlows := keycloakClient.Spec.AuthenticationFlowBindingOverrides

	flows, _, err := h.kClient.Realms.GetAuthenticationFlows(ctx, realmName)
	if err != nil {
		return nil, fmt.Errorf("unable to get realm: %w", err)
	}

	realmAuthFlows := maputil.SliceToMap(flows,
		func(f keycloakapi.AuthenticationFlowRepresentation) (string, bool) {
			return *f.Alias, f.Alias != nil && f.Id != nil
		},
		func(f keycloakapi.AuthenticationFlowRepresentation) string { return *f.Id },
	)

	authFlowOverrides := make(map[string]string)

	if clientAuthFlows.Browser != "" {
		if _, ok := realmAuthFlows[clientAuthFlows.Browser]; !ok {
			return nil, fmt.Errorf("auth flow %s not found in realm %s", clientAuthFlows.Browser, realmName)
		}

		authFlowOverrides[browserAuthFlow] = realmAuthFlows[clientAuthFlows.Browser]
	}

	if clientAuthFlows.DirectGrant != "" {
		if _, ok := realmAuthFlows[clientAuthFlows.DirectGrant]; !ok {
			return nil, fmt.Errorf("auth flow %s not found in realm %s", clientAuthFlows.DirectGrant, realmName)
		}

		authFlowOverrides[directGrantAuthFlow] = realmAuthFlows[clientAuthFlows.DirectGrant]
	}

	return authFlowOverrides, nil
}

// generateSecureSecret generates a cryptographically secure random secret
func generateSecureSecret() (string, error) {
	bytes := make([]byte, secretLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// convertSpecToClientRepresentation converts KeycloakClientSpec to keycloakapi.ClientRepresentation.
func convertSpecToClientRepresentation(
	spec *keycloakApi.KeycloakClientSpec,
	clientSecret string,
	authFlowOverrides map[string]string,
) keycloakapi.ClientRepresentation {
	serviceAccountsEnabled := spec.ServiceAccount != nil && spec.ServiceAccount.Enabled

	protocol := ""
	if spec.Protocol != nil {
		protocol = *spec.Protocol
	}

	cr := keycloakapi.ClientRepresentation{
		ClientId:                     &spec.ClientId,
		Name:                         &spec.Name,
		Description:                  &spec.Description,
		Enabled:                      &spec.Enabled,
		PublicClient:                 &spec.Public,
		DirectAccessGrantsEnabled:    &spec.DirectAccess,
		StandardFlowEnabled:          &spec.StandardFlowEnabled,
		ImplicitFlowEnabled:          &spec.ImplicitFlowEnabled,
		AuthorizationServicesEnabled: &spec.AuthorizationServicesEnabled,
		BearerOnly:                   &spec.BearerOnly,
		ConsentRequired:              &spec.ConsentRequired,
		FullScopeAllowed:             &spec.FullScopeAllowed,
		SurrogateAuthRequired:        &spec.SurrogateAuthRequired,
		ServiceAccountsEnabled:       &serviceAccountsEnabled,
		FrontchannelLogout:           &spec.FrontChannelLogout,
		ClientAuthenticatorType:      &spec.ClientAuthenticatorType,
	}

	if protocol != "" {
		cr.Protocol = &protocol
	}

	if spec.WebUrl != "" {
		cr.RootUrl = &spec.WebUrl

		if spec.HomeUrl == "" {
			cr.BaseUrl = &spec.WebUrl
		}
	}

	if spec.HomeUrl != "" {
		cr.BaseUrl = &spec.HomeUrl
	}

	if spec.AdminUrl != "" {
		cr.AdminUrl = &spec.AdminUrl
	}

	if clientSecret != "" {
		cr.Secret = &clientSecret
	}

	if len(spec.Attributes) > 0 {
		attrs := make(map[string]string, len(spec.Attributes))
		maps.Copy(attrs, spec.Attributes)

		cr.Attributes = &attrs
	}

	if len(spec.RedirectUris) > 0 {
		uris := make([]string, len(spec.RedirectUris))
		copy(uris, spec.RedirectUris)

		cr.RedirectUris = &uris
	} else if spec.WebUrl != "" {
		uris := []string{spec.WebUrl + "/*"}
		cr.RedirectUris = &uris
	}

	if len(spec.WebOrigins) > 0 {
		origins := make([]string, len(spec.WebOrigins))
		copy(origins, spec.WebOrigins)

		cr.WebOrigins = &origins
	} else if spec.WebUrl != "" {
		origins := []string{spec.WebUrl}
		cr.WebOrigins = &origins
	}

	if len(authFlowOverrides) > 0 {
		cr.AuthenticationFlowBindingOverrides = &authFlowOverrides
	}

	return cr
}
