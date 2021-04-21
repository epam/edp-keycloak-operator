package keycloakclient

import (
	"context"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakclient/chain"
	"github.com/go-logr/logr"
	pkgErrors "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	Fail                                = "FAIL"
	keyCloakClientOperatorFinalizerName = "keycloak.client.operator.finalizer.name"
)

func NewReconcileKeycloakClient(client client.Client, log logr.Logger, helper *helper.Helper) *ReconcileKeycloakClient {
	return &ReconcileKeycloakClient{
		client: client,
		helper: helper,
		log:    log.WithName("keycloak-client"),
	}
}

// ReconcileKeycloakClient reconciles a KeycloakClient object
type ReconcileKeycloakClient struct {
	client client.Client
	helper *helper.Helper
	log    logr.Logger
}

func (r *ReconcileKeycloakClient) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakClient{}, builder.WithPredicates(pred)).
		Complete(r)
}

func (r *ReconcileKeycloakClient) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling KeycloakClient")

	var instance keycloakApi.KeycloakClient
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return
		}
		resultErr = err
		return
	}

	if err := r.tryReconcile(&instance); err != nil {
		instance.Status.Value = Fail
		result.RequeueAfter = r.helper.SetFailureCount(&instance)
		log.Error(err, "an error has occurred while handling keycloak client", "name",
			request.Name)
	} else {
		helper.SetSuccessStatus(&instance)
	}

	if err := r.helper.UpdateStatus(&instance); err != nil {
		resultErr = pkgErrors.Wrap(err, "unable to update status")
	}

	return
}

func (r *ReconcileKeycloakClient) tryReconcile(keycloakClient *keycloakApi.KeycloakClient) error {
	ch := chain.Make(r.helper, r.client, ctrl.Log.WithName("chain").WithName("keycloak-client"),
		adapter.GoCloakAdapterFactory{
			Log: ctrl.Log.WithName("go-cloak-adapter-factory"),
		})
	if err := ch.Serve(keycloakClient); err != nil {
		return pkgErrors.Wrap(err, "error during kc chain")
	}

	if _, err := r.helper.TryToDelete(keycloakClient, makeTerminator(keycloakClient.Status.ClientID,
		keycloakClient.Spec.TargetRealm, ch.GetState().AdapterClient),
		keyCloakClientOperatorFinalizerName); err != nil {
		return pkgErrors.Wrap(err, "unable to delete kc client")
	}

	return nil
}
