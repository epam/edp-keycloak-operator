package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestRemoveOrganization_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		organization   *keycloakApi.KeycloakOrganization
		realm          *gocloak.RealmRepresentation
		keycloakClient func(t *testing.T) keycloak.Client
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
					OrganizationID: "", // Empty organization ID
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				return mocks.NewMockClient(t)
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
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				return mocks.NewMockClient(t)
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
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("DeleteOrganization", mock.Anything, "test-realm", "org-123").
					Return(nil).Once()
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
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("DeleteOrganization", mock.Anything, "test-realm", "org-123").
					Return(adapter.NotFoundError("organization not found")).Once()
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
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("DeleteOrganization", mock.Anything, "test-realm", "org-123").
					Return(errors.New("network error")).Once()
				return client
			},
			wantErr: require.Error,
		},
		{
			name: "organization with nil realm - should handle gracefully",
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
			realm: &gocloak.RealmRepresentation{
				Realm: nil, // Nil realm
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("DeleteOrganization", mock.Anything, "", "org-123").
					Return(nil).Once()
				return client
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewRemoveOrganization(tt.keycloakClient(t))
			err := handler.ServeRequest(context.Background(), tt.organization, tt.realm)

			tt.wantErr(t, err)
		})
	}
}
