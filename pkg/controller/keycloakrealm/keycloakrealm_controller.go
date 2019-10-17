package keycloakrealm

import (
	"context"
	"fmt"
	v1v1alpha1 "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
	"github.com/google/uuid"
	coreerrors "github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_keycloakrealm")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new KeycloakRealm Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKeycloakRealm{
		client:  mgr.GetClient(),
		scheme:  mgr.GetScheme(),
		factory: new(adapter.GoCloakAdapterFactory),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("keycloakrealm-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KeycloakRealm
	return c.Watch(&source.Kind{Type: &v1v1alpha1.KeycloakRealm{}}, &handler.EnqueueRequestForObject{})
}

// blank assignment to verify that ReconcileKeycloakRealm implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKeycloakRealm{}

// ReconcileKeycloakRealm reconciles a KeycloakRealm object
type ReconcileKeycloakRealm struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	scheme  *runtime.Scheme
	factory keycloak.ClientFactory
}

// Reconcile reads that state of the cluster for a KeycloakRealm object and makes changes based on the state read
// and what is in the KeycloakRealm.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKeycloakRealm) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KeycloakRealm")

	// Fetch the KeycloakRealm instance
	instance := &v1v1alpha1.KeycloakRealm{}
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

	err = r.tryReconcile(instance)
	instance.Status.Available = err == nil

	err = r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		err := r.client.Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Info(fmt.Sprintf("Couldn't update status for Keycloak Realm %s", instance.Name))
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, err
}

var (
	keycloakClientSecretTemplate = "keycloak-client.%s.secret"
	annotationKey                = "openid-configuration"
)

func (r *ReconcileKeycloakRealm) tryReconcile(realm *v1v1alpha1.KeycloakRealm) error {
	ownerKeycloak, err := helper.GetOwnerKeycloak(r.client, realm.ObjectMeta)
	if err != nil {
		return err
	}
	if ownerKeycloak == nil {
		return fmt.Errorf("cannot find owner keycloak for realm with name %s", realm.Name)
	}

	if !ownerKeycloak.Status.Connected {
		return coreerrors.New("Owner keycloak is not in connected status")
	}

	secret := &coreV1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{
		Name:      ownerKeycloak.Spec.Secret,
		Namespace: ownerKeycloak.Namespace,
	}, secret)
	if err != nil {
		return err
	}
	user := string(secret.Data["username"])
	pwd := string(secret.Data["password"])

	kClient, err := r.factory.New(dto.ConvertSpecToKeycloak(ownerKeycloak.Spec, user, pwd))
	if err != nil {
		return err
	}

	realmDto := dto.ConvertSpecToRealm(realm.Spec)

	err = generatePasswordsForAc(&realmDto)
	if err != nil {
		return err
	}

	err = r.putAcSecrets(realm, realmDto.ACCreatorPass, realmDto.ACReaderPass)

	if err != nil {
		return err
	}

	err = r.putRealm(realmDto, kClient)

	if err != nil {
		return err
	}

	err = r.putKeycloakClientCR(realm)

	if err != nil {
		return err
	}

	err = r.putKeycloakClientSecret(realm)

	if err != nil {
		return err
	}

	err = r.putRealmUsers(realm, kClient)
	if err != nil {
		return err
	}

	err = r.putRoleToUser(realm, kClient)
	if err != nil {
		return err
	}

	err = r.putOpenIdConfigAnnotation(realm, kClient)
	if err != nil {
		return err
	}

	return r.putIdentityProvider(realm, kClient)
}

func (r *ReconcileKeycloakRealm) putAcSecrets(realm *v1v1alpha1.KeycloakRealm,
	creatorPass string, readerPass string) error {

	err := r.putSecret(realm.Namespace, adapter.AcCreatorUsername, creatorPass)
	if err != nil {
		return err
	}

	return r.putSecret(realm.Namespace, adapter.AcReaderUsername, readerPass)
}

func (r *ReconcileKeycloakRealm) putSecret(namespace, secretName, pass string) error {
	secret, err := r.getSecret(types.NamespacedName{
		Name:      secretName,
		Namespace: namespace,
	})
	if err != nil {
		return err
	}
	if secret != nil {
		log.Info("Secret already exists. Skip adding", "namespace", namespace, "secret name", secretName)
		return nil
	}
	secret = &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      secretName,
		}, Data: map[string][]byte{
			"username": []byte(secretName),
			"password": []byte(pass),
		},
	}
	return r.client.Create(context.TODO(), secret)
}

