// +kubebuilder:skip

package keycloakclient

import (
	"context"

	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1/v1"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealm/chain"
)

func (r *ReconcileKeycloakClient) getOrCreateRealmOwner(keycloakClient *keycloakApi.KeycloakClient) (*keycloakApi.KeycloakRealm, error) {
	realm, err := r.helper.GetOrCreateRealmOwnerRef(&clientRealmFinder{parent: keycloakClient,
		client: r.client},
		&keycloakClient.ObjectMeta)
	if err != nil {
		return nil, errors.Wrap(err, "unable to GetOrCreateRealmOwnerRef")
	}

	if err = r.addTargetRealmIfNeed(keycloakClient, realm.Spec.RealmName); err != nil {
		return nil, errors.Wrap(err, "unable to addTargetRealmIfNeed")
	}

	return realm, nil
}

type clientRealmFinder struct {
	client client.Client
	parent *keycloakApi.KeycloakClient

	v1.TypeMeta
	//TODO: get rid of this field here or refactor struct. Controller-get tool treat clientRealmFinder as CRD
	v1.ObjectMeta
}

func (c *clientRealmFinder) GetNamespace() string {
	return c.parent.Namespace
}

func (c *clientRealmFinder) K8SParentRealmName() (string, error) {
	var realmList keycloakApi.KeycloakRealmList

	listOpts := client.ListOptions{Namespace: c.parent.Namespace}

	client.MatchingLabels(map[string]string{chain.TargetRealmLabel: c.parent.Spec.TargetRealm}).ApplyToList(&listOpts)

	if err := c.client.List(context.Background(), &realmList, &listOpts); err != nil {
		return "", errors.Wrap(err, "unable to get reams by label")
	}

	if len(realmList.Items) > 0 {
		return realmList.Items[0].Name, nil
	}

	if err := c.client.List(context.Background(), &realmList, &client.ListOptions{Namespace: c.Namespace}); err != nil {
		return "", errors.Wrap(err, "unable to get all reams")
	}

	for i := range realmList.Items {
		if realmList.Items[i].Spec.RealmName == c.parent.Spec.TargetRealm {
			return realmList.Items[i].Name, nil
		}
	}

	return "main", nil
}

func (c *clientRealmFinder) SetOwnerReferences(or []v1.OwnerReference) {
	c.parent.SetOwnerReferences(or)
}

func (r *ReconcileKeycloakClient) addTargetRealmIfNeed(keycloakClient *keycloakApi.KeycloakClient,
	reamName string) error {
	if keycloakClient.Spec.TargetRealm != "" {
		return nil
	}

	keycloakClient.Spec.TargetRealm = reamName
	if err := r.client.Update(context.TODO(), keycloakClient); err != nil {
		return errors.Wrap(err, "unable to set keycloak client target realm")
	}

	return nil
}
