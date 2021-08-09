package keycloakclientscope

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/model"
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

func getTestClientScope(realmName string) *v1alpha1.KeycloakClientScope {
	return &v1alpha1.KeycloakClientScope{
		ObjectMeta: metav1.ObjectMeta{
			Name: "scope1",
		},
		TypeMeta: metav1.TypeMeta{Kind: "KeycloakClientScope", APIVersion: "v1.edp.epam.com/v1alpha1"},
		Spec: v1alpha1.KeycloakClientScopeSpec{
			Name:  "scope1name",
			Realm: realmName,
		},
	}
}

func TestReconcile_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	ns := "security"
	keycloak := v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns},
		Spec: v1alpha1.KeycloakSpec{
			Secret: "keycloak-secret",
		},
		Status: v1alpha1.KeycloakStatus{Connected: true}}
	realm := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "Keycloak"}}},
		Spec: v1alpha1.KeycloakRealmSpec{RealmName: "ns.test"}}

	clientScope := getTestClientScope(realm.Name)

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(clientScope, &realm, &keycloak).Build()
	kClient := new(adapter.Mock)
	kClient.On("GetClientScope", clientScope.Spec.Name, realm.Spec.RealmName).
		Return(nil, adapter.ErrNotFound("not found"))
	kClient.On("CreateClientScope", realm.Spec.RealmName, &adapter.ClientScope{
		Name:            clientScope.Spec.Name,
		ProtocolMappers: []adapter.ProtocolMapper{},
	}).
		Return("scope12", nil)

	logger := mock.Logger{}
	h := helper.Mock{}
	h.On("CreateKeycloakClientForRealm", &realm, &logger).Return(kClient, nil)
	h.On("GetOrCreateRealmOwnerRef", clientScope, clientScope.ObjectMeta).Return(&realm, nil)

	updatedClientScopeWithID := getTestClientScope(realm.Name)
	updatedClientScopeWithID.Status.ID = "scope12"
	updatedClientScopeWithID.ResourceVersion = "999"

	updatedClientScopeWithStatus := getTestClientScope(realm.Name)
	updatedClientScopeWithStatus.Status.ID = "scope12"
	updatedClientScopeWithStatus.ResourceVersion = "999"
	updatedClientScopeWithStatus.Status.Value = helper.StatusOK

	h.On("UpdateStatus", updatedClientScopeWithStatus).Return(nil)

	h.On("TryToDelete", updatedClientScopeWithID,
		makeTerminator(context.Background(), kClient, realm.Spec.RealmName, "scope12"), finalizerName).
		Return(true, nil)

	rkr := Reconcile{
		log:    &logger,
		client: client,
		helper: &h,
	}

	if _, err := rkr.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      clientScope.Name,
			Namespace: clientScope.Namespace,
		}}); err != nil {
		t.Fatal(err)
	}
}

func TestSpecIsUpdated(t *testing.T) {
	cs := getTestClientScope("test")
	if isSpecUpdated(event.UpdateEvent{
		ObjectNew: cs,
		ObjectOld: cs,
	}) {
		t.Fatal("spec must not be updated")
	}
}

func TestNewReconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := mock.Logger{}
	rec := NewReconcile(client, scheme, &logger)
	if rec == nil {
		t.Fatal("reconciler is not inited")
	}

	if rec.log != &logger || rec.client != client || rec.scheme != scheme {
		t.Fatal("wrong reconciler params")
	}
}

func TestReconcile_Reconcile_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := mock.Logger{}
	rec := NewReconcile(client, scheme, &logger)

	if _, err := rec.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "bar"}}); err != nil {
		t.Fatal(err)
	}

	if _, ok := logger.InfoMessages["instance not found"]; !ok {
		t.Fatal("no info messages is logged")
	}
}

func TestConvertProtocolMappers(t *testing.T) {
	mappers := convertProtocolMappers([]v1alpha1.ProtocolMapper{
		{Name: "test1"},
	})

	if len(mappers) == 0 {
		t.Fatal("protocol mappers is not converted")
	}

	if mappers[0].Name != "test1" {
		t.Fatal("protocol mappers converted wrongly")
	}
}

func TestSyncClientScope(t *testing.T) {
	kClient := new(adapter.Mock)
	realm := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns",
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "Keycloak"}}},
		Spec: v1alpha1.KeycloakRealmSpec{RealmName: "ns.test"}}
	instance := getTestClientScope(realm.Name)
	instance.Status.ID = "scopeID1"

	kClient.On("GetClientScope", instance.Spec.Name, realm.Spec.RealmName).Return(&model.ClientScope{}, nil)
	kClient.On("UpdateClientScope", realm.Spec.RealmName, instance.Status.ID, &adapter.ClientScope{
		Name:            instance.Spec.Name,
		ProtocolMappers: []adapter.ProtocolMapper{},
	}).Return(nil)

	if err := syncClientScope(context.Background(), instance, &realm, kClient); err != nil {
		t.Fatal(err)
	}
}

func TestReconcile_Reconcile_FailureNoRealm(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	instance := getTestClientScope("test")

	client := fake.NewClientBuilder().WithRuntimeObjects(instance).WithScheme(scheme).Build()
	logger := mock.Logger{}
	rec := NewReconcile(client, scheme, &logger)

	if _, err := rec.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}}); err != nil {
		t.Fatalf("%+v", err)
	}

	err := logger.LastError()
	if err == nil {
		t.Fatal("no error logged")
	}

	if !strings.Contains(err.Error(), "unable to get realm owner ref") {
		t.Fatalf("wrong error logged: %s", err.Error())
	}
}
func TestReconcile_Reconcile_FailureNoClientForRealm(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	realm := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns",
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "Keycloak"}}},
		Spec: v1alpha1.KeycloakRealmSpec{RealmName: "ns.test"}}
	clientScope := getTestClientScope(realm.Name)

	client := fake.NewClientBuilder().WithRuntimeObjects(clientScope, &realm).WithScheme(scheme).Build()
	logger := mock.Logger{}
	rec := NewReconcile(client, scheme, &logger)

	h := helper.Mock{}
	h.On("GetOrCreateRealmOwnerRef", clientScope, clientScope.ObjectMeta).Return(&realm, nil)
	h.On("CreateKeycloakClientForRealm", &realm, &logger).
		Return(nil, errors.New("fatal"))

	updatedClientScope := getTestClientScope(realm.Name)
	updatedClientScope.Status.Value = "unable to create keycloak client: fatal"
	updatedClientScope.ResourceVersion = "999"

	h.On("SetFailureCount", updatedClientScope).Return(time.Minute)
	h.On("UpdateStatus", updatedClientScope).Return(nil)

	rec.helper = &h

	if _, err := rec.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{Name: clientScope.Name, Namespace: clientScope.Namespace}}); err != nil {
		t.Fatalf("%+v", err)
	}

	err := logger.LastError()
	if err == nil {
		t.Fatal("no error logged")
	}

	if !strings.Contains(err.Error(), "unable to create keycloak client") {
		t.Fatalf("wrong error logged: %s", err.Error())
	}
}
