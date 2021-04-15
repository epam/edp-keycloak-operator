package keycloak

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	keycloakApi "github.com/epam/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/keycloak-operator/pkg/client/keycloak"
	"github.com/epam/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/keycloak-operator/pkg/controller/helper"
	"github.com/go-logr/logr"
	pkgErrors "github.com/pkg/errors"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	defaultRealmName = "openshift"
	imgFolder        = "img"
	keycloakIcon     = "keycloak.svg"
)

// ReconcileKeycloak reconciles a Keycloak object
type ReconcileKeycloak struct {
	Client  client.Client
	Scheme  *runtime.Scheme
	Log     logr.Logger
	Factory keycloak.ClientFactory
}

func (r *ReconcileKeycloak) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.Keycloak{}, builder.WithPredicates(pred)).
		Complete(r)
}

func (r *ReconcileKeycloak) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling Keycloak")

	instance := &keycloakApi.Keycloak{}
	if err := r.Client.Get(ctx, request.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if err := r.updateConnectionStatusToKeycloak(ctx, instance); err != nil {
		return reconcile.Result{}, err
	}

	con, err := r.isStatusConnected(ctx, request)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !con {
		log.Info("Status is not connected")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime}, nil
	}

	if err := r.putMainRealm(ctx, instance); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.putEDPComponent(ctx, instance); err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Reconciling Keycloak has been finished")
	return reconcile.Result{}, err
}

func (r *ReconcileKeycloak) updateConnectionStatusToKeycloak(ctx context.Context, instance *keycloakApi.Keycloak) error {
	log := r.Log.WithValues("keycloak cr", instance)
	log.Info("Start updating connection status to Keycloak")

	secret := &v1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name:      instance.Spec.Secret,
		Namespace: instance.Namespace,
	}, secret)
	if err != nil {
		return pkgErrors.Wrapf(err, "unable to get secret: %s", instance.Spec.Secret)
	}
	user := string(secret.Data["username"])
	pwd := string(secret.Data["password"])

	_, err = r.Factory.New(dto.ConvertSpecToKeycloak(instance.Spec, user, pwd))
	if err != nil {
		log.Error(err, "error during the creation of connection")
	}
	instance.Status.Connected = err == nil
	err = r.Client.Status().Update(ctx, instance)
	if err != nil {
		log.Error(err, "unable to update keycloak cr status")
		err := r.Client.Update(ctx, instance)
		if err != nil {
			log.Info(fmt.Sprintf("Couldn't update status for Keycloak %s", instance.Name))
			return err
		}
	}
	log.Info("Status has been updated", "status", instance.Status)
	return nil
}

func (r *ReconcileKeycloak) putMainRealm(ctx context.Context, instance *keycloakApi.Keycloak) error {
	log := r.Log.WithValues("keycloak cr", instance)
	log.Info("Start put main realm into k8s")
	if !instance.Spec.GetInstallMainRealm() {
		log.Info("Creation of main realm disabled")
		return nil
	}
	nsn := types.NamespacedName{
		Name:      "main",
		Namespace: instance.Namespace,
	}
	realmCr := &keycloakApi.KeycloakRealm{}
	err := r.Client.Get(ctx, nsn, realmCr)
	log.Info("Realm has been retrieved from k8s", "realmCr", realmCr)
	if errors.IsNotFound(err) {
		return r.createMainRealm(ctx, instance)
	}
	return err
}

