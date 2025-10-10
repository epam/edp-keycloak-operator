package keycloakrealmrolebatch

import (
	"context"
	"testing"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_terminator_DeleteResource(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	tests := []struct {
		name                        string
		k8sClient                   func(t *testing.T) client.Client
		childRoles                  []keycloakApi.KeycloakRealmRole
		preserveResourcesOnDeletion bool
		wantErr                     assert.ErrorAssertionFunc
	}{
		{
			name: "role does not exist",
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
			},
			childRoles: []keycloakApi.KeycloakRealmRole{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "role"},
				},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := makeTerminator(tt.k8sClient(t), tt.childRoles, tt.preserveResourcesOnDeletion)
			gotErr := te.DeleteResource(context.Background())
			tt.wantErr(t, gotErr)
		})
	}
}
