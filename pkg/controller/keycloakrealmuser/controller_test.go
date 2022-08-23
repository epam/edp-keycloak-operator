package keycloakrealmuser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
)

func TestNewReconcile_Init(t *testing.T) {
	c := NewReconcile(nil, &mock.Logger{}, &helper.Mock{})
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
	helper      *helper.Mock
	logger      *mock.Logger
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
			Realm:    e.realmName,
		},
		Status: keycloakApi.KeycloakRealmUserStatus{
			Value: helper.StatusOK,
		},
	}
	e.k8sClient = fake.NewClientBuilder().WithScheme(e.scheme).WithRuntimeObjects(e.kcRealmUser).Build()
	e.helper = &helper.Mock{}
	e.logger = &mock.Logger{}
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
	e.helper.On("GetOrCreateRealmOwnerRef", e.kcRealmUser, &e.kcRealmUser.ObjectMeta).Return(e.kcRealm, nil)
	e.helper.On("CreateKeycloakClientForRealm", e.kcRealm).Return(e.kClient, nil)

	r := Reconcile{
		helper: e.helper,
		log:    e.logger,
		client: e.k8sClient,
	}

	e.kClient.On("SyncRealmUser", e.realmName, e.adapterUser, false).Return(nil)
	e.helper.On("UpdateStatus", e.kcRealmUser).Return(nil)
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

	e.helper.On("GetOrCreateRealmOwnerRef", e.kcRealmUser, &e.kcRealmUser.ObjectMeta).Return(e.kcRealm, nil)
	e.helper.On("CreateKeycloakClientForRealm", e.kcRealm).Return(e.kClient, nil)
	e.helper.On("TryToDelete", e.kcRealmUser,
		makeTerminator(e.realmName, e.kcRealmUser.Spec.Username, e.kClient, e.logger), finalizer).
		Return(false, nil)
	e.helper.On("UpdateStatus", e.kcRealmUser).Return(nil)

	r := Reconcile{
		helper: e.helper,
		log:    e.logger,
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