func generatePasswordsForAc(realm *dto.Realm) error {
	res, err := password.Generate(13, 7, 0, false, false)
	if err != nil {
		return err
	}
	realm.ACReaderPass = res
	res, err = password.Generate(13, 7, 0, false, false)
	if err != nil {
		return err
	}
	realm.ACCreatorPass = res
	return nil
}

func (r *ReconcileKeycloakRealm) putRealm(realmDto dto.Realm, kClient keycloak.Client) error {
	reqLog := log.WithValues("realm dto", realmDto)
	reqLog.Info("Start putting realm")

	exist, err := kClient.ExistRealm(realmDto)
	if err != nil {
		return err
	}
	if *exist {
		log.Info("Realm already exists")
		return nil
	}
	err = kClient.CreateRealmWithDefaultConfig(realmDto)
	if err != nil {
		return coreerrors.Wrap(err, "Cannot create realm")
	}
	return nil
}

func (r *ReconcileKeycloakRealm) putRealmUsers(realm *v1v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	reqLog := log.WithValues("keycloak users", realm.Spec.Users)
	reqLog.Info("Start putting users to realm")

	userDto := dto.ConvertSpecToRealm(realm.Spec)

	for _, user := range userDto.Users {
		exist, err := kClient.ExistRealmUser(realm.Spec.SsoRealmName, user)
		if err != nil {
			return err
		}

		if *exist {
			log.Info("User already exists", "user", user)
			continue
		}

		err = kClient.CreateRealmUser(realm.Spec.SsoRealmName, user)
		if err != nil {
			return err
		}
	}

	reqLog.Info("End put users to realm")
	return nil
}

func (r *ReconcileKeycloakRealm) putRoleToUser(realm *v1v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	reqLog := log.WithValues("keycloak users", realm.Spec.Users)
	reqLog.Info("Start putting users to realm")

	userDto := dto.ConvertSpecToRealm(realm.Spec)

	for _, user := range userDto.Users {
		for _, role := range user.RealmRoles {
			exist, err := kClient.HasUserClientRole(realm.Spec.SsoRealmName, realm.Spec.RealmName, user, role)
			if err != nil {
				return err
			}

			if *exist {
				log.Info("Role already exists", "user", user, "role", role)
				continue
			}

			err = kClient.AddClientRoleToUser(realm.Spec.SsoRealmName, realm.Spec.RealmName, user, role)
			if err != nil {
				return err
			}
		}
	}

	reqLog.Info("End put role to users")
	return nil
}

func (r *ReconcileKeycloakRealm) putKeycloakClientCR(realm *v1v1alpha1.KeycloakRealm) error {
	reqLog := log.WithValues("realm name", realm.Spec.RealmName)
	reqLog.Info("Start creation of Keycloak client CR")

	instance, err := r.getKeycloakClientCR(realm.Spec.RealmName, realm.Namespace)

	if err != nil {
		return err
	}
	if instance != nil {
		reqLog.Info("Required Keycloak client CR already exists")
		return nil
	}
	instance = &v1v1alpha1.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      realm.Spec.RealmName,
			Namespace: realm.Namespace,
		},
		Spec: v1v1alpha1.KeycloakClientSpec{
			Secret:      fmt.Sprintf(keycloakClientSecretTemplate, realm.Spec.RealmName),
			TargetRealm: realm.Spec.SsoRealmName,
			ClientId:    realm.Spec.RealmName,
			ClientRoles: []string{"administrator", "developer"},
		},
	}
	err = controllerutil.SetControllerReference(realm, instance, r.scheme)
	if err != nil {
		return coreerrors.Wrap(err, "cannot set owner ref for keycloak client CR")
	}
	err = r.client.Create(context.TODO(), instance)
	if err != nil {
		return coreerrors.Wrap(err, "cannot create keycloak client cr")
	}
	reqLog.Info("Keycloak client has been successfully created", "keycloak client", instance)

	return nil
}

