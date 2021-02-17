package helper

import (
	"context"
	"time"

	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DefaultRequeueTime = 120 * time.Second

func GetOwnerKeycloak(client client.Client, slave v1.ObjectMeta) (*v1alpha1.Keycloak, error) {
	keycloak := &v1alpha1.Keycloak{}
	err := getOwner(client, slave, keycloak, "Keycloak")
	if keycloak.Name == "" {
		return nil, err
	}
	return keycloak, err
}

func GetOwnerKeycloakRealm(client client.Client, slave v1.ObjectMeta) (*v1alpha1.KeycloakRealm, error) {
	realm := &v1alpha1.KeycloakRealm{}
	err := getOwner(client, slave, realm, "KeycloakRealm")
	if realm.Name == "" {
		return nil, err
	}
	return realm, err
}

func getOwner(client client.Client, slave v1.ObjectMeta, owner runtime.Object, ownerType string) error {
	ownerRefs := slave.GetOwnerReferences()
	if len(ownerRefs) == 0 {
		return nil
	}
	ownerRef := getOwnerRef(ownerRefs, ownerType)
	if ownerRef == nil {
		return nil
	}
	nsn := types.NamespacedName{
		Namespace: slave.Namespace,
		Name:      ownerRef.Name,
	}
	return client.Get(context.TODO(), nsn, owner)
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
			return nil, nil
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
			return nil, nil
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
