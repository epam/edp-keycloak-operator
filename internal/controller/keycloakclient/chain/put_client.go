package chain

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
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
	GetSecretFromRef(ctx context.Context, refVal, secretNamespace string) (value, version string, err error)
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

	clientSecret, secretVersion, err := h.getClientSecret(ctx, keycloakClient)
	if err != nil {
		return "", fmt.Errorf("error getting client secret: %w", err)
	}

	clientRep := convertSpecToClientRepresentation(&keycloakClient.Spec, clientSecret, authFlowOverrides)
	newHash := clientSecretsHash(secretVersion)

	existingClient, _, err := h.kClient.Clients.GetClientByClientID(ctx, realmName, keycloakClient.Spec.ClientId)
	if err != nil && !keycloakapi.IsNotFound(err) {
		return "", fmt.Errorf("unable to check client id: %w", err)
	}

	if existingClient != nil && existingClient.Id != nil {
		log.Info("Client already exists")

		clientUUID := *existingClient.Id

		needsUpdate := helper.SpecChanged(keycloakClient.Generation, keycloakClient.Status.ObservedGeneration) ||
			newHash != keycloakClient.Status.ConfigSecretsHash ||
			!clientMatchesSpec(existingClient, &clientRep)

		if needsUpdate {
			if _, updErr := h.kClient.Clients.UpdateClient(ctx, realmName, clientUUID, clientRep); updErr != nil {
				return "", fmt.Errorf("unable to update keycloak client: %w", updErr)
			}
		}

		// ObservedGeneration is stamped by the controller after the whole chain succeeds.
		keycloakClient.Status.ConfigSecretsHash = newHash

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

	keycloakClient.Status.ConfigSecretsHash = newHash

	return id, nil
}

// Returns "" when version is ""; otherwise ValuesHashSingle over the version token.
func clientSecretsHash(version string) string {
	if version == "" {
		return ""
	}

	return secretref.ValuesHashSingle(map[string]string{"spec.secret": version})
}

func (h *PutClient) getClientSecret(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) (value, version string, err error) {
	if keycloakClient.Spec.Public {
		return "", "", nil
	}

	return h.getSecret(ctx, keycloakClient)
}

func (h *PutClient) getSecret(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) (value, version string, err error) {
	if keycloakClient.Spec.Secret != "" {
		// We need to set secret in a new format for old clients for backward compatibility.
		// TODO: This code can be removed in the future.
		if !secretref.HasSecretRef(keycloakClient.Spec.Secret) {
			if err := h.setSecretRef(ctx, keycloakClient); err != nil {
				return "", "", err
			}
		}

		secretVal, secretVersion, err := h.secretRef.GetSecretFromRef(ctx, keycloakClient.Spec.Secret, keycloakClient.Namespace)
		if err != nil {
			return "", "", fmt.Errorf("unable to get secret from ref: %w", err)
		}

		return secretVal, secretVersion, nil
	}

	return h.generateSecret(ctx, keycloakClient)
}

