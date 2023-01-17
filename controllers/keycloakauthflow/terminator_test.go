package keycloakauthflow

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestTerminator(t *testing.T) {
	sch := runtime.NewScheme()
	assert.NoError(t, keycloakApi.AddToScheme(sch))

	fakeClient := fake.NewClientBuilder().WithScheme(sch).Build()

	lg := mock.NewLogr()
	kClient := new(adapter.Mock)

	keycloakAuthFlow := adapter.KeycloakAuthFlow{Alias: "foo"}
	realm := keycloakApi.KeycloakRealm{Spec: keycloakApi.KeycloakRealmSpec{RealmName: "foo"}}

	term := makeTerminator(&realm, &keycloakAuthFlow, fakeClient, kClient, lg)

	kClient.On("DeleteAuthFlow", "foo", &keycloakAuthFlow).Return(nil).Once()

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteAuthFlow", "foo", &keycloakAuthFlow).Return(errors.New("fatal")).Once()

	err = term.DeleteResource(context.Background())
	require.Error(t, err)

	loggerSink, ok := lg.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")
	assert.NotEmpty(t, loggerSink.InfoMessages(), "no info messages logged")
}

func TestTerminatorDeleteResourceWithChildErr(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(scheme))

	flow := keycloakApi.KeycloakAuthFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flow123",
			Namespace: "namespace1",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KeycloakAuthFlow",
			APIVersion: "v1.edp.epam.com/v1",
		},
		Spec: keycloakApi.KeycloakAuthFlowSpec{
			Alias:      "flow123",
			Realm:      "foo",
			ParentName: "foo",
		},
		Status: keycloakApi.KeycloakAuthFlowStatus{
			Value: helper.StatusOK,
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&flow).Build()
	lg := mock.NewLogr()
	kClient := new(adapter.Mock)
	keycloakAuthFlow := adapter.KeycloakAuthFlow{Alias: "foo"}
	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "foo",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}

	term := makeTerminator(&realm, &keycloakAuthFlow, fakeClient, kClient, lg)

	kClient.On("DeleteAuthFlow", "foo", &keycloakAuthFlow).Return(nil).Once()

	err := term.DeleteResource(context.Background())
	assert.Error(t, err)
}
