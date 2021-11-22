package keycloakrealmgroup

import (
	"context"
	"testing"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileKeycloakRealmGroup_Reconcile(t *testing.T) {
	sch := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(sch))
	utilruntime.Must(corev1.AddToScheme(sch))

	ns := "security"
	keycloak := v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "keycloak1", Namespace: ns}, Status: v1alpha1.KeycloakStatus{Connected: true},
		Spec: v1alpha1.KeycloakSpec{Secret: "keycloak-secret"}}
	realm := v1alpha1.KeycloakRealm{TypeMeta: metav1.TypeMeta{
		APIVersion: "v1.edp.epam.com/v1alpha1", Kind: "KeycloakRealm",
	},
		ObjectMeta: metav1.ObjectMeta{Name: "realm1", Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{{Name: "keycloak1", Kind: "Keycloak"}}},
		Spec: v1alpha1.KeycloakRealmSpec{RealmName: "ns.realm1"}}
	//now := metav1.Time{Time: time.Now()}
	group := v1alpha1.KeycloakRealmGroup{TypeMeta: metav1.TypeMeta{
		APIVersion: "v1.edp.epam.com/v1alpha1", Kind: "KeycloakRealmGroup",
	}, ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "group1" /*, DeletionTimestamp: &now*/},
		Spec:   v1alpha1.KeycloakRealmGroupSpec{Realm: "realm1", RealmRoles: []string{"role1", "role2"}, Name: "group1"},
		Status: v1alpha1.KeycloakRealmGroupStatus{ID: "id11", Value: helper.StatusOK}}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: ns},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}

	client := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&group, &realm, &keycloak, &secret).Build()
	kClient := new(adapter.Mock)

	kClient.On("SyncRealmGroup", "ns.realm1", &group.Spec).Return("gid1", nil)
	kClient.On("DeleteGroup", "ns.realm1", group.Spec.Name).Return(nil)

	logger := mock.Logger{}
	h := helper.Mock{}
	kcMock := adapter.Mock{}

	h.On("GetOrCreateRealmOwnerRef", &group, group.ObjectMeta).Return(&realm, nil)
	h.On("CreateKeycloakClientForRealm", &realm).Return(&kcMock, nil)
	kcMock.On("SyncRealmGroup", "ns.realm1", &group.Spec).Return("id11", nil)
	h.On("TryToDelete", &group, makeTerminator(&kcMock, realm.Spec.RealmName, group.Spec.Name, &logger),
		keyCloakRealmGroupOperatorFinalizerName).Return(true, nil)
	h.On("UpdateStatus", &group).Return(nil)

	r := ReconcileKeycloakRealmGroup{
		client:                  client,
		helper:                  &h,
		log:                     &logger,
		successReconcileTimeout: time.Hour,
	}

	res, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: ns,
		Name:      "group1",
	}})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if err := logger.LastError(); err != nil {
		t.Fatalf("%+v", err)
	}

	if res.RequeueAfter != r.successReconcileTimeout {
		t.Fatal("success reconcile timeout is not set")
	}
}
