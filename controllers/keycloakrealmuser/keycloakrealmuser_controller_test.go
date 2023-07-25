package keycloakrealmuser

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	helpermock "github.com/epam/edp-keycloak-operator/controllers/helper/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

func TestNewReconcile_Init(t *testing.T) {
	c := NewReconcile(nil, nil)
	if c.client != nil {
		t.Fatal("something went wrong")
	}
}

type TestControllerSuite struct {
	suite.Suite
	namespace   string
	scheme      *runtime.Scheme
	realmName   string
	kcRealmUser *keycloakApi.KeycloakRealmUser
	k8sClient   client.Client
	helper      *helpermock.ControllerHelper
	kcRealm     *keycloakApi.KeycloakRealm
	kClient     *adapter.Mock
	adapterUser *adapter.KeycloakUser
}

func (e *TestControllerSuite) SetupTest() {
	e.namespace = "ns"
	e.scheme = runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(e.scheme))
	e.realmName = "realmName"
	e.kcRealmUser = &keycloakApi.KeycloakRealmUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user321",
			Namespace: e.namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KeycloakRealmUser",
			APIVersion: "v1.edp.epam.com/v1",
		},
		Spec: keycloakApi.KeycloakRealmUserSpec{
			Email:    "usr@gmail.com",
			Username: "user.g1",
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: e.realmName,
			},
		},
		Status: keycloakApi.KeycloakRealmUserStatus{
			Value: helper.StatusOK,
		},
	}
	e.k8sClient = fake.NewClientBuilder().WithScheme(e.scheme).WithRuntimeObjects(e.kcRealmUser).Build()
	e.helper = helpermock.NewControllerHelper(e.T())
	e.kcRealm = &keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: e.realmName,
		},
	}
	e.kClient = &adapter.Mock{}
	e.adapterUser = &adapter.KeycloakUser{
		Username:            e.kcRealmUser.Spec.Username,
		Groups:              e.kcRealmUser.Spec.Groups,
		Roles:               e.kcRealmUser.Spec.Roles,
		RequiredUserActions: e.kcRealmUser.Spec.RequiredUserActions,
		LastName:            e.kcRealmUser.Spec.LastName,
		FirstName:           e.kcRealmUser.Spec.FirstName,
		EmailVerified:       e.kcRealmUser.Spec.EmailVerified,
		Enabled:             e.kcRealmUser.Spec.Enabled,
		Email:               e.kcRealmUser.Spec.Email,
	}
}

func (e *TestControllerSuite) TestNewReconcile() {
	e.helper.On("SetRealmOwnerRef", testifymock.Anything, testifymock.Anything).Return(nil)
	e.helper.On("CreateKeycloakClientFromRealmRef", testifymock.Anything, testifymock.Anything).Return(e.kClient, nil)
	e.helper.On("GetKeycloakRealmFromRef", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(&gocloak.RealmRepresentation{
			Realm: gocloak.StringP(e.realmName),
		}, nil)

	r := Reconcile{
		helper: e.helper,
		client: e.k8sClient,
	}

	e.kClient.On("SyncRealmUser", e.realmName, e.adapterUser, false).Return(nil)

	_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: e.namespace,
		Name:      e.kcRealmUser.Name,
	}})
	assert.NoError(e.T(), err)

	var checkUser keycloakApi.KeycloakRealmUser
	err = e.k8sClient.Get(context.Background(),
		types.NamespacedName{Name: e.kcRealmUser.Name, Namespace: e.kcRealmUser.Namespace}, &checkUser)
	assert.Error(e.T(), err, "user is not deleted")
	assert.True(e.T(), k8sErrors.IsNotFound(err), "wrong error returned")
}

func (e *TestControllerSuite) TestReconcileKeep() {
	e.kcRealmUser.Spec.KeepResource = true
	e.k8sClient = fake.NewClientBuilder().WithScheme(e.scheme).WithRuntimeObjects(e.kcRealmUser).Build()

	e.helper.On("SetRealmOwnerRef", testifymock.Anything, testifymock.Anything).Return(nil)
	e.helper.On("CreateKeycloakClientFromRealmRef", testifymock.Anything, testifymock.Anything).Return(e.kClient, nil)
	e.helper.On("GetKeycloakRealmFromRef", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(&gocloak.RealmRepresentation{
			Realm: gocloak.StringP(e.realmName),
		}, nil)
	e.helper.On("TryToDelete", testifymock.Anything, testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(false, nil)

	r := Reconcile{
		helper: e.helper,
		client: e.k8sClient,
	}

	e.kClient.On("SyncRealmUser", e.realmName, e.adapterUser, false).Return(nil)

	_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: e.namespace,
		Name:      e.kcRealmUser.Name,
	}})
	assert.NoError(e.T(), err)

	var checkUser keycloakApi.KeycloakRealmUser
	err = e.k8sClient.Get(context.Background(),
		types.NamespacedName{Name: e.kcRealmUser.Name, Namespace: e.kcRealmUser.Namespace}, &checkUser)
	assert.NoError(e.T(), err)
}

func TestAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(TestControllerSuite))
}

func (e *TestControllerSuite) TestGetPassword() {
	e.kcRealmUser.Spec.PasswordSecret.Name = "my-secret"
	e.kcRealmUser.Spec.PasswordSecret.Key = "my-key"

	secret := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-secret",
			Namespace: e.namespace,
		},
		Data: map[string][]byte{
			"my-key": []byte("my-secret-password"),
		},
	}

	e.scheme.AddKnownTypes(coreV1.SchemeGroupVersion, secret)

	e.k8sClient = fake.NewClientBuilder().WithScheme(e.scheme).WithRuntimeObjects(e.kcRealmUser, secret).Build()

	r := &Reconcile{
		client: e.k8sClient,
	}

	password, err := r.getPassword(context.Background(), e.kcRealmUser)
	assert.NoError(e.T(), err)
	assert.Equal(e.T(), "my-secret-password", password)

	e.kcRealmUser.Spec.PasswordSecret.Key = "non-existent-key"
	password, err = r.getPassword(context.Background(), e.kcRealmUser)
	assert.Error(e.T(), err)
	assert.Equal(e.T(), "", password)

	e.kcRealmUser.Spec.PasswordSecret.Name = "non-existent-secret"
	password, err = r.getPassword(context.Background(), e.kcRealmUser)
	assert.Error(e.T(), err)
	assert.Equal(e.T(), "", password)

	e.kcRealmUser.Spec.PasswordSecret.Name = ""
	e.kcRealmUser.Spec.Password = "spec-password"
	password, err = r.getPassword(context.Background(), e.kcRealmUser)
	assert.NoError(e.T(), err)
	assert.Equal(e.T(), "spec-password", password)
}
