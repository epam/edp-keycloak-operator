package chain

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/realmbuilder"
)

var log = ctrl.Log.WithName("realm_handler")

func CreateDefChain(k8sClient client.Client, scheme *runtime.Scheme, controllerHelper Helper) handler.RealmHandler {
	return PutRealm{
		hlp:    controllerHelper,
		client: k8sClient,
		next: SetLabels{
			client: k8sClient,
			next: PutUsers{
				next: PutUsersRoles{
					next: RealmSettings{
						settingsBuilder: realmbuilder.NewSettingsBuilder(),
						next: AuthFlow{
							next: UserProfile{
								next: ConfigureEmail{
									client: k8sClient,
								},
							},
						},
					},
				},
			},
		},
	}
}

func nextServeOrNil(ctx context.Context, next handler.RealmHandler, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
	if next != nil {
		err := next.ServeRequest(ctx, realm, kClient)
		if err != nil {
			return fmt.Errorf("chain failed %s: %w", reflect.TypeOf(next).Name(), err)
		}

		return nil
	}

	log.Info("handling of realm has been finished", "realm name", realm.Spec.RealmName)

	return nil
}
