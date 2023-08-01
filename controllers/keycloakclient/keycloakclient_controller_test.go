package keycloakclient

import (
	"context"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	helpermock "github.com/epam/edp-keycloak-operator/controllers/helper/mocks"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakclient/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

func TestReconcileKeycloakClient_WithoutOwnerReference(t *testing.T) {
	kc := &keycloakApi.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main",
			Namespace: "namespace",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KeycloakClient",
			APIVersion: "apps/v1",
		},
		Spec: keycloakApi.KeycloakClientSpec{
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
			Secret: "keycloak-secret",
			RealmRoles: &[]keycloakApi.RealmRole{
				{
					Name:      "fake-client-administrators",
					Composite: "administrator",
				},
				{
					Name:      "fake-client-users",
					Composite: "developer",
				},
			},
			Public:                  true,
			ClientId:                "fake-client",
			WebUrl:                  "fake-url",
			DirectAccess:            false,
			AdvancedProtocolMappers: true,
			ClientRoles:             nil,
			Attributes: map[string]string{
				clientAttributeLogoutRedirectUris: clientAttributeLogoutRedirectUrisDefValue,
			},
		},
		Status: keycloakApi.KeycloakClientStatus{
			Value: "error during kc chain: fatal",
		},
	}
	s := scheme.Scheme
	require.NoError(t, keycloakApi.AddToScheme(s))

	client := fake.NewClientBuilder().WithRuntimeObjects(kc).Build()
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "main",
			Namespace: "namespace",
		},
	}
	h := helpermock.NewControllerHelper(t)
	chainMock := chain.Mock{}
	kClient := adapter.Mock{}

	chainMock.On("Serve", testifymock.Anything).Return(errors.New("fatal"))

	h.On("SetFailureCount", testifymock.Anything).Return(time.Second)
	h.On("SetRealmOwnerRef", testifymock.Anything, testifymock.Anything).Return(nil)
	h.On("CreateKeycloakClientFromRealmRef", testifymock.Anything, testifymock.Anything).Return(&kClient, nil)
	h.On("GetKeycloakRealmFromRef", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(&gocloak.RealmRepresentation{
			Realm: gocloak.StringP("realm"),
		}, nil)

	r := ReconcileKeycloakClient{
		client: client,
		helper: h,
		chain:  &chainMock,
	}
	res, err := r.Reconcile(context.TODO(), req)
	assert.Nil(t, err)

	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}

	assert.False(t, res.Requeue)

	persKc := &keycloakApi.KeycloakClient{}
	assert.Nil(t, client.Get(context.TODO(), req.NamespacedName, persKc))
	assert.Contains(t, persKc.Status.Value, "fatal")
	assert.Empty(t, persKc.Status.ClientID)
}

func TestReconcileKeycloakClient_ReconcileWithMappers(t *testing.T) {
	kc := keycloakApi.KeycloakClient{ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "namespace"},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KeycloakClient",
			APIVersion: "apps/v1",
		},
		Spec: keycloakApi.KeycloakClientSpec{TargetRealm: "namespace.main", Secret: "keycloak-secret",
			RealmRoles: &[]keycloakApi.RealmRole{{Name: "fake-client-administrators", Composite: "administrator"},
				{Name: "fake-client-users", Composite: "developer"},
			}, Public: true, ClientId: "fake-client", WebUrl: "fake-url", DirectAccess: false,
			AdvancedProtocolMappers: true, ClientRoles: nil, ProtocolMappers: &[]keycloakApi.ProtocolMapper{
				{Name: "bar", Config: map[string]string{"bar": "1"}},
				{Name: "foo", Config: map[string]string{"foo": "2"}},
			},
			Attributes: map[string]string{
				clientAttributeLogoutRedirectUris: clientAttributeLogoutRedirectUrisDefValue,
			},
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
		},
		Status: keycloakApi.KeycloakClientStatus{
			Value: helper.StatusOK,
		},
	}

	s := scheme.Scheme
	require.NoError(t, keycloakApi.AddToScheme(s))

	client := fake.NewClientBuilder().WithRuntimeObjects(&kc).Build()
	kclient := new(adapter.Mock)
	h := helpermock.NewControllerHelper(t)
	chainMock := chain.Mock{}
	chainMock.On("Serve", testifymock.Anything, testifymock.Anything, testifymock.Anything).Return(nil)

	h.On("SetRealmOwnerRef", testifymock.Anything, testifymock.Anything).Return(nil)
	h.On("CreateKeycloakClientFromRealmRef", testifymock.Anything, testifymock.Anything).Return(kclient, nil)
	h.On("TryToDelete", testifymock.Anything, testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(false, nil)

	r := ReconcileKeycloakClient{
		client:                  client,
		helper:                  h,
		chain:                   &chainMock,
		successReconcileTimeout: time.Hour,
	}

	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: "main", Namespace: "namespace"}}
	res, err := r.Reconcile(context.TODO(), req)
	require.NoError(t, err)

	if res.RequeueAfter != r.successReconcileTimeout {
		t.Fatal("success reconcile timeout is not set")
	}
}

func TestReconcileKeycloakClient_applyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		keycloakClient     *keycloakApi.KeycloakClient
		objects            []runtime.Object
		want               bool
		wantKeycloakClient *keycloakApi.KeycloakClient
		wantErr            require.ErrorAssertionFunc
	}{
		{
			name: "should set all default values",
			objects: []runtime.Object{
				&keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "realm",
					},
				},
			},
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					TargetRealm: "realm",
				},
			},
			wantKeycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					Attributes: map[string]string{
						clientAttributeLogoutRedirectUris: clientAttributeLogoutRedirectUrisDefValue,
					},
				},
			},
			want:    true,
			wantErr: require.NoError,
		},
		{
			name: "all default values are already set",
			objects: []runtime.Object{
				&keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "realm",
					},
				},
			},
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					Attributes: map[string]string{
						clientAttributeLogoutRedirectUris: "-",
						"foo":                             "bar",
					},
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "realm",
					},
				},
			},
			wantKeycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					Attributes: map[string]string{
						clientAttributeLogoutRedirectUris: "-",
						"foo":                             "bar",
					},
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "realm",
					},
				},
			},
			want:    false,
			wantErr: require.NoError,
		},
		{
			name: "should set main realm",
			objects: []runtime.Object{
				&keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "main",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "realm",
					},
				},
			},
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					TargetRealm: "some-realm",
				},
			},
			wantKeycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					Attributes: map[string]string{
						clientAttributeLogoutRedirectUris: clientAttributeLogoutRedirectUrisDefValue,
					},
				},
			},
			want:    true,
			wantErr: require.NoError,
		},
		{
			name: "realm not found",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					TargetRealm: "some-realm",
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "realm some-realm not found")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := runtime.NewScheme()
			require.NoError(t, keycloakApi.AddToScheme(sc))

			r := &ReconcileKeycloakClient{
				client: fake.NewClientBuilder().
					WithScheme(sc).
					WithRuntimeObjects(append(tt.objects, tt.keycloakClient)...).
					Build(),
			}

			got, err := r.applyDefaults(context.Background(), tt.keycloakClient)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
