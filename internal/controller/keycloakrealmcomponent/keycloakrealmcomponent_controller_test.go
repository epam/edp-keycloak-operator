package keycloakrealmcomponent

import (
	"context"
	"errors"
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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	helpermock "github.com/epam/edp-keycloak-operator/internal/controller/helper/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

func TestReconcile_Reconcile(t *testing.T) {
	logger := mock.NewLogr()
	sch := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(sch))
	utilruntime.Must(corev1.AddToScheme(sch))

	var (
		comp = keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{Name: "test-comp-name", Namespace: "ns"},
			TypeMeta:   metav1.TypeMeta{Kind: "KeycloakRealmComponent", APIVersion: "v1.edp.epam.com/v1"},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name: "test-comp",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: "realm",
				},
			},
			Status: keycloakApi.KeycloakComponentStatus{Value: common.StatusOK},
		}
		realm = keycloakApi.KeycloakRealm{TypeMeta: metav1.TypeMeta{
			APIVersion: "v1.edp.epam.com/v1", Kind: "KeycloakRealm",
		},
			ObjectMeta: metav1.ObjectMeta{Name: "realm1", Namespace: "ns",
				OwnerReferences: []metav1.OwnerReference{{Name: "keycloak1", Kind: "Keycloak"}}},
			Spec: keycloakApi.KeycloakRealmSpec{RealmName: "realm11"}}
		testComp = adapter.Component{ID: "component-id1", Name: comp.Spec.Name, Config: make(map[string][]string)}
	)

	kcAdapter := mocks.NewMockClient(t)
	client := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&comp).WithStatusSubresource(&comp).Build()
	h := helpermock.NewControllerHelper(t)

	h.On("SetRealmOwnerRef", testifymock.Anything, testifymock.Anything).Return(nil)
	h.On("CreateKeycloakClientFromRealmRef", testifymock.Anything, testifymock.Anything).Return(kcAdapter, nil)
	h.On("TryToDelete", testifymock.Anything, testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(false, nil)
	h.On("GetKeycloakRealmFromRef", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(&gocloak.RealmRepresentation{
			Realm: gocloak.StringP(realm.Spec.RealmName),
		}, nil)
	kcAdapter.On("GetComponent", testifymock.Anything, realm.Spec.RealmName, comp.Spec.Name).Return(&testComp, nil).Once()
	kcAdapter.On("UpdateComponent", testifymock.Anything, realm.Spec.RealmName, &testComp).Return(nil)

	r := NewReconcile(client, sch, h, secretref.NewSecretRef(client))

	res, err := r.Reconcile(ctrl.LoggerInto(context.Background(), logger), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      comp.Name,
		Namespace: comp.Namespace,
	}})
	require.NoError(t, err)

	loggerSink, ok := logger.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")
	require.NoError(t, loggerSink.LastError())

	if res.RequeueAfter != r.successReconcileTimeout {
		t.Fatalf("wrong RequeueAfter: %d", res.RequeueAfter)
	}

	kcAdapter.On("GetComponent", testifymock.Anything, realm.Spec.RealmName, comp.Spec.Name).Return(nil,
		adapter.NotFoundError("not found")).Once()
	kcAdapter.On("CreateComponent", testifymock.Anything, realm.Spec.RealmName,
		testifymock.Anything).Return(errors.New("create fatal"))

	failureComp := comp.DeepCopy()
	failureComp.Status.Value = "unable to create component: create fatal"

	h.On("SetFailureCount", testifymock.Anything).Return(time.Minute)

	_, err = r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      comp.Name,
		Namespace: comp.Namespace,
	}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "create fatal")
}

func TestIsSpecUpdated(t *testing.T) {
	comp := keycloakApi.KeycloakRealmComponent{}

	if isSpecUpdated(event.UpdateEvent{ObjectNew: &comp, ObjectOld: &comp}) {
		t.Fatal("spec is updated")
	}
}
