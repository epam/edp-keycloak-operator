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

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
)

func TestTerminator(t *testing.T) {
	sch := runtime.NewScheme()
	assert.NoError(t, keycloakApi.AddToScheme(sch))

	fakeClient := fake.NewClientBuilder().WithScheme(sch).Build()

	lg := mock.Logger{}
	kClient := new(adapter.Mock)

	keycloakAuthFlow := adapter.KeycloakAuthFlow{Alias: "foo"}
	realm := keycloakApi.KeycloakRealm{Spec: keycloakApi.KeycloakRealmSpec{RealmName: "foo"}}

	term := makeTerminator(&realm, &keycloakAuthFlow, fakeClient, kClient, &lg)

	if term.GetLogger() != &lg {
		t.Fatal("wrong logger set")
	}

	kClient.On("DeleteAuthFlow", "foo", &keycloakAuthFlow).Return(nil).Once()
	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteAuthFlow", "foo", &keycloakAuthFlow).Return(errors.New("fatal")).Once()
	if err := term.DeleteResource(context.Background()); err == nil {
		t.Fatal("no error returned")
	}

	if len(lg.InfoMessages) == 0 {
		t.Fatal("no info messages logged")
	}
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
	lg := mock.Logger{}
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

	term := makeTerminator(&realm, &keycloakAuthFlow, fakeClient, kClient, &lg)

	kClient.On("DeleteAuthFlow", "foo", &keycloakAuthFlow).Return(nil).Once()

	err := term.DeleteResource(context.Background())
	assert.Error(t, err)
}
