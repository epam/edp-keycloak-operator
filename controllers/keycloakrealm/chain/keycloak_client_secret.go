package chain

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type PutKeycloakClientSecret struct {
	next   handler.RealmHandler
	client client.Client
	scheme *runtime.Scheme
}

func (h PutKeycloakClientSecret) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start creation of Keycloak client secret")

	if !realm.Spec.SSOEnabled() {
		rLog.Info("sso realm disabled skip creation of Keycloak client secret")
		return nextServeOrNil(ctx, h.next, realm, kClient)
	}

	sn := fmt.Sprintf(clientSecretName, realm.Spec.RealmName)

	s, err := helper.GetSecret(ctx, h.client, types.NamespacedName{
		Name:      sn,
		Namespace: realm.Namespace,
	})
	if err != nil {
		return fmt.Errorf("failed to get seret %s: %w", sn, err)
	}

	if s != nil {
		rLog.Info("Keycloak client secret already exist")
		return nextServeOrNil(ctx, h.next, realm, kClient)
	}

	s = &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sn,
			Namespace: realm.Namespace,
		},
		Data: map[string][]byte{
			keycloakApi.ClientSecretKey: []byte(uuid.New().String()),
		},
	}
	cl := &keycloakApi.KeycloakClient{}

	err = h.client.Get(ctx, types.NamespacedName{
		Namespace: realm.Namespace,
		Name:      realm.Spec.RealmName,
	}, cl)
	if err != nil {
		return fmt.Errorf("failed to get keycloak client: %w", err)
	}

	err = controllerutil.SetControllerReference(cl, s, h.scheme)
	if err != nil {
		return fmt.Errorf("failed to set controller reference for secret: %w", err)
	}

	err = h.client.Create(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	rLog.Info("End of put Keycloak client secret")

	return nextServeOrNil(ctx, h.next, realm, kClient)
}
