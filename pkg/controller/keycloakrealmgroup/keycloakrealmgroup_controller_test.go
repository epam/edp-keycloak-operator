package keycloakrealmgroup

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
)

func TestReconcileKeycloakRealmGroup_Reconcile(t *testing.T) {
	sch := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(sch))
	utilruntime.Must(corev1.AddToScheme(sch))

	ns := "security"
	keycloak := keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "keycloak1", Namespace: ns}, Status: keycloakApi.KeycloakStatus{Connected: true},
		Spec: keycloakApi.KeycloakSpec{Secret: "keycloak-secret"}}
	realm := keycloakApi.KeycloakRealm{TypeMeta: metav1.TypeMeta{
		APIVersion: "v1.edp.epam.com/v1", Kind: "KeycloakRealm",
	},
		ObjectMeta: metav1.ObjectMeta{Name: "realm1", Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{{Name: "keycloak1", Kind: "Keycloak"}}},
		Spec: keycloakApi.KeycloakRealmSpec{RealmName: "ns.realm1"}}
	//now := metav1.Time{Time: time.Now()}
	group := keycloakApi.KeycloakRealmGroup{TypeMeta: metav1.TypeMeta{
		APIVersion: "v1.edp.epam.com/v1", Kind: "KeycloakRealmGroup",
	}, ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "group1" /*, DeletionTimestamp: &now*/},
		Spec:   keycloakApi.KeycloakRealmGroupSpec{Realm: "realm1", RealmRoles: []string{"role1", "role2"}, Name: "group1"},
		Status: keycloakApi.KeycloakRealmGroupStatus{ID: "id11", Value: helper.StatusOK}}
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
