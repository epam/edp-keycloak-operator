package keycloakrealmgroup

import (
	"context"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v12"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	helpermock "github.com/epam/edp-keycloak-operator/controllers/helper/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
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
	group := keycloakApi.KeycloakRealmGroup{TypeMeta: metav1.TypeMeta{
		APIVersion: "v1.edp.epam.com/v1", Kind: "KeycloakRealmGroup",
	}, ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "group1" /*, DeletionTimestamp: &now*/},
		Spec: keycloakApi.KeycloakRealmGroupSpec{
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: realm.Name,
			},
			RealmRoles: []string{"role1", "role2"}, Name: "group1"},
		Status: keycloakApi.KeycloakRealmGroupStatus{ID: "id11", Value: helper.StatusOK}}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: ns},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}

	client := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&group, &realm, &keycloak, &secret).Build()
	kClient := new(adapter.Mock)

	kClient.On("SyncRealmGroup", "ns.realm1", &group.Spec).Return("gid1", nil)
	kClient.On("DeleteGroup", "ns.realm1", group.Spec.Name).Return(nil)

	logger := mock.NewLogr()
	h := helpermock.NewControllerHelper(t)
	kcMock := adapter.Mock{}

	h.On("SetRealmOwnerRef", testifymock.Anything, testifymock.Anything).Return(nil)
	h.On("CreateKeycloakClientFromRealmRef", testifymock.Anything, testifymock.Anything).Return(&kcMock, nil)
	h.On("GetKeycloakRealmFromRef", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(&gocloak.RealmRepresentation{
			Realm: gocloak.StringP(realm.Spec.RealmName),
		}, nil)
	h.On("TryToDelete", testifymock.Anything, testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(false, nil)
	kcMock.On("SyncRealmGroup", "ns.realm1", &group.Spec).Return("id11", nil)

	r := ReconcileKeycloakRealmGroup{
		client:                  client,
		helper:                  h,
		successReconcileTimeout: time.Hour,
	}

	res, err := r.Reconcile(ctrl.LoggerInto(context.Background(), logger), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: ns,
		Name:      "group1",
	}})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	loggerSink, ok := logger.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")
	require.NoError(t, loggerSink.LastError())

	if res.RequeueAfter != r.successReconcileTimeout {
		t.Fatal("success reconcile timeout is not set")
	}
}
