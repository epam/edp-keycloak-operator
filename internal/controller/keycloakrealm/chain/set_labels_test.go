package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

func TestSetLabels_ServeRequest(t *testing.T) {
	t.Parallel()

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))

	tests := []struct {
		name          string
		realm         *keycloakApi.KeycloakRealm
		wantUnchanged bool
	}{
		{
			name: "label already correct — no k8s update",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "realm1",
					Namespace: "default",
					Labels:    map[string]string{TargetRealmLabel: "realm1"},
				},
				Spec: keycloakApi.KeycloakRealmSpec{RealmName: "realm1"},
			},
			wantUnchanged: true,
		},
		{
			name: "label missing — k8s update applied",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "realm2",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{RealmName: "realm2"},
			},
			wantUnchanged: false,
		},
		{
			name: "label stale — k8s update applied",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "realm3",
					Namespace: "default",
					Labels:    map[string]string{TargetRealmLabel: "old-realm-name"},
				},
				Spec: keycloakApi.KeycloakRealmSpec{RealmName: "realm3"},
			},
			wantUnchanged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			k8sClient := fake.NewClientBuilder().WithScheme(s).WithObjects(tt.realm.DeepCopy()).Build()

			fetched := &keycloakApi.KeycloakRealm{}
			require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(tt.realm), fetched))
			initialResourceVersion := fetched.ResourceVersion

			h := SetLabels{client: k8sClient}
			err := h.ServeRequest(ctx, fetched, &keycloakapi.KeycloakClient{})
			require.NoError(t, err)
			require.Equal(t, tt.realm.Spec.RealmName, fetched.Labels[TargetRealmLabel])

			stored := &keycloakApi.KeycloakRealm{}
			require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(tt.realm), stored))

			if tt.wantUnchanged {
				require.Equal(t, initialResourceVersion, stored.ResourceVersion)
			} else {
				require.NotEqual(t, initialResourceVersion, stored.ResourceVersion)
				require.Equal(t, tt.realm.Spec.RealmName, stored.Labels[TargetRealmLabel])
			}
		})
	}
}
