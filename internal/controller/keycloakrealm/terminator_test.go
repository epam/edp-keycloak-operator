package keycloakrealm

import (
	"context"
	"testing"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

func Test_terminator_DeleteResource(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	tests := []struct {
		name                        string
		realmName                   string
		kClient                     func(t *testing.T) keycloak.Client
		preserveResourcesOnDeletion bool
		wantErr                     assert.ErrorAssertionFunc
	}{
		{
			name:      "realm does not exist",
			realmName: "realm",
			kClient: func(t *testing.T) keycloak.Client {
				m := mocks.NewMockClient(t)

				m.On("DeleteRealm", testifymock.Anything, "realm").Return(adapter.NotFoundError("not found"))

				return m
			},
			preserveResourcesOnDeletion: false,
			wantErr:                     assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := makeTerminator(tt.realmName, tt.kClient(t), tt.preserveResourcesOnDeletion)
			gotErr := te.DeleteResource(context.Background())
			tt.wantErr(t, gotErr)
		})
	}
}
