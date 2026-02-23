package keycloakrealm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func Test_terminator_DeleteResource(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	tests := []struct {
		name                        string
		realmName                   string
		realmClient                 func(t *testing.T) keycloakv2.RealmClient
		preserveResourcesOnDeletion bool
		wantErr                     assert.ErrorAssertionFunc
	}{
		{
			name:      "realm does not exist",
			realmName: "realm",
			realmClient: func(t *testing.T) keycloakv2.RealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.On("DeleteRealm", mock.Anything, "realm").
					Return(nil, &keycloakv2.ApiError{Code: 404})
				return m
			},
			preserveResourcesOnDeletion: false,
			wantErr:                     assert.NoError,
		},
		{
			name:      "realm deleted successfully",
			realmName: "realm",
			realmClient: func(t *testing.T) keycloakv2.RealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.On("DeleteRealm", mock.Anything, "realm").Return(nil, nil)
				return m
			},
			preserveResourcesOnDeletion: false,
			wantErr:                     assert.NoError,
		},
		{
			name:      "preserve resources on deletion — skip",
			realmName: "realm",
			realmClient: func(t *testing.T) keycloakv2.RealmClient {
				return v2mocks.NewMockRealmClient(t)
			},
			preserveResourcesOnDeletion: true,
			wantErr:                     assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := makeTerminator(tt.realmName, tt.realmClient(t), tt.preserveResourcesOnDeletion)
			gotErr := te.DeleteResource(context.Background())
			tt.wantErr(t, gotErr)
		})
	}
}
