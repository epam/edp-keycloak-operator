package helper

import (
	"context"
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

const DefaultRequeueTime = 120 * time.Second

type Helper struct {
	client client.Client
	scheme *runtime.Scheme
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

func (h *Helper) getKeycloakMainRealm(object v1.Object) (*v1alpha1.KeycloakRealm, error) {
	var realm v1alpha1.KeycloakRealm
	if err := h.client.Get(context.TODO(), types.NamespacedName{
		Name:      "main",
		Namespace: object.GetNamespace(),
	}, &realm); err != nil {
		return nil, errors.Wrap(err, "unable to get main realm from k8s")
	}

	return &realm, controllerutil.SetControllerReference(&realm, object, h.scheme)
}

func (h *Helper) GetOrCreateRealmOwnerRef(
	object v1.Object, objectMeta v1.ObjectMeta) (*v1alpha1.KeycloakRealm, error) {
	realm, err := h.GetOwnerKeycloakRealm(objectMeta)
	if err != nil {
		switch errors.Cause(err).(type) {
		case ErrOwnerNotFound:
			realm, err = h.getKeycloakMainRealm(object)
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
