package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakmocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestRemoveOrganization_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		organization   *keycloakApi.KeycloakOrganization
		realmName      string
		keycloakClient func(t *testing.T) keycloakapi.OrganizationsClient
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name: "organization ID not set in status - should skip",
			organization: &keycloakApi.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-org",
				},
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:    "Test Organization",
					Alias:   "test-org",
					Domains: []string{"test.com"},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "",
				},
			},
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				return keycloakmocks.NewMockOrganizationsClient(t)
			},
			wantErr: require.NoError,
		},
		{
			name: "preserve resources on deletion - should skip",
			organization: &keycloakApi.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-org",
					Annotations: map[string]string{
						"edp.epam.com/preserve-resources-on-deletion": "true",
					},
				},
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:    "Test Organization",
					Alias:   "test-org",
					Domains: []string{"test.com"},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				return keycloakmocks.NewMockOrganizationsClient(t)
			},
			wantErr: require.NoError,
		},
		{
			name: "organization deleted successfully",
			organization: &keycloakApi.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-org",
				},
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:    "Test Organization",
					Alias:   "test-org",
					Domains: []string{"test.com"},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakmocks.NewMockOrganizationsClient(t)
				client.On("DeleteOrganization", mock.Anything, "test-realm", "org-123").
					Return((*keycloakapi.Response)(nil), nil).Once()
				return client
			},
			wantErr: require.NoError,
		},
		{
			name: "organization not found - should skip",
			organization: &keycloakApi.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-org",
				},
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:    "Test Organization",
					Alias:   "test-org",
					Domains: []string{"test.com"},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakmocks.NewMockOrganizationsClient(t)
				client.On("DeleteOrganization", mock.Anything, "test-realm", "org-123").
					Return((*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: 404, Message: "organization not found"}).Once()
				return client
			},
			wantErr: require.NoError,
		},
		{
			name: "delete organization fails with error",
			organization: &keycloakApi.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-org",
				},
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:    "Test Organization",
					Alias:   "test-org",
					Domains: []string{"test.com"},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakmocks.NewMockOrganizationsClient(t)
				client.On("DeleteOrganization", mock.Anything, "test-realm", "org-123").
					Return((*keycloakapi.Response)(nil), errors.New("network error")).Once()
				return client
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgClient := tt.keycloakClient(t)
			kc := &keycloakapi.APIClient{}
			kc.Organizations = orgClient

			handler := NewRemoveOrganization(kc)
			err := handler.ServeRequest(context.Background(), tt.organization, tt.realmName)

			tt.wantErr(t, err)
		})
	}
}
