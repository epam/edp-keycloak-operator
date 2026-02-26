package helper

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakApiAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
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

func TestMakeHelper(t *testing.T) {
	rCl := resty.New()

	mockServer := fakehttp.NewServerBuilder().
		AddStringResponder("/auth/realms/master/protocol/openid-connect/token/", "{}").
		BuildAndStart()
	defer mockServer.Close()

	logger := mock.NewLogr()
	h := MakeHelper(nil, nil, "default", EnableOwnerRef(true))
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
	assert.True(t, h.enableOwnerRef)
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

func TestHelper_SetRealmOwnerRef(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))
	require.NoError(t, keycloakApiAlpha.AddToScheme(scheme))

	type fields struct {
		client         func(t *testing.T) client.Client
		enableOwnerRef bool
	}

	type args struct {
		object ObjectWithRealmRef
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr require.ErrorAssertionFunc
		want    func(t *testing.T, k8sCl client.Client)
	}{
		{
			name: "set KeycloakRealm owner reference",
			fields: fields{
				client: func(t *testing.T) client.Client {
					realm := &keycloakApi.KeycloakRealm{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "test",
							Name:      "realm",
						},
					}
					group := &keycloakApi.KeycloakRealmGroup{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "test",
							Name:      "test-group",
						},
						Spec: keycloakApi.KeycloakRealmGroupSpec{
							RealmRef: common.RealmRef{
								Kind: keycloakApi.KeycloakRealmKind,
								Name: "realm",
							},
						},
					}

					return fake.NewClientBuilder().WithScheme(scheme).WithObjects(realm, group).Build()
				},
				enableOwnerRef: true,
			},
			args: args{
				object: &keycloakApi.KeycloakRealmGroup{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       "test",
						Name:            "test-group",
						ResourceVersion: "999",
					},
					Spec: keycloakApi.KeycloakRealmGroupSpec{
						RealmRef: common.RealmRef{
							Kind: keycloakApi.KeycloakRealmKind,
							Name: "realm",
						},
					},
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sCl client.Client) {
				group := &keycloakApi.KeycloakRealmGroup{}
				err := k8sCl.Get(context.Background(), types.NamespacedName{
					Namespace: "test",
					Name:      "test-group",
				}, group)

				require.NoError(t, err)
				require.NotNil(t, metav1.GetControllerOf(group))
			},
		},
		{
			name: "set ClusterKeycloakRealm owner reference",
			fields: fields{
				client: func(t *testing.T) client.Client {
					realm := &keycloakApiAlpha.ClusterKeycloakRealm{
						ObjectMeta: metav1.ObjectMeta{
							Name: "realm",
						},
					}
					group := &keycloakApi.KeycloakRealmGroup{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "test",
							Name:      "test-group",
						},
						Spec: keycloakApi.KeycloakRealmGroupSpec{
							RealmRef: common.RealmRef{
								Kind: keycloakApiAlpha.ClusterKeycloakRealmKind,
								Name: "realm",
							},
						},
					}

					return fake.NewClientBuilder().WithScheme(scheme).WithObjects(realm, group).Build()
				},
				enableOwnerRef: true,
			},
			args: args{
				object: &keycloakApi.KeycloakRealmGroup{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       "test",
						Name:            "test-group",
						ResourceVersion: "999",
					},
					Spec: keycloakApi.KeycloakRealmGroupSpec{
						RealmRef: common.RealmRef{
							Kind: keycloakApiAlpha.ClusterKeycloakRealmKind,
							Name: "realm",
						},
					},
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sCl client.Client) {
				group := &keycloakApi.KeycloakRealmGroup{}
				err := k8sCl.Get(context.Background(), types.NamespacedName{
					Namespace: "test",
					Name:      "test-group",
				}, group)

				require.NoError(t, err)
				require.NotNil(t, metav1.GetControllerOf(group))
			},
		},
		{
			name: "owner reference not set when enableOwnerRef is false",
			fields: fields{
				client: func(t *testing.T) client.Client {
					group := &keycloakApi.KeycloakRealmGroup{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "test",
							Name:      "test-group",
						},
						Spec: keycloakApi.KeycloakRealmGroupSpec{
							RealmRef: common.RealmRef{
								Kind: keycloakApi.KeycloakRealmKind,
								Name: "realm",
							},
						},
					}

					return fake.NewClientBuilder().WithScheme(scheme).WithObjects(group).Build()
				},
				enableOwnerRef: false,
			},
			args: args{
				object: &keycloakApi.KeycloakRealmGroup{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test",
						Name:      "test-group",
					},
					Spec: keycloakApi.KeycloakRealmGroupSpec{
						RealmRef: common.RealmRef{
							Kind: keycloakApi.KeycloakRealmKind,
							Name: "realm",
						},
					},
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sCl client.Client) {
				group := &keycloakApi.KeycloakRealmGroup{}
				err := k8sCl.Get(context.Background(), types.NamespacedName{
					Namespace: "test",
					Name:      "test-group",
				}, group)
				require.NoError(t, err)
				require.Nil(t, metav1.GetControllerOf(group))
			},
		},
		{
			name: "error when KeycloakRealm not found",
			fields: fields{
				client: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().WithScheme(scheme).Build()
				},
				enableOwnerRef: true,
			},
			args: args{
				object: &keycloakApi.KeycloakRealmGroup{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test",
						Name:      "test-group",
					},
					Spec: keycloakApi.KeycloakRealmGroupSpec{
						RealmRef: common.RealmRef{
							Kind: keycloakApi.KeycloakRealmKind,
							Name: "nonexistent",
						},
					},
				},
			},
			wantErr: require.Error,
			want:    func(t *testing.T, k8sCl client.Client) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sCl := tt.fields.client(t)
			h := &Helper{
				client:         k8sCl,
				scheme:         scheme,
				enableOwnerRef: tt.fields.enableOwnerRef,
			}
			err := h.SetRealmOwnerRef(context.Background(), tt.args.object)

			tt.wantErr(t, err)

			if tt.want != nil {
				tt.want(t, k8sCl)
			}
		})
	}
}

