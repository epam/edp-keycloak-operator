package keycloakrealmidentityprovider

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

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestNewReconcileUnexpectedError(t *testing.T) {
	sch := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(sch))
	utilruntime.Must(corev1.AddToScheme(sch))

	nn := types.NamespacedName{
		Name:      "foo",
		Namespace: "bar",
	}
	fakeCl := helper.K8SClientMock{}
	fakeCl.On("Get", nn, &keycloakApi.KeycloakRealmIdentityProvider{}).Return(errors.New("fatal"))

	l := mock.NewLogr()

	r := NewReconcile(&fakeCl, l, nil)
	_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: nn})

	require.Error(t, err)

	if err.Error() != "unable to get keycloak realm idp from k8s: fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestNewReconcileNotFound(t *testing.T) {
	sch := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(sch))
	utilruntime.Must(corev1.AddToScheme(sch))
	fakeCl := fake.NewClientBuilder().WithScheme(sch).Build()

	l := mock.NewLogr()

	r := NewReconcile(fakeCl, l, nil)
	_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      "foo",
		Namespace: "bar",
	}})
	require.NoError(t, err)

	loggerSink, ok := l.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")
	require.NoError(t, loggerSink.LastError())

	if _, ok := loggerSink.InfoMessages()["instance not found"]; !ok {
		t.Fatal("no 404 logged")
	}
}

func TestNewReconcile(t *testing.T) {
	hlp := helper.Mock{}
	l := mock.NewLogr()
	kcAdapter := adapter.Mock{}
	idp := keycloakApi.KeycloakRealmIdentityProvider{
		ObjectMeta: metav1.ObjectMeta{Name: "idp1", Namespace: "ns"},
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1.edp.epam.com/v1", Kind: "KeycloakRealmIdentityProvider"},
		Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
			Alias: "alias1",
			Mappers: []keycloakApi.IdentityProviderMapper{
				{
					Name: "mapper1",
				},
			},
		},
		Status: keycloakApi.KeycloakRealmIdentityProviderStatus{Value: helper.StatusOK},
	}

	realm := keycloakApi.KeycloakRealm{TypeMeta: metav1.TypeMeta{
		APIVersion: "v1.edp.epam.com/v1", Kind: "KeycloakRealm",
	},
		ObjectMeta: metav1.ObjectMeta{Name: "realm1", Namespace: "ns",
			OwnerReferences: []metav1.OwnerReference{{Name: "keycloak1", Kind: "Keycloak"}}},
		Spec: keycloakApi.KeycloakRealmSpec{RealmName: "ns.realm1"}}

	sch := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(sch))
	utilruntime.Must(corev1.AddToScheme(sch))

	fakeCl := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&idp).Build()

	hlp.On("GetOrCreateRealmOwnerRef", &idp, &idp.ObjectMeta).Return(&realm, nil)
	hlp.On("CreateKeycloakClientForRealm", &realm).Return(&kcAdapter, nil)

	kcAdapter.On("IdentityProviderExists", context.Background(), realm.Spec.RealmName, idp.Spec.Alias).
		Return(false, nil).Once()
	kcAdapter.On("CreateIdentityProvider", realm.Spec.RealmName,
		&adapter.IdentityProvider{Alias: idp.Spec.Alias}).Return(nil).Once()
	kcAdapter.On("GetIDPMappers", realm.Spec.RealmName, idp.Spec.Alias).
		Return([]adapter.IdentityProviderMapper{
			{
				ID:   "mapper-id1",
				Name: "mapper-name1",
			},
		}, nil)
	kcAdapter.On("DeleteIDPMapper", realm.Spec.RealmName, idp.Spec.Alias, "mapper-id1").
		Return(nil)
	kcAdapter.On("CreateIDPMapper", realm.Spec.RealmName, idp.Spec.Alias,
		&adapter.IdentityProviderMapper{Name: "mapper1", IdentityProviderAlias: idp.Spec.Alias}).
		Return("mp1", nil)

	hlp.On("TryToDelete", &idp,
		makeTerminator(realm.Spec.RealmName, idp.Spec.Alias, &kcAdapter, l), finalizerName).
		Return(false, nil)
	hlp.On("UpdateStatus", &idp).Return(nil)

	r := NewReconcile(fakeCl, l, &hlp)
	_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      idp.Name,
		Namespace: idp.Namespace,
	}})
	require.NoError(t, err)

	loggerSink, ok := l.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")
	require.NoError(t, loggerSink.LastError())

	kcAdapter.On("IdentityProviderExists", context.Background(), realm.Spec.RealmName, idp.Spec.Alias).
		Return(true, nil).Once()
	kcAdapter.On("UpdateIdentityProvider", realm.Spec.RealmName,
		&adapter.IdentityProvider{Alias: idp.Spec.Alias}).Return(nil).Once()

	_, err = r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      idp.Name,
		Namespace: idp.Namespace,
	}})
	require.NoError(t, err)

	kcAdapter.On("IdentityProviderExists", context.Background(), realm.Spec.RealmName, idp.Spec.Alias).
		Return(true, nil).Once()
	kcAdapter.On("UpdateIdentityProvider", realm.Spec.RealmName,
		&adapter.IdentityProvider{Alias: idp.Spec.Alias}).Return(errors.New("update idp fatal")).Once()

	idp.Status.Value = "unable to update idp: update idp fatal"
	r.client = fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&idp).Build()

	hlp.On("SetFailureCount", &idp).Return(time.Second)

	_, err = r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      idp.Name,
		Namespace: idp.Namespace,
	}})
	require.NoError(t, err)

	require.Error(t, loggerSink.LastError())
	assert.Equal(t, "unable to update idp: update idp fatal", loggerSink.LastError().Error())
}

func TestIsSpecUpdated(t *testing.T) {
	idp := keycloakApi.KeycloakRealmIdentityProvider{}

	if isSpecUpdated(event.UpdateEvent{ObjectOld: &idp, ObjectNew: &idp}) {
		t.Fatal("spec updated")
	}
}
