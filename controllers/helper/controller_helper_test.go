package helper

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/fakehttp"
)

func TestHelper_GetOrCreateRealmOwnerRef(t *testing.T) {
	mc := K8SClientMock{}

	sch := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(sch))

	helper := MakeHelper(&mc, sch, "default")

	kcGroup := keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
		},
		Spec: keycloakApi.KeycloakRealmGroupSpec{
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
		},
	}

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "realm",
	}, &keycloakApi.KeycloakRealm{}).Return(nil)
	mc.On("Update", testifymock.Anything, testifymock.Anything).Return(nil)

	err := helper.SetRealmOwnerRef(context.Background(), &kcGroup)
	require.NoError(t, err)

	kcGroup = keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
		},
		Spec: keycloakApi.KeycloakRealmGroupSpec{
			Realm: "foo13",
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
		},
	}

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      "foo13",
	}, &keycloakApi.KeycloakRealm{}).Return(nil)

	err = helper.SetRealmOwnerRef(context.Background(), &kcGroup)
	require.NoError(t, err)
}

func TestHelper_GetOrCreateRealmOwnerRef_Failure(t *testing.T) {
	mc := K8SClientMock{}

	sch := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(sch))

	helper := MakeHelper(&mc, sch, "default")

	kcGroup := keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "foo",
					Kind: "KeycloakRealm",
				},
			},
		},
		Spec: keycloakApi.KeycloakRealmGroupSpec{
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
		},
	}

	mockErr := errors.New("mock error")

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      kcGroup.Spec.RealmRef.Name,
	}, &keycloakApi.KeycloakRealm{}).Return(mockErr)

	err := helper.SetRealmOwnerRef(context.Background(), &kcGroup)
	if err == nil {
		t.Fatal("no error on k8s client get fatal")
	}

	assert.ErrorIs(t, err, mockErr)

	kcGroup = keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
		},
		Spec: keycloakApi.KeycloakRealmGroupSpec{
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
		},
	}

	mc.On("Get", types.NamespacedName{
		Namespace: "test",
		Name:      kcGroup.Spec.RealmRef.Name,
	}, &keycloakApi.KeycloakRealm{}).Return(mockErr)

	err = helper.SetRealmOwnerRef(context.Background(), &kcGroup)
	if err == nil {
		t.Fatal("no error on k8s client get fatal")
	}

	assert.ErrorIs(t, err, mockErr)
}

func TestMakeHelper(t *testing.T) {
	rCl := resty.New()

	mockServer := fakehttp.NewServerBuilder().
		AddStringResponder("/auth/realms/master/protocol/openid-connect/token/", "{}").
		BuildAndStart()
	defer mockServer.Close()

	logger := mock.NewLogr()
	h := MakeHelper(nil, nil, "default")
	_, err := h.adapterBuilder(
		context.Background(),
		adapter.GoCloakConfig{
			Url:      mockServer.GetURL(),
			User:     "foo",
			Password: "bar",
		},
		keycloakApi.KeycloakAdminTypeServiceAccount,
		logger,
		rCl,
	)
	require.NoError(t, err)
}

type testTerminator struct {
	err error
	log logr.Logger
}

func (t *testTerminator) DeleteResource(ctx context.Context) error {
	return t.err
}
func (t *testTerminator) GetLogger() logr.Logger {
	return t.log
}

func TestHelper_TryToDelete(t *testing.T) {
	logger := mock.NewLogr()

	term := testTerminator{
		log: logger,
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-secret1"}}
	fakeClient := fake.NewClientBuilder().WithRuntimeObjects(&secret).Build()
	h := Helper{client: fakeClient}

	_, err := h.TryToDelete(context.Background(), &secret, &term, "fin")
	require.NoError(t, err)

	term.err = errors.New("delete resource fatal")
	secret.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	_, err = h.TryToDelete(context.Background(), &secret, &term, "fin")
	require.Error(t, err)

	if err.Error() != "error during keycloak resource deletion: delete resource fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