func TestHelper_GetRealmNameFromRef(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))
	require.NoError(t, keycloakApiAlpha.AddToScheme(scheme))

	tests := []struct {
		name     string
		client   func(t *testing.T) client.Client
		object   ObjectWithRealmRef
		wantName string
		wantErr  require.ErrorAssertionFunc
	}{
		{
			name: "get realm name from KeycloakRealm",
			client: func(t *testing.T) client.Client {
				realm := &keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test",
						Name:      "realm-cr",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "my-realm",
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(realm).Build()
			},
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test-group",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "realm-cr",
					},
				},
			},
			wantName: "my-realm",
			wantErr:  require.NoError,
		},
		{
			name: "get realm name from ClusterKeycloakRealm",
			client: func(t *testing.T) client.Client {
				clusterRealm := &keycloakApiAlpha.ClusterKeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-realm",
					},
					Spec: keycloakApiAlpha.ClusterKeycloakRealmSpec{
						RealmName: "cluster-my-realm",
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(clusterRealm).Build()
			},
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test-group",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApiAlpha.ClusterKeycloakRealmKind,
						Name: "cluster-realm",
					},
				},
			},
			wantName: "cluster-my-realm",
			wantErr:  require.NoError,
		},
		{
			name: "error when KeycloakRealm not found",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test-group",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "nonexistent",
					},
				},
			},
			wantErr: require.Error,
		},
		{
			name: "error when ClusterKeycloakRealm not found",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test-group",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApiAlpha.ClusterKeycloakRealmKind,
						Name: "nonexistent",
					},
				},
			},
			wantErr: require.Error,
		},
		{
			name: "error on unknown realm kind",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test-group",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: "UnknownKind",
						Name: "some-realm",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unknown realm kind: UnknownKind")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Helper{
				client: tt.client(t),
				scheme: scheme,
			}

			got, err := h.GetRealmNameFromRef(context.Background(), tt.object)
			tt.wantErr(t, err)

			if err == nil {
				assert.Equal(t, tt.wantName, got)
			}
		})
	}
}

