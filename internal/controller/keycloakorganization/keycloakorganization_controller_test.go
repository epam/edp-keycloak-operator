package keycloakorganization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

func TestReconcileOrganization_applyDefaults(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	newOrg := func(mutate func(*keycloakApi.KeycloakOrganization)) *keycloakApi.KeycloakOrganization {
		org := &keycloakApi.KeycloakOrganization{
			ObjectMeta: metav1.ObjectMeta{Name: "org", Namespace: "ns"},
			Spec: keycloakApi.KeycloakOrganizationSpec{
				Name: "Org", Alias: "org",
				RealmRef: common.RealmRef{Kind: "KeycloakRealm", Name: "realm"},
			},
		}
		mutate(org)

		return org
	}

	tests := []struct {
		name        string
		org         *keycloakApi.KeycloakOrganization
		conflict    bool
		prune       bool
		wantUpdated bool
		wantErr     bool
		wantErrMsg  string
		wantV2      []keycloakApi.OrgDomain
	}{
		{
			name: "legacy domains are copied in order and kept",
			org: newOrg(func(o *keycloakApi.KeycloakOrganization) {
				//nolint:staticcheck // legacy field under test
				o.Spec.Domains = []string{"b.com", "a.com"}
			}),
			wantUpdated: true,
			wantV2:      []keycloakApi.OrgDomain{{Name: "b.com"}, {Name: "a.com"}},
		},
		{
			name: "domainsV2 set: no update",
			org: newOrg(func(o *keycloakApi.KeycloakOrganization) {
				//nolint:staticcheck // legacy field under test
				o.Spec.Domains = []string{"legacy.com"}
				o.Spec.DomainsV2 = []keycloakApi.OrgDomain{{Name: "v2.com"}}
			}),
			wantV2: []keycloakApi.OrgDomain{{Name: "v2.com"}},
		},
		{
			name: "stale cache conflict is returned",
			org: newOrg(func(o *keycloakApi.KeycloakOrganization) {
				//nolint:staticcheck // legacy field under test
				o.Spec.Domains = []string{"a.com"}
			}),
			conflict: true,
			wantErr:  true,
		},
		{
			name: "CRD without domainsV2 prunes the migration",
			org: newOrg(func(o *keycloakApi.KeycloakOrganization) {
				//nolint:staticcheck // legacy field under test
				o.Spec.Domains = []string{"a.com"}
			}),
			prune:      true,
			wantErr:    true,
			wantErrMsg: "domainsV2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.org)

			if tt.conflict {
				builder = builder.WithInterceptorFuncs(interceptor.Funcs{
					Update: func(_ context.Context, _ client.WithWatch, obj client.Object, _ ...client.UpdateOption) error {
						return k8sConflict(obj)
					},
				})
			}

			if tt.prune {
				builder = builder.WithInterceptorFuncs(interceptor.Funcs{
					Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
						obj.(*keycloakApi.KeycloakOrganization).Spec.DomainsV2 = nil

						return c.Update(ctx, obj, opts...)
					},
				})
			}

			k8sClient := builder.Build()
			r := NewReconcileOrganization(k8sClient, nil)

			org := &keycloakApi.KeycloakOrganization{}
			require.NoError(t, k8sClient.Get(context.Background(), types.NamespacedName{Name: "org", Namespace: "ns"}, org))

			updated, err := r.applyDefaults(context.Background(), org)

			if tt.wantErr {
				if tt.conflict {
					require.True(t, k8sErrors.IsConflict(err))
					return
				}

				require.ErrorContains(t, err, tt.wantErrMsg)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantUpdated, updated)

			stored := &keycloakApi.KeycloakOrganization{}
			require.NoError(t, k8sClient.Get(context.Background(), types.NamespacedName{Name: "org", Namespace: "ns"}, stored))
			require.Equal(t, tt.wantV2, stored.Spec.DomainsV2)
		})
	}
}

func k8sConflict(obj client.Object) error {
	return k8sErrors.NewConflict(
		schema.GroupResource{Group: "v1.edp.epam.com", Resource: "keycloakorganizations"}, obj.GetName(), nil)
}

type fakeHelper struct{}

func (fakeHelper) CreateKeycloakClientFromRealmRef(
	context.Context, helper.ObjectWithRealmRef,
) (*keycloakapi.KeycloakClient, error) {
	return &keycloakapi.KeycloakClient{}, nil
}

func (fakeHelper) GetRealmNameFromRef(context.Context, helper.ObjectWithRealmRef) (string, error) {
	return "realm", nil
}

func TestReconcileOrganization_DeletingSkipsMigration(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	now := metav1.Now()
	org := &keycloakApi.KeycloakOrganization{
		ObjectMeta: metav1.ObjectMeta{
			Name: "org", Namespace: "ns", DeletionTimestamp: &now, Finalizers: []string{"other"},
		},
		Spec: keycloakApi.KeycloakOrganizationSpec{
			Name: "Org", Alias: "org",
			//nolint:staticcheck // legacy field under test
			Domains: []string{"a.com"},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(org).Build()
	r := NewReconcileOrganization(k8sClient, fakeHelper{})

	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "org", Namespace: "ns"},
	})
	require.NoError(t, err)

	stored := &keycloakApi.KeycloakOrganization{}
	require.NoError(t, k8sClient.Get(context.Background(), types.NamespacedName{Name: "org", Namespace: "ns"}, stored))
	require.Empty(t, stored.Spec.DomainsV2)
}
