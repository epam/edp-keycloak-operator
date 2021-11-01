package chain

import (
	"context"

	v1v1alpha1 "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = ctrl.Log.WithName("realm_handler")

func CreateDefChain(client client.Client, scheme *runtime.Scheme, hlp Helper) handler.RealmHandler {
	return PutRealm{
		hlp: hlp,
		next: SetLabels{
			next: PutKeycloakClientScope{
				next: PutKeycloakClientCR{
					next: PutKeycloakClientSecret{
						next: PutUsers{
							next: PutUsersRoles{
								next: PutOpenIdConfigAnnotation{
									next: PutIdentityProvider{
										next: PutDefaultIdP{
											next: RealmSettings{
												next: AuthFlow{},
											},
										},
										client: client,
									},
									client: client,
								},
							},
						},
						client: client,
						scheme: scheme,
					},
					client: client,
					scheme: scheme,
				},
				client: client,
				scheme: scheme,
			},
			client: client,
		},
		client: client,
	}
}

func nextServeOrNil(ctx context.Context, next handler.RealmHandler, realm *v1v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	if next != nil {
		return next.ServeRequest(ctx, realm, kClient)
	}

	log.Info("handling of realm has been finished", "realm name", realm.Spec.RealmName)
	return nil
}
