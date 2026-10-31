package chain

import (
	"context"
	stderrors "errors"
	"fmt"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

var (
	// errSecretKeyNotFound is returned when the specified key is not found in the secret.
	errSecretKeyNotFound = stderrors.New("secret key not found")
)

const (
	// ConditionPasswordSynced indicates whether the user password has been synced to Keycloak.
	ConditionPasswordSynced = "PasswordSynced"

	// ReasonPasswordSetFromSecret indicates that non-temporary password was set from secret.
	ReasonPasswordSetFromSecret = "PasswordSetFromSecret"

	// ReasonTemporaryPasswordSet indicates that temporary password was set from secret (written once per source).
	ReasonTemporaryPasswordSet = "TemporaryPasswordSet"

	// ReasonPasswordSetFromSpec indicates that password was set from deprecated spec.password field.
	ReasonPasswordSetFromSpec = "PasswordSetFromSpec"

	// ReasonSecretNotFound indicates the password secret does not exist.
	ReasonSecretNotFound = "SecretNotFound"

	// ReasonSecretKeyMissing indicates the specified key is missing from the secret.
	ReasonSecretKeyMissing = "SecretKeyMissing"

	// ReasonKeycloakAPIError indicates Keycloak API call failed.
	ReasonKeycloakAPIError = "KeycloakAPIError"
)

// passwordResult holds the password and metadata about its source.
type passwordResult struct {
	Value         string
	Temporary     bool
	SecretVersion string // resourceVersion of the secret, empty if password is from spec
	FromSecret    bool   // true if password is from secret, false if from spec.password
	SourceHash    string // passwordSourceHash of the current source, empty if password is from spec
}

type passwordSyncDecision int

const (
	passwordWrite passwordSyncDecision = iota
	passwordSkip
)

type SetUserPassword struct {
	k8sClient client.Client
	kClient   *keycloakapi.KeycloakClient
}

func NewSetUserPassword(k8sClient client.Client, kClient *keycloakapi.KeycloakClient) *SetUserPassword {
	return &SetUserPassword{k8sClient: k8sClient, kClient: kClient}
}

func (h *SetUserPassword) Serve(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	realmName string,
	userCtx *UserContext,
) error {
	log := ctrl.LoggerFrom(ctx)

	if user.Spec.PasswordSecret == nil && user.Spec.Password == "" {
		log.Info("No password configured, skipping password sync")

		return nil
	}

	pwdResult, err := h.getPassword(ctx, user, realmName, userCtx.UserID)
	if err != nil {
		h.setPasswordErrorCondition(ctx, user, err)

		return err
	}

	switch decidePasswordSync(user, pwdResult) {
	case passwordSkip:
		log.Info("Password source unchanged, skipping password sync",
			"secret", user.Spec.PasswordSecret.Name, "key", user.Spec.PasswordSecret.Key, "temporary", pwdResult.Temporary)

		setPasswordSyncedCondition(user, pwdResult)

		return nil
	case passwordWrite:
		logPasswordWrite(ctx, user, pwdResult)
	}

	log.Info("Setting user password")

	_, err = h.kClient.Users.SetUserPassword(ctx, realmName, userCtx.UserID, keycloakapi.CredentialRepresentation{
		Type:      ptr.To("password"),
		Value:     ptr.To(pwdResult.Value),
		Temporary: ptr.To(pwdResult.Temporary),
	})
	if err != nil {
		wrappedErr := fmt.Errorf("unable to set user password: %w", err)
		h.setPasswordErrorCondition(ctx, user, wrappedErr)

		return wrappedErr
	}

	setPasswordSyncedCondition(user, pwdResult)

	user.Status.PasswordSecretHash = pwdResult.SourceHash
	user.Status.LastSyncedPasswordSecretVersion = pwdResult.SecretVersion

	log.Info("User password set successfully")

	return nil
}

// passwordSourceHash identifies the passwordSecret source and the write target: Secret name, key, temporary flag,
// realm name and Keycloak user ID. A non-temporary password adds the Secret UID and resourceVersion:
// a Secret rotation rewrites it. A temporary password is written once per source and ignores Secret rotation.
// The input is metadata only, never the password.
func passwordSourceHash(secret *coreV1.Secret, ps *keycloakApi.PasswordSecret, realmName, userID string) string {
	source := "secret:" + ps.Name + ":" + ps.Key
	temporary := "true"

	if !ps.Temporary {
		source = secretref.SecretKeyVersion(secret, ps.Key)
		temporary = "false"
	}

	return secretref.ValuesHashSingle(map[string]string{
		"passwordSecret": source,
		"temporary":      temporary,
		"realm":          realmName,
		"userID":         userID,
	})
}

// decidePasswordSync compares the stored source hash with the current one.
func decidePasswordSync(user *keycloakApi.KeycloakRealmUser, pwdResult *passwordResult) passwordSyncDecision {
	if !pwdResult.FromSecret {
		return passwordWrite
	}

	if user.Status.PasswordSecretHash == pwdResult.SourceHash {
		return passwordSkip
	}

	return passwordWrite
}

func logPasswordWrite(ctx context.Context, user *keycloakApi.KeycloakRealmUser, pwdResult *passwordResult) {
	log := ctrl.LoggerFrom(ctx)

	if !pwdResult.FromSecret {
		log.Info("Password from spec.password (deprecated), will set password")

		return
	}

	log.Info("Password source changed, will set password",
		"secret", user.Spec.PasswordSecret.Name, "key", user.Spec.PasswordSecret.Key, "temporary", pwdResult.Temporary)
}

// setPasswordSyncedCondition sets the PasswordSynced condition to True for the current generation.
func setPasswordSyncedCondition(user *keycloakApi.KeycloakRealmUser, pwdResult *passwordResult) {
	meta.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
		Type:               ConditionPasswordSynced,
		Status:             metav1.ConditionTrue,
		Reason:             getPasswordReason(pwdResult),
		Message:            getPasswordMessage(pwdResult, user.Spec.PasswordSecret),
		ObservedGeneration: user.Generation,
	})
}

