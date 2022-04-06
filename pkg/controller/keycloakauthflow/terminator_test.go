package keycloakauthflow

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestTerminator(t *testing.T) {
	sch := runtime.NewScheme()
	assert.NoError(t, v1alpha1.AddToScheme(sch))

	fakeClient := fake.NewClientBuilder().WithScheme(sch).Build()

	lg := mock.Logger{}
	kClient := new(adapter.Mock)

	keycloakAuthFlow := adapter.KeycloakAuthFlow{Alias: "foo"}
	realm := v1alpha1.KeycloakRealm{Spec: v1alpha1.KeycloakRealmSpec{RealmName: "foo"}}

	term := makeTerminator(&realm, &keycloakAuthFlow, fakeClient, kClient, &lg)

	if term.GetLogger() != &lg {
		t.Fatal("wrong logger set")
	}

	kClient.On("DeleteAuthFlow", "foo", &keycloakAuthFlow).Return(nil).Once()
	if err := term.DeleteResource(context.Background()); err != nil {
		t.Fatal(err)
	}

	kClient.On("DeleteAuthFlow", "foo", &keycloakAuthFlow).Return(errors.New("fatal")).Once()
	if err := term.DeleteResource(context.Background()); err == nil {
		t.Fatal("no error returned")
	}

	if len(lg.InfoMessages) == 0 {
		t.Fatal("no info messages logged")
	}
}
