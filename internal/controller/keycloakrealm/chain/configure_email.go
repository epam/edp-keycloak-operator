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
	"github.com/epam/edp-keycloak-operator/pkg/destination"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

type ConfigureEmail struct {
	next   handler.RealmHandler
	client client.Client
	guard  *destination.Guard
}

func (s ConfigureEmail) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient *keycloakapi.KeycloakClient) error {
	if realm.Spec.Smtp == nil {
		return nextServeOrNil(ctx, s.next, realm, kClient)
	}

	l := ctrl.LoggerFrom(ctx)
	l.Info("Configuring email for realm")

	newHash, err := ConfigureRealmEmail(ctx, RealmEmailParams{
		RealmName:        realm.Spec.RealmName,
		EmailSpec:        realm.Spec.Smtp,
		SecretsNamespace: realm.Namespace,
		RealmClient:      kClient.Realms,
		K8sClient:        s.client,
		StoredHash:       realm.Status.ConfigSecretsHash,
		ForceWrite:       helper.SpecChanged(realm.Generation, realm.Status.ObservedGeneration),
		Guard:            s.guard,
	})
	if err != nil {
		return err
	}

	realm.Status.ConfigSecretsHash = newHash

	l.Info("Email has been configured")

	return nextServeOrNil(ctx, s.next, realm, kClient)
}

// RealmEmailParams carries the inputs of ConfigureRealmEmail.
type RealmEmailParams struct {
	RealmName        string
	EmailSpec        *common.SMTP
	SecretsNamespace string
	RealmClient      keycloakapi.RealmClient
	K8sClient        client.Client
	StoredHash       string
	ForceWrite       bool
	Guard            *destination.Guard
}

// ConfigureRealmEmail applies EmailSpec to the realm's SMTP server config and returns the
// resolved-secret hash. GetRealm always runs to obtain the comparison baseline; the UpdateRealm
// write is skipped when ForceWrite is false, StoredHash matches the newly resolved hash, and
// the fetched SMTP config already matches spec.
func ConfigureRealmEmail(ctx context.Context, p RealmEmailParams) (string, error) {
	if p.EmailSpec == nil {
		return "", nil
	}

	// Keycloak, not the operator, dials this host, and it carries the SMTP password.
	// Checked before the password Secret is resolved below.
	if err := p.Guard.RequireHost(ctx, "spec.smtp.connection.host", p.EmailSpec.Connection.Host); err != nil {
		return "", err
	}

	current, _, err := p.RealmClient.GetRealm(ctx, p.RealmName)
	if err != nil {
		return "", fmt.Errorf("unable to get realm %v: %w", p.RealmName, err)
	}

	emailMap, passwordVersion, err := convertEmailSpecToMap(ctx, p.EmailSpec, p.SecretsNamespace, p.K8sClient)
	if err != nil {
		return "", err
	}

	// Rotating the password's k8s Secret bumps no CR generation; the hash of the secret's
	// version token forces the write instead.
	passwordVersions := map[string]string{}
	if p.EmailSpec.Connection.Authentication != nil {
		passwordVersions["password"] = passwordVersion
	}

	newHash := secretref.ValuesHashSingle(passwordVersions)

	if !p.ForceWrite && p.StoredHash == newHash && smtpMatchesSpec(current.SmtpServer, emailMap) {
		return newHash, nil
	}

	current.SmtpServer = &emailMap

	if _, err = p.RealmClient.UpdateRealm(ctx, p.RealmName, *current); err != nil {
		return "", fmt.Errorf("unable to update realm %v: %w", p.RealmName, err)
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

// convertEmailSpecToMap resolves emailSpec into the realm SMTP config map. The second return
// is the password's version token ("" without authentication).
func convertEmailSpecToMap(
	ctx context.Context,
	emailSpec *common.SMTP,
	secretsNamespace string,
	k8sClient client.Client,
) (map[string]string, string, error) {
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

	passwordVersion := ""

	if emailSpec.Connection.Authentication != nil {
		username, err := secretref.GetValueFromSourceRefOrVal(
			ctx,
			&emailSpec.Connection.Authentication.Username,
			secretsNamespace,
			k8sClient,
		)
		if err != nil {
			return nil, "", fmt.Errorf("unable to get username: %w", err)
		}

		emailMap["user"] = username

		password, version, err := secretref.GetValueAndVersionFromSourceRef(
			ctx,
			&emailSpec.Connection.Authentication.Password,
			secretsNamespace,
			k8sClient,
		)
		if err != nil {
			return nil, "", fmt.Errorf("unable to get password: %w", err)
		}

		emailMap["password"] = password
		passwordVersion = version
	}

	return emailMap, passwordVersion, nil
}
