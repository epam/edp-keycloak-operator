package chain

import (
	"context"
	"fmt"
	"strconv"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

type ConfigureEmail struct {
	next   handler.RealmHandler
	client client.Client
}

func (s ConfigureEmail) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
	if realm.Spec.Smtp == nil {
		return nextServeOrNil(ctx, s.next, realm, kClient)
	}

	l := ctrl.LoggerFrom(ctx)
	l.Info("Configuring email for realm")

	if err := ConfigureRamlEmail(
		ctx,
		realm.Spec.RealmName,
		realm.Spec.Smtp,
		realm.Namespace,
		kClient,
		s.client,
	); err != nil {
		return err
	}

	l.Info("Email has been configured")

	return nextServeOrNil(ctx, s.next, realm, kClient)
}

func ConfigureRamlEmail(
	ctx context.Context,
	realmName string,
	emailSpec *common.SMTP,
	secretsNamespace string,
	kcClient keycloak.Client,
	k8sClient client.Client,
) error {
	if emailSpec == nil {
		return nil
	}

	realm, err := kcClient.GetRealm(ctx, realmName)
	if err != nil {
		return fmt.Errorf("unable to get realm %v: %w", realmName, err)
	}

	emailMap, err := convertEmailSpecToMap(ctx, emailSpec, secretsNamespace, k8sClient)
	if err != nil {
		return err
	}

	realm.SMTPServer = &emailMap

	if err = kcClient.UpdateRealm(ctx, realm); err != nil {
		return fmt.Errorf("unable to update realm %v: %w", realmName, err)
	}

	return nil
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