func (h *PutClient) generateSecret(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) (value, version string, err error) {
	var clientSecret corev1.Secret

	secretName := fmt.Sprintf("keycloak-client-%s-secret", keycloakClient.Name)

	secretErr := h.k8sClient.Get(ctx, types.NamespacedName{Namespace: keycloakClient.Namespace,
		Name: secretName}, &clientSecret)
	if secretErr != nil && !k8sErrors.IsNotFound(secretErr) {
		return "", "", fmt.Errorf("unable to check client secret existence: %w", secretErr)
	}

	pass, err := generateSecureSecret()
	if err != nil {
		return "", "", fmt.Errorf("unable to generate secret: %w", err)
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
			return "", "", fmt.Errorf("unable to set controller ref for secret: %w", err)
		}

		if err := h.k8sClient.Create(ctx, &clientSecret); err != nil {
			return "", "", fmt.Errorf("unable to create secret %s/%s: %w", keycloakClient.Namespace, secretName, err)
		}
	}

	keycloakClient.Spec.Secret = secretref.GenerateSecretRef(clientSecret.Name, keycloakApi.ClientSecretKey)

	if err := h.k8sClient.Update(ctx, keycloakClient); err != nil {
		return "", "", fmt.Errorf("unable to update client with new secret: %s, err: %w", clientSecret.Name, err)
	}

	return string(clientSecret.Data[keycloakApi.ClientSecretKey]),
		secretref.SecretKeyVersion(&clientSecret, keycloakApi.ClientSecretKey), nil
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

	attrs := make(map[string]string, len(spec.Attributes))
	maps.Copy(attrs, spec.Attributes)

	if spec.AdvancedSettings != nil && spec.AdvancedSettings.AccessTokenLifespan != nil {
		attrs["access.token.lifespan"] = strconv.Itoa(*spec.AdvancedSettings.AccessTokenLifespan)
	}

	if len(attrs) > 0 {
		cr.Attributes = &attrs
	}

	// RedirectUris, WebOrigins: nil derives from WebUrl, or omits the key so Keycloak keeps its list.
	// Non-nil, including empty: sent as-is; an empty list clears it in Keycloak.
	if spec.RedirectUris != nil {
		uris := make([]string, len(*spec.RedirectUris))
		copy(uris, *spec.RedirectUris)

		cr.RedirectUris = &uris
	} else if spec.WebUrl != "" {
		uris := []string{spec.WebUrl + "/*"}
		cr.RedirectUris = &uris
	}

	if spec.WebOrigins != nil {
		origins := make([]string, len(*spec.WebOrigins))
		copy(origins, *spec.WebOrigins)

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

// clientMatchesSpec compares the keys the PUT body sends against the fetched representation.
// A masked fetched secret or attribute value is skipped. Attributes and flow binding overrides are compared as
// subsets; redirect URIs and web origins as sets without blank entries.
func clientMatchesSpec(existing, desired *keycloakapi.ClientRepresentation) bool {
	strs := []struct{ existing, desired *string }{
		{existing.ClientId, desired.ClientId},
		{existing.Name, desired.Name},
		{existing.Description, desired.Description},
		{existing.Protocol, desired.Protocol},
		{existing.RootUrl, desired.RootUrl},
		{existing.BaseUrl, desired.BaseUrl},
		{existing.AdminUrl, desired.AdminUrl},
		{existing.ClientAuthenticatorType, desired.ClientAuthenticatorType},
	}

	for _, p := range strs {
		if p.desired != nil && ptr.Deref(p.existing, "") != *p.desired {
			return false
		}
	}

	// Keycloak forces authorizationServicesEnabled to false for public and bearer-only clients.
	desiredAuthz := desired.AuthorizationServicesEnabled
	if ptr.Deref(desired.PublicClient, false) || ptr.Deref(desired.BearerOnly, false) {
		desiredAuthz = ptr.To(false)
	}

	bools := []struct{ existing, desired *bool }{
		{existing.Enabled, desired.Enabled},
		{existing.PublicClient, desired.PublicClient},
		{existing.DirectAccessGrantsEnabled, desired.DirectAccessGrantsEnabled},
		{existing.StandardFlowEnabled, desired.StandardFlowEnabled},
		{existing.ImplicitFlowEnabled, desired.ImplicitFlowEnabled},
		{existing.AuthorizationServicesEnabled, desiredAuthz},
		{existing.BearerOnly, desired.BearerOnly},
		{existing.ConsentRequired, desired.ConsentRequired},
		{existing.FullScopeAllowed, desired.FullScopeAllowed},
		{existing.SurrogateAuthRequired, desired.SurrogateAuthRequired},
		{existing.ServiceAccountsEnabled, desired.ServiceAccountsEnabled},
		{existing.FrontchannelLogout, desired.FrontchannelLogout},
	}

	for _, p := range bools {
		if p.desired != nil && ptr.Deref(p.existing, false) != *p.desired {
			return false
		}
	}

	if desired.Secret != nil &&
		ptr.Deref(existing.Secret, "") != keycloakapi.MaskedSecretValue &&
		ptr.Deref(existing.Secret, "") != *desired.Secret {
		return false
	}

	if desired.Attributes != nil && !clientAttributesMatch(ptr.Deref(existing.Attributes, nil), *desired.Attributes) {
		return false
	}

	if desired.RedirectUris != nil && !stringSetsEqual(ptr.Deref(existing.RedirectUris, nil), *desired.RedirectUris) {
		return false
	}

	if desired.WebOrigins != nil && !stringSetsEqual(ptr.Deref(existing.WebOrigins, nil), *desired.WebOrigins) {
		return false
	}

	if desired.AuthenticationFlowBindingOverrides != nil {
		fetched := ptr.Deref(existing.AuthenticationFlowBindingOverrides, nil)

		for k, v := range *desired.AuthenticationFlowBindingOverrides {
			if fetched[k] != v {
				return false
			}
		}
	}

	return true
}

// clientAttributesMatch compares desired attributes against existing with secretref.ConfigMatches:
// a key whose fetched value is keycloakapi.MaskedSecretValue is skipped.
// A blank desired value whose key is absent in existing also matches:
// Keycloak drops blank attribute values on write.
func clientAttributesMatch(existing, desired map[string]string) bool {
	filtered := make(map[string]string, len(desired))

	for k, v := range desired {
		if _, ok := existing[k]; !ok && isJavaTrimBlank(v) {
			continue
		}

		filtered[k] = v
	}

	return secretref.ConfigMatches(existing, filtered, nil)
}

// isJavaTrimBlank matches "".equals(s.trim()): every rune is <= U+0020.
func isJavaTrimBlank(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return r > ' ' }) < 0
}

// isJavaBlank matches Java String.isBlank: every rune is Character.isWhitespace.
func isJavaBlank(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return !isJavaWhitespace(r) }) < 0
}

// isJavaWhitespace matches Java Character.isWhitespace: Unicode space, line and paragraph
// separators except the non-breaking U+00A0, U+2007, U+202F, plus \t \n \u000B \f \r \u001C-\u001F.
func isJavaWhitespace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', '\u001C', '\u001D', '\u001E', '\u001F':
		return true
	case '\u00A0', '\u2007', '\u202F':
		return false
	}

	return unicode.In(r, unicode.Zs, unicode.Zl, unicode.Zp)
}

// stringSetsEqual compares as sets; blank entries in desired are dropped (Keycloak filterEmptyValues, String.isBlank).
func stringSetsEqual(existing, desired []string) bool {
	want := slices.DeleteFunc(slices.Clone(desired), isJavaBlank)
	slices.Sort(want)

	have := slices.Clone(existing)
	slices.Sort(have)

	return slices.Equal(slices.Compact(have), slices.Compact(want))
}