func (r *ReconcileKeycloakRealm) getKeycloakClientCR(name, namespace string) (*v1v1alpha1.KeycloakClient, error) {
	reqLog := log.WithValues("keycloak client name", name, "keycloak client namespace", namespace)
	reqLog.Info("Start retrieve keycloak client cr...")

	instance := &v1v1alpha1.KeycloakClient{}
	nsn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	err := r.client.Get(context.TODO(), nsn, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLog.Info("Keycloak Client CR has not been found")
			return nil, nil
		}
		return nil, coreerrors.Wrap(err, "cannot read keycloak client CR")
	}
	reqLog.Info("Keycloak client has bee retrieved", "keycloak client cr", instance)
	return instance, nil
}

func (r *ReconcileKeycloakRealm) putIdentityProvider(realm *v1v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	reqLog := log.WithValues("realm name", realm.Name, "realm namespace", realm.Namespace)
	reqLog.Info("Start put identity provider for realm...")

	keycloakClient, err := r.getKeycloakClientCR(realm.Spec.RealmName, realm.Namespace)
	if err != nil {
		return err
	}
	if keycloakClient == nil {
		return fmt.Errorf("required keycloak client cr with name %s does not exist in namespace %s",
			realm.Spec.RealmName, realm.Namespace)
	}
	realmDto := dto.ConvertSpecToRealm(realm.Spec)
	exist, err := kClient.ExistCentralIdentityProvider(realmDto)
	if err != nil {
		return err
	}

	if *exist {
		reqLog.Info("IdP already exists")
		return nil
	}

	secret, err := r.getSecret(types.NamespacedName{
		Name:      keycloakClient.Spec.Secret,
		Namespace: keycloakClient.Namespace,
	})
	if secret == nil {
		return coreerrors.Errorf("secret %s does not exist", secret.Name)
	}
	if err != nil {
		return err
	}

	err = kClient.CreateCentralIdentityProvider(realmDto, dto.Client{
		ClientId:     realm.Spec.RealmName,
		ClientSecret: string(secret.Data["clientSecret"]),
	})

	if err != nil {
		return err
	}

	reqLog.Info("End put identity provider for realm")
	return nil
}

func (r *ReconcileKeycloakRealm) putKeycloakClientSecret(realm *v1v1alpha1.KeycloakRealm) error {
	reqLog := log.WithValues("realm name", realm.Spec.RealmName)
	reqLog.Info("Start creation of Keycloak client secret")

	sn := fmt.Sprintf(keycloakClientSecretTemplate, realm.Spec.RealmName)
	secret, err := r.getSecret(types.NamespacedName{
		Name:      sn,
		Namespace: realm.Namespace,
	})
	if err != nil {
		return err
	}
	if secret != nil {
		reqLog.Info("Keycloak client secret already exist")
		return nil
	}
	secret = &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sn,
			Namespace: realm.Namespace,
		},
		Data: map[string][]byte{
			"clientSecret": []byte(uuid.New().String()),
		},
	}
	cl, err := r.getKeycloakClientCR(realm.Spec.RealmName, realm.Namespace)
	if cl == nil {
		return fmt.Errorf("required keycloak client %s does not exist", realm.Spec.RealmName)
	}
	err = controllerutil.SetControllerReference(cl, secret, r.scheme)
	if err != nil {
		return err
	}
	err = r.client.Create(context.TODO(), secret)

	if err != nil {
		return err
	}
	reqLog.Info("End of put Keycloak client secret")
	return nil
}

func (r *ReconcileKeycloakRealm) getSecret(nsn types.NamespacedName) (*coreV1.Secret, error) {
	secret := &coreV1.Secret{}
	err := r.client.Get(context.TODO(), nsn, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, coreerrors.Wrap(err, "cannot get secret")
	}
	return secret, nil
}

func (r *ReconcileKeycloakRealm) putOpenIdConfigAnnotation(realm *v1v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	reqLog := log.WithValues("realm spec", realm.Spec)
	reqLog.Info("Start put openid configuration annotation...")

	openIdConfig, err := kClient.GetOpenIdConfig(dto.ConvertSpecToRealm(realm.Spec))
	if err != nil {
		return err
	}
	an := realm.GetAnnotations()
	if an == nil {
		an = make(map[string]string)
	}
	an[annotationKey] = *openIdConfig
	realm.SetAnnotations(an)
	err = r.client.Update(context.TODO(), realm)
	if err != nil {
		return err
	}

	reqLog.Info("end put openid configuration annotation")
	return nil
}
