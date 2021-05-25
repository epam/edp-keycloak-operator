package keycloakrealmgroup

import (
	"testing"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/apis"
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileKeycloakRealmGroup_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &corev1.Secret{})
	ns := "security"
	keycloak := v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "keycloak1", Namespace: ns}, Status: v1alpha1.KeycloakStatus{Connected: true}}
	realm := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "realm1", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "keycloak1", Kind: "Keycloak"}}},
		Spec: v1alpha1.KeycloakRealmSpec{RealmName: "ns.realm1"}}
	now := metav1.Time{Time: time.Now()}
	group := v1alpha1.KeycloakRealmGroup{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "group1", DeletionTimestamp: &now},
		Spec: v1alpha1.KeycloakRealmGroupSpec{Realm: "realm1", RealmRoles: []string{"role1", "role2"}, Name: "group1"}}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: ns},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}

	client := fake.NewFakeClientWithScheme(scheme, &group, &realm, &keycloak, &secret)
	kClient := new(mock.KeycloakClient)
	factory := new(mock.GoCloakFactory)
	factory.On("New", dto.Keycloak{User: "user", Pwd: "pass"}).
		Return(kClient, nil)

	kClient.On("SyncRealmGroup", "ns.realm1", &group.Spec).Return("gid1", nil)
	kClient.On("DeleteGroup", "ns.realm1", group.Spec.Name).Return(nil)

	r := ReconcileKeycloakRealmGroup{
		client:  client,
		helper:  helper.MakeHelper(client, scheme),
		scheme:  scheme,
		factory: factory,
	}

	if _, err := r.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: ns,
		Name:      "group1",
	}}); err != nil {
		t.Fatalf("%+v", err)
	}
}
