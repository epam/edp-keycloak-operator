package keycloakclient

import (
	"context"

	"github.com/Nerzal/gocloak/v8"
	v1v1alpha1 "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/consts"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
	pkgErrors "github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_keycloakclient")

const (
	Ok                                  = "OK"
	Fail                                = "FAIL"
	keyCloakClientOperatorFinalizerName = "keycloak.client.operator.finalizer.name"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new KeycloakClient Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKeycloakClient{
		client:  mgr.GetClient(),
		scheme:  mgr.GetScheme(),
		factory: new(adapter.GoCloakAdapterFactory),
		helper:  helper.MakeHelper(mgr.GetClient(), mgr.GetScheme()),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("keycloakclient-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KeycloakClient
	return c.Watch(&source.Kind{Type: &v1v1alpha1.KeycloakClient{}}, &handler.EnqueueRequestForObject{})
}

// blank assignment to verify that ReconcileKeycloakClient implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKeycloakClient{}

// ReconcileKeycloakClient reconciles a KeycloakClient object
type ReconcileKeycloakClient struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	scheme  *runtime.Scheme
	factory keycloak.ClientFactory
	helper  *helper.Helper
}

// Reconcile reads that state of the cluster for a KeycloakClient object and makes changes based on the state read
// and what is in the KeycloakClient.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKeycloakClient) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KeycloakClient")

	// Fetch the KeycloakClient instance
	instance := &v1v1alpha1.KeycloakClient{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	defer r.updateStatus(instance)

	err = r.tryReconcile(instance)
	r.setStatus(err, instance)

	return reconcile.Result{}, err
}

func (r *ReconcileKeycloakClient) setStatus(err error, instance *v1v1alpha1.KeycloakClient) {
	if err != nil {
		instance.Status.Value = Fail
		return
	}
	instance.Status.Value = Ok
}

func (r *ReconcileKeycloakClient) updateStatus(kc *v1v1alpha1.KeycloakClient) {
	err := r.client.Status().Update(context.TODO(), kc)
	if err != nil {
		_ = r.client.Update(context.TODO(), kc)
	}
}

func (r *ReconcileKeycloakClient) tryReconcile(keycloakClient *v1v1alpha1.KeycloakClient) error {
	realm, err := r.helper.GetOrCreateRealmOwnerRef(keycloakClient, keycloakClient.ObjectMeta)
	if err != nil {
		return pkgErrors.Wrap(err, "unable to GetOrCreateRealmOwnerRef")
	}

	if err = r.addTargetRealmIfNeed(keycloakClient, realm.Spec.RealmName); err != nil {
		return pkgErrors.Wrap(err, "unable to addTargetRealmIfNeed")
	}

	kClient, err := r.helper.CreateKeycloakClient(realm, r.factory)
	if err != nil {
		return pkgErrors.Wrap(err, "unable to CreateKeycloakClient")
	}

	id, err := r.putKeycloakClient(keycloakClient, kClient)
	if err != nil {
		return pkgErrors.Wrap(err, "unable to putKeycloakClient")
	}
	keycloakClient.Status.Id = *id

	if err := r.putKeycloakClientRole(keycloakClient, kClient); err != nil {
		return pkgErrors.Wrap(err, "unable to putKeycloakClientRole")
	}

	if err := r.putRealmRoles(realm, keycloakClient, kClient); err != nil {
		return pkgErrors.Wrap(err, "unable to putRealmRoles")
	}

	if err := r.putClientScope(realm.Spec.RealmName, keycloakClient, kClient); err != nil {
		return pkgErrors.Wrap(err, "unable to put client scope")
	}

	if err := r.putProtocolMappers(keycloakClient, kClient); err != nil {
		return pkgErrors.Wrap(err, "unable to putProtocolMappers")
	}

	if _, err := r.helper.TryToDelete(keycloakClient, makeTerminator(*id, keycloakClient.Spec.TargetRealm, kClient),
		keyCloakClientOperatorFinalizerName); err != nil {
		return pkgErrors.Wrap(err, "unable to delete kc client")
	}

	return nil
}

func copyMap(in map[string]string) map[string]string {
	out := make(map[string]string)
	for k, v := range in {
		out[k] = v
	}

	return out
}

func (r *ReconcileKeycloakClient) putProtocolMappers(kc *v1v1alpha1.KeycloakClient,
	kClient keycloak.Client) error {

	var protocolMappers []gocloak.ProtocolMapperRepresentation

	if kc.Spec.ProtocolMappers != nil {
		protocolMappers = make([]gocloak.ProtocolMapperRepresentation, 0, len(*kc.Spec.ProtocolMappers))
		for _, mapper := range *kc.Spec.ProtocolMappers {
			configCopy := copyMap(mapper.Config)

			protocolMappers = append(protocolMappers, gocloak.ProtocolMapperRepresentation{
				Name:           gocloak.StringP(mapper.Name),
				Protocol:       gocloak.StringP(mapper.Protocol),
				ProtocolMapper: gocloak.StringP(mapper.ProtocolMapper),
				Config:         &configCopy,
			})
		}
	}

	if err := kClient.SyncClientProtocolMapper(dto.ConvertSpecToClient(kc.Spec, ""),
		protocolMappers); err != nil {
		return pkgErrors.Wrap(err, "unable to sync protocol mapper")
	}

	return nil
}

func (r *ReconcileKeycloakClient) putClientScope(realmName string, kc *v1v1alpha1.KeycloakClient,
	kClient keycloak.Client) error {
	if !kc.Spec.AudRequired {
		return nil
	}
	scope, err := kClient.GetClientScope(consts.DefaultClientScopeName, realmName)
	if err != nil {
		return pkgErrors.Wrap(err, "error during GetClientScope")
	}
	if err := kClient.PutClientScopeMapper(kc.Spec.ClientId, *scope.ID, realmName); err != nil {
		return pkgErrors.Wrap(err, "error during PutClientScopeMapper")
	}
	if err := kClient.LinkClientScopeToClient(kc.Spec.ClientId, *scope.ID, realmName); err != nil {
		return pkgErrors.Wrap(err, "error during LinkClientScopeToClient")
	}

	return nil
}

func (r *ReconcileKeycloakClient) addTargetRealmIfNeed(keycloakClient *v1v1alpha1.KeycloakClient,
	mainRealm string) error {
	if keycloakClient.Spec.TargetRealm == "" {
		keycloakClient.Spec.TargetRealm = mainRealm
	}
	return r.client.Update(context.TODO(), keycloakClient)
}

func (r *ReconcileKeycloakClient) putKeycloakClient(keycloakClient *v1v1alpha1.KeycloakClient,
	kClient keycloak.Client) (*string, error) {
	reqLog := log.WithValues("keycloak client cr", keycloakClient)
	reqLog.Info("Start put keycloak client...")

	clientDto, err := r.convertCrToDto(keycloakClient)
	if err != nil {
		return nil, pkgErrors.Wrap(err, "error during convertCrToDto")
	}

	exist, err := kClient.ExistClient(*clientDto)
	if err != nil {
		return nil, pkgErrors.Wrap(err, "error during ExistClient")
	}

	if *exist {
		reqLog.Info("Client already exists")
		return kClient.GetClientId(*clientDto)
	}

	err = kClient.CreateClient(*clientDto)
	if err != nil {
		return nil, pkgErrors.Wrap(err, "error during CreateClient")
	}

	reqLog.Info("End put keycloak client")
	return kClient.GetClientId(*clientDto)
}

func (r *ReconcileKeycloakClient) putKeycloakClientRole(keycloakClient *v1v1alpha1.KeycloakClient,
	kClient keycloak.Client) error {
	reqLog := log.WithValues("keycloak client cr", keycloakClient)
	reqLog.Info("Start put keycloak client role...")

	clientDto := dto.ConvertSpecToClient(keycloakClient.Spec, "")

	for _, role := range clientDto.Roles {
		exist, err := kClient.ExistClientRole(clientDto, role)
		if err != nil {
			return err
		}

		if *exist {
			reqLog.Info("Client role already exists", "role", role)
			continue
		}

		err = kClient.CreateClientRole(clientDto, role)
		if err != nil {
			return err
		}
	}

	reqLog.Info("End put keycloak client role")
	return nil
}

func (r *ReconcileKeycloakClient) putRealmRoles(
	realm *v1v1alpha1.KeycloakRealm, keycloakClient *v1v1alpha1.KeycloakClient, kClient keycloak.Client) error {
	reqLog := log.WithValues("keycloak client cr", keycloakClient)
	reqLog.Info("Start put realm roles...")

	if keycloakClient.Spec.RealmRoles == nil || len(*keycloakClient.Spec.RealmRoles) == 0 {
		reqLog.Info("Keycloak client does not have realm roles")
		return nil
	}

	realmDto := dto.ConvertSpecToRealm(realm.Spec)

	for _, el := range *keycloakClient.Spec.RealmRoles {
		roleDto := dto.RealmRole{
			Name:        el.Name,
			Composites:  []string{el.Composite},
			IsComposite: el.Composite != "",
		}
		exist, err := kClient.ExistRealmRole(realmDto, roleDto)
		if err != nil {
			return err
		}
		if *exist {
			reqLog.Info("Client already exists")
			return nil
		}
		err = kClient.CreateRealmRole(realmDto, roleDto)
		if err != nil {
			return err
		}
	}

	reqLog.Info("End put realm roles")
	return nil
}

func (r *ReconcileKeycloakClient) convertCrToDto(keycloakClient *v1v1alpha1.KeycloakClient) (*dto.Client, error) {
	if keycloakClient.Spec.Public {
		res := dto.ConvertSpecToClient(keycloakClient.Spec, "")
		return &res, nil
	}
	clientSecret := &coreV1.Secret{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Name:      keycloakClient.Spec.Secret,
		Namespace: keycloakClient.Namespace,
	}, clientSecret)
	if err != nil {
		return nil, err
	}
	clientSecretVal := string(clientSecret.Data["clientSecret"])

	res := dto.ConvertSpecToClient(keycloakClient.Spec, clientSecretVal)
	return &res, nil
}
