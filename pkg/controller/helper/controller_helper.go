package helper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/util"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	DefaultRequeueTime         = 120 * time.Second
	StatusOK                   = "OK"
	defaultConfigsAbsolutePath = "/usr/local/configs"
	localConfigsRelativePath   = "configs"
)

type adapterBuilder func(ctx context.Context, url, user, password, adminType string, log logr.Logger,
	restyClient *resty.Client) (keycloak.Client, error)

type Helper struct {
	client         client.Client
	scheme         *runtime.Scheme
	restyClient    *resty.Client
	logger         logr.Logger
	adapterBuilder adapterBuilder
}

func (h *Helper) GetScheme() *runtime.Scheme {
	return h.scheme
}

func MakeHelper(client client.Client, scheme *runtime.Scheme, logger logr.Logger) *Helper {
	return &Helper{
		client: client,
		scheme: scheme,
		logger: logger,
		adapterBuilder: func(ctx context.Context, url, user, password, adminType string, log logr.Logger,
			restyClient *resty.Client) (keycloak.Client, error) {
			if adminType == v1alpha1.KeycloakAdminTypeServiceAccount {
				return adapter.MakeFromServiceAccount(ctx, url, user, password, "master", log, restyClient)
			}

			return adapter.Make(ctx, url, user, password, log, restyClient)
		},
	}
}

type ErrOwnerNotFound string

func (e ErrOwnerNotFound) Error() string {
	return string(e)
}

func (h *Helper) GetOwnerKeycloak(slave v1.ObjectMeta) (*v1alpha1.Keycloak, error) {
	var kc v1alpha1.Keycloak
	if err := h.GetOwner(slave, &kc, "Keycloak"); err != nil {
		return nil, errors.Wrap(err, "unable to get keycloak owner")
	}

	return &kc, nil
}

func (h *Helper) GetOwnerKeycloakRealm(slave v1.ObjectMeta) (*v1alpha1.KeycloakRealm, error) {
	var realm v1alpha1.KeycloakRealm
	if err := h.GetOwner(slave, &realm, "KeycloakRealm"); err != nil {
		return nil, errors.Wrap(err, "unable to get keycloak realm owner")
	}

	return &realm, nil
}

func (h *Helper) IsOwner(slave client.Object, master client.Object) bool {
	for _, ref := range slave.GetOwnerReferences() {
		if ref.UID == master.GetUID() {
			return true
		}
	}

	return false
}

func (h *Helper) GetOwner(slave v1.ObjectMeta, owner client.Object, ownerType string) error {
	ownerRefs := slave.GetOwnerReferences()
	if len(ownerRefs) == 0 {
		return ErrOwnerNotFound("owner not found")
	}

	ownerRef := getOwnerRef(ownerRefs, ownerType)
	if ownerRef == nil {
		return ErrOwnerNotFound("owner not found")
	}

	if err := h.client.Get(context.TODO(), types.NamespacedName{
		Namespace: slave.Namespace,
		Name:      ownerRef.Name,
	}, owner); err != nil {
		return errors.Wrap(err, "unable to get owner reference")
	}

	return nil
}

func getOwnerRef(references []v1.OwnerReference, typeName string) *v1.OwnerReference {
	for _, el := range references {
		if el.Kind == typeName {
			return &el
		}
	}

	return nil
}

func GetKeycloakClientCR(client client.Client, nsn types.NamespacedName) (*v1alpha1.KeycloakClient, error) {
	instance := &v1alpha1.KeycloakClient{}
	err := client.Get(context.TODO(), nsn, instance)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil //todo maybe refactor?
		}
		return nil, errors.Wrap(err, "cannot read keycloak client CR")
	}
	return instance, nil
}

func GetSecret(ctx context.Context, client client.Client, nsn types.NamespacedName) (*coreV1.Secret, error) {
	secret := &coreV1.Secret{}
	err := client.Get(ctx, nsn, secret)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil //todo maybe refactor?
		}
		return nil, errors.Wrap(err, "cannot get secret")
	}
	return secret, nil
}

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func (h *Helper) getKeycloakFromSpec(realm *v1alpha1.KeycloakRealm) (*v1alpha1.Keycloak, error) {
	if realm.Spec.KeycloakOwner == "" {
		return nil, errors.Errorf(
			"keycloak owner is not specified neither in ownerReference nor in spec for realm %s", realm.Name)
	}

	var k v1alpha1.Keycloak
	if err := h.client.Get(context.TODO(), types.NamespacedName{
		Namespace: realm.Namespace,
		Name:      realm.Spec.KeycloakOwner,
	}, &k); err != nil {
		return nil, errors.Wrap(err, "unable to get spec Keycloak from k8s")
	}

	return &k, controllerutil.SetControllerReference(&k, realm, h.scheme)
}