func TestHelper_SetKeycloakOwnerRef(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))
	require.NoError(t, keycloakApiAlpha.AddToScheme(scheme))

	type fields struct {
		client         func(t *testing.T) client.Client
		enableOwnerRef bool
	}

	type args struct {
		object ObjectWithKeycloakRef
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr require.ErrorAssertionFunc
		want    func(t *testing.T, k8sCl client.Client)
	}{
		{
			name: "set Keycloak owner reference",
			fields: fields{
				client: func(t *testing.T) client.Client {
					keycloak := &keycloakApi.Keycloak{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "test",
							Name:      "keycloak",
						},
					}
					realm := &keycloakApi.KeycloakRealm{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "test",
							Name:      "test-realm",
						},
						Spec: keycloakApi.KeycloakRealmSpec{
							KeycloakRef: common.KeycloakRef{
								Kind: keycloakApi.KeycloakKind,
								Name: "keycloak",
							},
						},
					}

					return fake.NewClientBuilder().WithScheme(scheme).WithObjects(keycloak, realm).Build()
				},
				enableOwnerRef: true,
			},
			args: args{
				object: &keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       "test",
						Name:            "test-realm",
						ResourceVersion: "999",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						KeycloakRef: common.KeycloakRef{
							Kind: keycloakApi.KeycloakKind,
							Name: "keycloak",
						},
					},
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sCl client.Client) {
				realm := &keycloakApi.KeycloakRealm{}
				err := k8sCl.Get(context.Background(), types.NamespacedName{
					Namespace: "test",
					Name:      "test-realm",
				}, realm)

				require.NoError(t, err)
				require.NotNil(t, metav1.GetControllerOf(realm))
			},
		},
		{
			name: "set ClusterKeycloak owner reference",
			fields: fields{
				client: func(t *testing.T) client.Client {
					clusterKeycloak := &keycloakApiAlpha.ClusterKeycloak{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-keycloak",
						},
					}
					realm := &keycloakApi.KeycloakRealm{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "test",
							Name:      "test-realm",
						},
						Spec: keycloakApi.KeycloakRealmSpec{
							KeycloakRef: common.KeycloakRef{
								Kind: keycloakApiAlpha.ClusterKeycloakKind,
								Name: "cluster-keycloak",
							},
						},
					}

					return fake.NewClientBuilder().WithScheme(scheme).WithObjects(clusterKeycloak, realm).Build()
				},
				enableOwnerRef: true,
			},
			args: args{
				object: &keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       "test",
						Name:            "test-realm",
						ResourceVersion: "999",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						KeycloakRef: common.KeycloakRef{
							Kind: keycloakApiAlpha.ClusterKeycloakKind,
							Name: "cluster-keycloak",
						},
					},
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sCl client.Client) {
				realm := &keycloakApi.KeycloakRealm{}
				err := k8sCl.Get(context.Background(), types.NamespacedName{
					Namespace: "test",
					Name:      "test-realm",
				}, realm)

				require.NoError(t, err)
				require.NotNil(t, metav1.GetControllerOf(realm))
			},
		},
		{
			name: "owner reference not set when enableOwnerRef is false",
			fields: fields{
				client: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().Build()
				},
				enableOwnerRef: false,
			},
			args: args{
				object: &keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test",
						Name:      "test-realm",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						KeycloakRef: common.KeycloakRef{
							Kind: keycloakApi.KeycloakKind,
							Name: "test",
						},
					},
				},
			},
			wantErr: require.NoError,
			want:    func(t *testing.T, k8sCl client.Client) {},
		},
		{
			name: "error when Keycloak not found",
			fields: fields{
				client: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().WithScheme(scheme).Build()
				},
				enableOwnerRef: true,
			},
			args: args{
				object: &keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test",
						Name:      "test-realm",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						KeycloakRef: common.KeycloakRef{
							Kind: keycloakApi.KeycloakKind,
							Name: "nonexistent",
						},
					},
				},
			},
			wantErr: require.Error,
			want:    func(t *testing.T, k8sCl client.Client) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sCl := tt.fields.client(t)
			h := &Helper{
				client:         k8sCl,
				scheme:         scheme,
				enableOwnerRef: tt.fields.enableOwnerRef,
			}
			err := h.SetKeycloakOwnerRef(context.Background(), tt.args.object)

			tt.wantErr(t, err)

			if tt.want != nil {
				tt.want(t, k8sCl)
			}
		})
	}
}
