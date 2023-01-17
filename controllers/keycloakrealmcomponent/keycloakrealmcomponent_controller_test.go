package keycloakrealmcomponent

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestReconcile_Reconcile(t *testing.T) {
	logger := mock.NewLogr()
	sch := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(sch))
	utilruntime.Must(corev1.AddToScheme(sch))

	var (
		hlp       helper.Mock
		kcAdapter adapter.Mock
		comp      = keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{Name: "test-comp-name", Namespace: "ns"},
			TypeMeta:   metav1.TypeMeta{Kind: "KeycloakRealmComponent", APIVersion: "v1.edp.epam.com/v1"},
			Spec:       keycloakApi.KeycloakComponentSpec{Name: "test-comp"},
			Status:     keycloakApi.KeycloakComponentStatus{Value: helper.StatusOK},
		}
		realm = keycloakApi.KeycloakRealm{TypeMeta: metav1.TypeMeta{
			APIVersion: "v1.edp.epam.com/v1", Kind: "KeycloakRealm",
		},
			ObjectMeta: metav1.ObjectMeta{Name: "realm1", Namespace: "ns",
				OwnerReferences: []metav1.OwnerReference{{Name: "keycloak1", Kind: "Keycloak"}}},
			Spec: keycloakApi.KeycloakRealmSpec{RealmName: "ns.realm1"}}
		testComp = adapter.Component{ID: "component-id1", Name: comp.Spec.Name}
	)

	client := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&comp).Build()
	hlp.On("GetOrCreateRealmOwnerRef", &comp, &comp.ObjectMeta).Return(&realm, nil)
	hlp.On("CreateKeycloakClientForRealm", &realm).Return(&kcAdapter, nil)
	kcAdapter.On("GetComponent", realm.Spec.RealmName, comp.Spec.Name).Return(&testComp, nil).Once()
	kcAdapter.On("UpdateComponent", realm.Spec.RealmName, &testComp).Return(nil)
	hlp.On("TryToDelete", &comp, makeTerminator(realm.Spec.RealmName, comp.Spec.Name, &kcAdapter, logger),
		finalizerName).Return(false, nil)
	hlp.On("UpdateStatus", &comp).Return(nil)
	r := NewReconcile(client, logger, &hlp)

	res, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
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

	kcAdapter.On("GetComponent", realm.Spec.RealmName, comp.Spec.Name).Return(nil,
		adapter.NotFoundError("not found")).Once()
	kcAdapter.On("CreateComponent", realm.Spec.RealmName,
		createKeycloakComponentFromSpec(&comp.Spec)).Return(errors.New("create fatal"))

	failureComp := comp.DeepCopy()
	failureComp.Status.Value = "unable to create component: create fatal"
	hlp.On("SetFailureCount", failureComp).Return(time.Minute)
	hlp.On("UpdateStatus", failureComp).Return(nil)

	_, err = r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      comp.Name,
		Namespace: comp.Namespace,
	}})
	require.NoError(t, err)
	require.Error(t, loggerSink.LastError())
	assert.Equal(t, "unable to create component: create fatal", loggerSink.LastError().Error())
}

func TestIsSpecUpdated(t *testing.T) {
	comp := keycloakApi.KeycloakRealmComponent{}

	if isSpecUpdated(event.UpdateEvent{ObjectNew: &comp, ObjectOld: &comp}) {
		t.Fatal("spec is updated")
	}
}
