package helper

import (
	"context"
	"math"
	"time"

	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
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
	DefaultRequeueTime = 120 * time.Second
	StatusOK           = "OK"
)

type Helper struct {
	client client.Client
	scheme *runtime.Scheme
}

func (h *Helper) GetScheme() *runtime.Scheme {
	return h.scheme
}

func MakeHelper(client client.Client, scheme *runtime.Scheme) *Helper {
	return &Helper{
		client: client,
		scheme: scheme,
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

func (h *Helper) GetTimeout(factor int64, baseDuration time.Duration) time.Duration {
	return time.Duration(float64(baseDuration) * math.Pow(math.E, float64(factor+1)))
}

func (h *Helper) GetOwner(slave v1.ObjectMeta, owner runtime.Object, ownerType string) error {
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

func GetSecret(client client.Client, nsn types.NamespacedName) (*coreV1.Secret, error) {
	secret := &coreV1.Secret{}
	err := client.Get(context.TODO(), nsn, secret)
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
	K8SParentRealmName() string
	v1.Object
}

func (h *Helper) GetOrCreateRealmOwnerRef(
	object RealmChild, objectMeta v1.ObjectMeta) (*v1alpha1.KeycloakRealm, error) {
	realm, err := h.GetOwnerKeycloakRealm(objectMeta)
	if err != nil {
		switch errors.Cause(err).(type) {
		case ErrOwnerNotFound:
			realm, err = h.getKeycloakRealm(object, object.K8SParentRealmName())
			if err != nil {
				return nil, errors.Wrap(err, "unable to get keycloak from spec")
			}
		default:
			return nil, errors.Wrap(err, "unable to get owner keycloak")
		}
	}

	return realm, nil
}

func (h *Helper) CreateKeycloakClient(
	realm *v1alpha1.KeycloakRealm, factory keycloak.ClientFactory) (keycloak.Client, error) {

	o, err := h.GetOrCreateKeycloakOwnerRef(realm)
	if err != nil {
		return nil, err
	}

	if !o.Status.Connected {
		return nil, errors.New("Owner keycloak is not in connected status")
	}

	var secret coreV1.Secret
	if err = h.client.Get(context.TODO(), types.NamespacedName{
		Name:      o.Spec.Secret,
		Namespace: o.Namespace,
	}, &secret); err != nil {
		return nil, err
	}

	return factory.New(
		dto.ConvertSpecToKeycloak(o.Spec, string(secret.Data["username"]), string(secret.Data["password"])))
}

type FailureCountable interface {
	GetFailureCount() int64
	SetFailureCount(count int64)
}

func (h *Helper) SetFailureCount(fc FailureCountable) time.Duration {
	failures := fc.GetFailureCount()
	timeout := h.GetTimeout(failures, 10*time.Second)
	failures += 1
	fc.SetFailureCount(failures)

	return timeout
}

func (h *Helper) UpdateStatus(obj runtime.Object) error {
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
}

func (h *Helper) TryToDelete(obj Deletable, terminator Terminator, finalizer string) (isDeleted bool, resultErr error) {
	finalizers := obj.GetFinalizers()

	if obj.GetDeletionTimestamp().IsZero() {
		if !ContainsString(finalizers, finalizer) {
			finalizers = append(finalizers, finalizer)
			obj.SetFinalizers(finalizers)

			if err := h.client.Update(context.TODO(), obj); err != nil {
				return false, errors.Wrap(err, "unable to update deletable object")
			}
		}

		return false, nil
	}

	if err := terminator.DeleteResource(); err != nil {
		return false, errors.Wrap(err, "error during keycloak client delete func")
	}

	finalizers = RemoveString(finalizers, finalizer)
	obj.SetFinalizers(finalizers)

	if err := h.client.Update(context.TODO(), obj); err != nil {
		return false, errors.Wrap(err, "unable to update realm role cr")
	}

	return true, nil
}
