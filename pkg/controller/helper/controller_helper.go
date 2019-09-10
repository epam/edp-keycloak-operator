package helper

import (
	"context"
	"fmt"
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetOwnerKeycloak(client client.Client, slave v1.ObjectMeta) (*v1alpha1.Keycloak, error) {
	keycloak := &v1alpha1.Keycloak{}
	return keycloak, getOwner(client, slave, keycloak, "Keycloak")
}

func GetOwnerKeycloakRealm(client client.Client, slave v1.ObjectMeta) (*v1alpha1.KeycloakRealm, error) {
	realm := &v1alpha1.KeycloakRealm{}
	return realm, getOwner(client, slave, realm, "KeycloakRealm")
}

func getOwner(client client.Client, slave v1.ObjectMeta, owner runtime.Object, ownerType string) error {
	ownerRefs := slave.GetOwnerReferences()
	if len(ownerRefs) == 0 {
		return errors.New("resource does not have owner references")
	}
	ownerRef := getOwnerRef(ownerRefs, ownerType)
	if ownerRef == nil {
		return fmt.Errorf("resource does not hava owner reference with type: %s", ownerType)
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
