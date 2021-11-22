package keycloakrealmcomponent

import (
	"context"
	"testing"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile_Reconcile(t *testing.T) {
	logger := mock.Logger{}
	sch := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(sch))
	utilruntime.Must(corev1.AddToScheme(sch))

	var (
		hlp       helper.Mock
		kcAdapter adapter.Mock
		comp      = v1alpha1.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{Name: "test-comp-name", Namespace: "ns"},
			TypeMeta:   metav1.TypeMeta{Kind: "KeycloakRealmComponent", APIVersion: "v1.edp.epam.com/v1alpha1"},
			Spec:       v1alpha1.KeycloakComponentSpec{Name: "test-comp"},
			Status:     v1alpha1.KeycloakComponentStatus{Value: helper.StatusOK},
		}
		realm = v1alpha1.KeycloakRealm{TypeMeta: metav1.TypeMeta{
			APIVersion: "v1.edp.epam.com/v1alpha1", Kind: "KeycloakRealm",
		},
			ObjectMeta: metav1.ObjectMeta{Name: "realm1", Namespace: "ns",
				OwnerReferences: []metav1.OwnerReference{{Name: "keycloak1", Kind: "Keycloak"}}},
			Spec: v1alpha1.KeycloakRealmSpec{RealmName: "ns.realm1"}}
		testComp = adapter.Component{ID: "component-id1", Name: comp.Spec.Name}
	)

	client := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&comp).Build()
	hlp.On("GetOrCreateRealmOwnerRef", &comp, comp.ObjectMeta).Return(&realm, nil)
	hlp.On("CreateKeycloakClientForRealm", &realm).Return(&kcAdapter, nil)
	kcAdapter.On("GetComponent", realm.Spec.RealmName, comp.Spec.Name).Return(&testComp, nil).Once()
	kcAdapter.On("UpdateComponent", realm.Spec.RealmName, &testComp).Return(nil)
	hlp.On("TryToDelete", &comp, makeTerminator(realm.Spec.RealmName, comp.Spec.Name, &kcAdapter, &logger),
		finalizerName).Return(false, nil)
	hlp.On("UpdateStatus", &comp).Return(nil)
	r := NewReconcile(client, &logger, &hlp)

	res, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      comp.Name,
		Namespace: comp.Namespace,
	}})
	if err != nil {
		t.Fatal(err)
	}

	if err := logger.LastError(); err != nil {
		t.Fatal(err)
	}

	if res.RequeueAfter != r.successReconcileTimeout {
		t.Fatalf("wrong RequeueAfter: %d", res.RequeueAfter)
	}

	kcAdapter.On("GetComponent", realm.Spec.RealmName, comp.Spec.Name).Return(nil,
		adapter.ErrNotFound("not found")).Once()
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
	if err != nil {
		t.Fatal(err)
	}

	err = logger.LastError()
	if err == nil {
		t.Fatal("no error returned")
	}

	if err.Error() != "unable to create component: create fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestIsSpecUpdated(t *testing.T) {
	comp := v1alpha1.KeycloakRealmComponent{}
	if isSpecUpdated(event.UpdateEvent{ObjectNew: &comp, ObjectOld: &comp}) {
		t.Fatal("spec is updated")
	}
}