func (r *ReconcileKeycloak) createMainRealm(ctx context.Context, instance *keycloakApi.Keycloak) error {
	log := r.Log.WithValues("keycloak cr", instance)
	log.Info("Start creation of main Keycloak Realm CR")
	ssoRealm := defaultRealmName
	if len(instance.Spec.SsoRealmName) != 0 {
		ssoRealm = instance.Spec.SsoRealmName
	}

	realmCr := &keycloakApi.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main",
			Namespace: instance.Namespace,
		},
		Spec: keycloakApi.KeycloakRealmSpec{
			KeycloakOwner: instance.Name,
			RealmName:     fmt.Sprintf("%s-%s", instance.Namespace, "main"),
			Users:         instance.Spec.Users,
			SsoRealmName:  ssoRealm,
		},
	}

	err := controllerutil.SetControllerReference(instance, realmCr, r.Scheme)
	if err != nil {
		return pkgErrors.Wrapf(err, "unable to update ControllerReference of main realm, realm: %+v", realmCr)
	}

	if err := r.Client.Create(ctx, realmCr); err != nil {
		return pkgErrors.Wrapf(err, "unable to create main realm cr: %+v", realmCr)
	}

	log.Info("Keycloak Realm CR has been created", "keycloak realm", realmCr)
	return nil
}

func (r *ReconcileKeycloak) isStatusConnected(ctx context.Context, request reconcile.Request) (bool, error) {
	r.Log.Info("Check is status of CR is connected", "request", request)
	instance := &keycloakApi.Keycloak{}
	err := r.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		return false, pkgErrors.Wrapf(err, "unable to get keycloak instance, request: %+v", request)
	}
	r.Log.Info("Retrieved the actual cr for Keycloak", "keycloak cr", instance)
	return instance.Status.Connected, nil
}

func (r *ReconcileKeycloak) putEDPComponent(ctx context.Context, instance *keycloakApi.Keycloak) error {
	log := r.Log.WithValues("instance", instance)
	log.Info("Start put edp component")
	nsn := types.NamespacedName{
		Name:      fmt.Sprintf("%v-keycloak", instance.Name),
		Namespace: instance.Namespace,
	}
	comp := &edpCompApi.EDPComponent{}
	err := r.Client.Get(ctx, nsn, comp)
	if err == nil {
		log.V(1).Info("EDP Component has been retrieved from k8s", "edp component", comp)
		return nil
	}
	if errors.IsNotFound(err) {
		return r.createEDPComponent(ctx, instance)
	}

	return pkgErrors.Wrapf(err, "unable to get edp component")
}

func (r *ReconcileKeycloak) createEDPComponent(ctx context.Context, instance *keycloakApi.Keycloak) error {
	log := r.Log.WithValues("instance", instance)
	log.Info("Start creation of EDP Component for Keycloak")

	icon, err := getIcon()
	if err != nil {
		return pkgErrors.Wrapf(err, "unable to get icon for instance: %+v", instance)
	}

	comp := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-keycloak", instance.Name),
			Namespace: instance.Namespace,
		},
		Spec: edpCompApi.EDPComponentSpec{
			Type:    "keycloak",
			Url:     fmt.Sprintf("%v/%v", instance.Spec.Url, "auth"),
			Icon:    *icon,
			Visible: true,
		},
	}

	err = controllerutil.SetControllerReference(instance, comp, r.Scheme)
	if err != nil {
		return pkgErrors.Wrapf(err, "unable to set controller reference for component: %+v", comp)
	}
	err = r.Client.Create(ctx, comp)
	if err != nil {
		return pkgErrors.Wrapf(err, "unable to create component: %+v", comp)
	}
	log.Info("EDP component has been created", "edp component", comp)
	return nil
}

func getIcon() (*string, error) {
	p, err := helper.CreatePathToTemplateDirectory(imgFolder)
	if err != nil {
		return nil, pkgErrors.Wrapf(err, "unable to create path to template dir: %s", imgFolder)
	}
	fp := fmt.Sprintf("%v/%v", p, keycloakIcon)
	f, err := os.Open(fp)
	if err != nil {
		return nil, pkgErrors.Wrapf(err, "unable to open file: %s", fp)
	}
	reader := bufio.NewReader(f)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, pkgErrors.Wrapf(err, "unable to read content of file: %s", fp)
	}
	encoded := base64.StdEncoding.EncodeToString(content)
	return &encoded, nil
}
