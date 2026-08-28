package chain

import (
	"context"
	"fmt"
	"maps"
	"strconv"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

type ConfigureEmail struct {
	next   handler.RealmHandler
	client client.Client
}

func (s ConfigureEmail) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient *keycloakapi.KeycloakClient) error {
	if realm.Spec.Smtp == nil {
		return nextServeOrNil(ctx, s.next, realm, kClient)
	}

	l := ctrl.LoggerFrom(ctx)
	l.Info("Configuring email for realm")

	newHash, err := ConfigureRealmEmail(
		ctx,
		realm.Spec.RealmName,
		realm.Spec.Smtp,
		realm.Namespace,
		kClient.Realms,
		s.client,
		realm.Status.ConfigSecretsHash,
		helper.SpecChanged(realm.Generation, realm.Status.ObservedGeneration),
	)
	if err != nil {
		return err
	}

	realm.Status.ConfigSecretsHash = newHash

	l.Info("Email has been configured")

	return nextServeOrNil(ctx, s.next, realm, kClient)
}

// ConfigureRealmEmail applies emailSpec to the realm's SMTP server config and returns the
// resolved-secret hash. GetRealm always runs to obtain the comparison baseline; the UpdateRealm
// write is skipped when forceWrite is false, storedHash matches the newly resolved hash, and
// the fetched SMTP config already matches spec.
func ConfigureRealmEmail(
	ctx context.Context,
	realmName string,
	emailSpec *common.SMTP,
	secretsNamespace string,
	realmClient keycloakapi.RealmClient,
	k8sClient client.Client,
	storedHash string,
	forceWrite bool,
) (string, error) {
	if emailSpec == nil {
		return "", nil
	}

	current, _, err := realmClient.GetRealm(ctx, realmName)
	if err != nil {
		return "", fmt.Errorf("unable to get realm %v: %w", realmName, err)
	}

	emailMap, err := convertEmailSpecToMap(ctx, emailSpec, secretsNamespace, k8sClient)
	if err != nil {
		return "", err
	}

	// Rotating the password's k8s Secret bumps no CR generation; the hash of the resolved
	// value forces the write instead.
	passwordValues := map[string][]string{}
	if emailSpec.Connection.Authentication != nil {
		passwordValues["password"] = []string{emailMap["password"]}
	}

	newHash := secretref.ValuesHash(passwordValues)

	if !forceWrite && storedHash == newHash && smtpMatchesSpec(current.SmtpServer, emailMap) {
		return newHash, nil
	}

	current.SmtpServer = &emailMap

	if _, err = realmClient.UpdateRealm(ctx, realmName, *current); err != nil {
		return "", fmt.Errorf("unable to update realm %v: %w", realmName, err)
	}

	return newHash, nil
}

// smtpMatchesSpec reports whether the fetched SMTP config already matches the desired one.
// The "password" value is excluded from the comparison since Keycloak masks it on GET; key
// sets must match exactly so keys removed from the spec still trigger a write.
func smtpMatchesSpec(existing *map[string]string, desired map[string]string) bool {
	if existing == nil || len(*existing) != len(desired) {
		return false
	}

	_, existingHasPassword := (*existing)["password"]
	_, desiredHasPassword := desired["password"]

	if existingHasPassword != desiredHasPassword {
		return false
	}

	want := maps.Clone(desired)
	delete(want, "password")

	return maputil.ContainsSubset(*existing, want)
}

func convertEmailSpecToMap(
	ctx context.Context,
	emailSpec *common.SMTP,
	secretsNamespace string,
	k8sClient client.Client,
) (map[string]string, error) {
	emailMap := make(map[string]string)
	emailMap["from"] = emailSpec.Template.From
	emailMap["fromDisplayName"] = emailSpec.Template.FromDisplayName
	emailMap["replyTo"] = emailSpec.Template.ReplyTo
	emailMap["replyToDisplayName"] = emailSpec.Template.ReplyToDisplayName
	emailMap["envelopeFrom"] = emailSpec.Template.EnvelopeFrom
	emailMap["host"] = emailSpec.Connection.Host
	emailMap["port"] = strconv.Itoa(emailSpec.Connection.Port)
	emailMap["ssl"] = strconv.FormatBool(emailSpec.Connection.EnableSSL)
	emailMap["starttls"] = strconv.FormatBool(emailSpec.Connection.EnableStartTLS)
	emailMap["auth"] = strconv.FormatBool(emailSpec.Connection.Authentication != nil)

	if emailSpec.Connection.Authentication != nil {
		username, err := secretref.GetValueFromSourceRefOrVal(
			ctx,
			&emailSpec.Connection.Authentication.Username,
			secretsNamespace,
			k8sClient,
		)
		if err != nil {
			return nil, fmt.Errorf("unable to get username: %w", err)
		}

		emailMap["user"] = username

		password, err := secretref.GetValueFromSourceRef(
			ctx,
			&emailSpec.Connection.Authentication.Password,
			secretsNamespace,
			k8sClient,
		)
		if err != nil {
			return nil, fmt.Errorf("unable to get password: %w", err)
		}

		emailMap["password"] = password
	}

	return emailMap, nil
}