func (h *Helper) GetOrCreateKeycloakOwnerRef(realm *v1alpha1.KeycloakRealm) (*v1alpha1.Keycloak, error) {
	o, err := h.GetOwnerKeycloak(realm.ObjectMeta)
	if err != nil {
		switch errors.Cause(err).(type) {
		case ErrOwnerNotFound:
			o, err = h.getKeycloakFromSpec(realm)
			if err != nil {
				return nil, errors.Wrap(err, "unable to get keycloak from spec")
			}
		default:
			return nil, errors.Wrap(err, "unable to get owner keycloak")
		}
	}

	return o, nil
}

func (h *Helper) getKeycloakRealm(object v1.Object, name string) (*v1alpha1.KeycloakRealm, error) {
	var realm v1alpha1.KeycloakRealm
	if err := h.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: object.GetNamespace(),
	}, &realm); err != nil {
		return nil, errors.Wrap(err, "unable to get main realm from k8s")
	}

	return &realm, controllerutil.SetControllerReference(&realm, object, h.scheme)
}

type RealmChild interface {
	K8SParentRealmName() (string, error)
	v1.Object
}

func (h *Helper) GetOrCreateRealmOwnerRef(
	object RealmChild, objectMeta v1.ObjectMeta) (*v1alpha1.KeycloakRealm, error) {
	realm, err := h.GetOwnerKeycloakRealm(objectMeta)
	if err != nil {
		switch errors.Cause(err).(type) {
		case ErrOwnerNotFound:
			parentRealm, err := object.K8SParentRealmName()
			if err != nil {
				return nil, errors.Wrapf(err, "unable get parent realm for: %+v", object)
			}

			realm, err = h.getKeycloakRealm(object, parentRealm)
			if err != nil {
				return nil, errors.Wrap(err, "unable to get keycloak from spec")
			}
		default:
			return nil, errors.Wrap(err, "unable to get owner keycloak")
		}
	}

	return realm, nil
}

func (h *Helper) UpdateStatus(obj client.Object) error {
	if err := h.client.Status().Update(context.TODO(), obj); err != nil {
		return errors.Wrap(err, "unable to update object status")
	}

	return nil
}

type Deletable interface {
	v1.Object
	runtime.Object
}

type Terminator interface {
	DeleteResource() error
	GetLogger() logr.Logger
}

func (h *Helper) TryToDelete(ctx context.Context, obj Deletable, terminator Terminator, finalizer string) (isDeleted bool, resultErr error) {
	finalizers := obj.GetFinalizers()
	logger := terminator.GetLogger()

	if obj.GetDeletionTimestamp().IsZero() {
		logger.Info("instance timestamp is zero")

		if !ContainsString(finalizers, finalizer) {
			logger.Info("instance has not finalizers, adding...")
			finalizers = append(finalizers, finalizer)
			obj.SetFinalizers(finalizers)

			if err := h.client.Update(ctx, obj); err != nil {
				return false, errors.Wrap(err, "unable to update deletable object")
			}
		}

		logger.Info("processing finalizers done, exit.")
		return false, nil
	}

	logger.Info("terminator deleting resource")
	if err := terminator.DeleteResource(); err != nil {
		return false, errors.Wrap(err, "error during keycloak client delete func")
	}

	logger.Info("terminator removing finalizers")
	finalizers = RemoveString(finalizers, finalizer)
	obj.SetFinalizers(finalizers)

	if err := h.client.Update(ctx, obj); err != nil {
		return false, errors.Wrap(err, "unable to update realm role cr")
	}

	logger.Info("terminator deleting instance done, exit")
	return true, nil
}

func getExecutableFilePath() (string, error) {
	executableFilePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(executableFilePath), nil
}

func createPath(directory string, localRun bool) (string, error) {
	if localRun {
		executableFilePath, err := getExecutableFilePath()
		if err != nil {
			return "", errors.Wrapf(err, "Unable to get executable file path")
		}
		templatePath := fmt.Sprintf("%v/../%v/%v", executableFilePath, localConfigsRelativePath, directory)
		return templatePath, nil
	}

	templatePath := fmt.Sprintf("%s/%s", defaultConfigsAbsolutePath, directory)
	return templatePath, nil

}

func checkIfRunningLocally() bool {
	return !util.RunningInCluster()
}

func CreatePathToTemplateDirectory(directory string) (string, error) {
	localRun := checkIfRunningLocally()
	return createPath(directory, localRun)
}