// setPasswordErrorCondition sets the PasswordSynced condition to False with appropriate reason and message.
func (h *SetUserPassword) setPasswordErrorCondition(ctx context.Context, user *keycloakApi.KeycloakRealmUser, err error) {
	log := ctrl.LoggerFrom(ctx)

	reason, message := h.classifyPasswordError(err, user)

	log.Info("Setting password error condition", "reason", reason, "message", message)

	meta.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
		Type:               ConditionPasswordSynced,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: user.Generation,
	})
}

// classifyPasswordError determines the appropriate Reason and Message for a password sync error.
func (h *SetUserPassword) classifyPasswordError(err error, user *keycloakApi.KeycloakRealmUser) (reason, message string) {
	// Check for K8s API errors (secret not found)
	if errors.IsNotFound(err) {
		secretName := ""
		if user.Spec.PasswordSecret != nil {
			secretName = user.Spec.PasswordSecret.Name
		}

		return ReasonSecretNotFound, fmt.Sprintf("Password secret %q not found in namespace %q", secretName, user.Namespace)
	}

	// Check for secret key missing error
	if stderrors.Is(err, errSecretKeyNotFound) {
		keyName := ""
		secretName := ""

		if user.Spec.PasswordSecret != nil {
			keyName = user.Spec.PasswordSecret.Key
			secretName = user.Spec.PasswordSecret.Name
		}

		return ReasonSecretKeyMissing, fmt.Sprintf("Key %q not found in secret %q", keyName, secretName)
	}

	// Keycloak API error (default case)
	return ReasonKeycloakAPIError, fmt.Sprintf("Failed to set password in Keycloak: %s", err.Error())
}

func (h *SetUserPassword) getPassword(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	realmName, userID string,
) (*passwordResult, error) {
	log := ctrl.LoggerFrom(ctx)

	if user.Spec.PasswordSecret != nil {
		secret := &coreV1.Secret{}
		if err := h.k8sClient.Get(ctx, types.NamespacedName{Name: user.Spec.PasswordSecret.Name, Namespace: user.Namespace}, secret); err != nil {
			return nil, fmt.Errorf("failed to get secret %s with user password: %w", user.Spec.PasswordSecret.Name, err)
		}

		passwordBytes, ok := secret.Data[user.Spec.PasswordSecret.Key]
		if !ok {
			return nil, fmt.Errorf("key %s not found in secret %s: %w", user.Spec.PasswordSecret.Key, user.Spec.PasswordSecret.Name, errSecretKeyNotFound)
		}

		log.Info("Using password from secret", "secret", user.Spec.PasswordSecret.Name)

		return &passwordResult{
			Value:         string(passwordBytes),
			Temporary:     user.Spec.PasswordSecret.Temporary,
			SecretVersion: secret.ResourceVersion,
			FromSecret:    true,
			SourceHash:    passwordSourceHash(secret, user.Spec.PasswordSecret, realmName, userID),
		}, nil
	}

	log.Info("Using password from spec")

	// Deprecated spec.password field usage. We still support it for backward compatibility.
	return &passwordResult{
		Value:      user.Spec.Password,
		Temporary:  false,
		FromSecret: false,
	}, nil
}

// getPasswordReason returns the appropriate Reason value based on password source.
func getPasswordReason(pwdResult *passwordResult) string {
	if !pwdResult.FromSecret {
		return ReasonPasswordSetFromSpec
	}

	if pwdResult.Temporary {
		return ReasonTemporaryPasswordSet
	}

	return ReasonPasswordSetFromSecret
}

// getPasswordMessage returns a human-readable message for the condition.
func getPasswordMessage(pwdResult *passwordResult, passwordSecret *keycloakApi.PasswordSecret) string {
	if !pwdResult.FromSecret {
		return "Password set from spec.password field (deprecated)"
	}

	secretName := ""
	if passwordSecret != nil {
		secretName = passwordSecret.Name
	}

	if pwdResult.Temporary {
		return fmt.Sprintf("Temporary password set from secret %s (written once per source)", secretName)
	}

	return fmt.Sprintf("Password synced from secret %s", secretName)
}
