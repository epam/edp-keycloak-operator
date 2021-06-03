package keycloakclient

import (
	"context"

	v1v1alpha1 "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileKeycloakClient) getOrCreateRealmOwner(
	keycloakClient *v1v1alpha1.KeycloakClient) (*v1v1alpha1.KeycloakRealm, error) {

	realm, err := r.helper.GetOrCreateRealmOwnerRef(&clientRealmFinder{parent: keycloakClient,
		client: r.client},
		keycloakClient.ObjectMeta)
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
	parent *v1v1alpha1.KeycloakClient

	v1.TypeMeta
	v1.ObjectMeta
}

func (c *clientRealmFinder) GetNamespace() string {
	return c.parent.Namespace
}

func (c *clientRealmFinder) K8SParentRealmName() (string, error) {
	var realmList v1v1alpha1.KeycloakRealmList
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
	for _, r := range realmList.Items {
		if r.Spec.RealmName == c.parent.Spec.TargetRealm {
			return r.Name, nil
		}
	}

	return "main", nil
}

func (c *clientRealmFinder) SetOwnerReferences(or []v1.OwnerReference) {
	c.parent.SetOwnerReferences(or)
}

func (r *ReconcileKeycloakClient) addTargetRealmIfNeed(keycloakClient *v1v1alpha1.KeycloakClient,
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
