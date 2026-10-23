package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapimocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

type serveRequestCase struct {
	name           string
	organization   *keycloakApi.KeycloakOrganization
	realmName      string
	keycloakClient func(t *testing.T) keycloakapi.OrganizationsClient
	wantErr        require.ErrorAssertionFunc
	expectedOrgID  string
}

func TestCreateOrganization_ServeRequest(t *testing.T) {
	tests := []serveRequestCase{
		{
			name: "successfully create new organization",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:        "Test Organization",
					Alias:       "test-org",
					Description: "Test organization",
					RedirectURL: "https://example.com/redirect",
					DomainsV2:   domainsV2("example.com", "test.com"),
					Attributes: map[string][]string{
						"attr1": {"value1"},
						"attr2": {"value2", "value3"},
					},
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				// First call: GetOrganizationByAlias returns not found
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "test-org").
					Return((*keycloakapi.OrganizationRepresentation)(nil), (*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: 404, Message: "organization not found"}).Once()

				// Second call: CreateOrganization succeeds
				client.On("CreateOrganization", mock.Anything, testRealmName, mock.MatchedBy(func(org keycloakapi.OrganizationRepresentation) bool {
					return ptr.Deref(org.Name, "") == "Test Organization" &&
						ptr.Deref(org.Alias, "") == "test-org" &&
						ptr.Deref(org.Description, "") == "Test organization" &&
						ptr.Deref(org.RedirectUrl, "") == "https://example.com/redirect" &&
						len(ptr.Deref(org.Domains, nil)) == 2 &&
						len(ptr.Deref(org.Attributes, nil)) == 2
				})).Return((*keycloakapi.Response)(nil), nil).Once()

				// Third call: GetOrganizationByAlias returns the created organization
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "test-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To(testOrgID),
						Alias: ptr.To("test-org"),
					}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: testOrgID,
		},
		{
			name: "successfully update existing organization",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:        "Updated Organization",
					Alias:       "existing-org",
					Description: "Updated organization",
					RedirectURL: "https://updated.com/redirect",
					DomainsV2:   domainsV2("updated.com"),
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				// Fetched organization has none of the spec's fields set: drift forces the update.
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "existing-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("existing-org-456"),
						Alias: ptr.To("existing-org"),
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("UpdateOrganization", mock.Anything, testRealmName, "existing-org-456", mock.MatchedBy(func(org keycloakapi.OrganizationRepresentation) bool {
					return ptr.Deref(org.Name, "") == "Updated Organization" &&
						ptr.Deref(org.Alias, "") == "existing-org" &&
						ptr.Deref(org.Description, "") == "Updated organization" &&
						ptr.Deref(org.RedirectUrl, "") == "https://updated.com/redirect" &&
						len(ptr.Deref(org.Domains, nil)) == 1
				})).Return((*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "existing-org-456",
		},
		{
			name: "no update when existing organization matches spec",
			organization: &keycloakApi.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{Generation: 3},
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:        "In Sync Org",
					Alias:       "in-sync-org",
					Description: "In sync description",
					RedirectURL: "https://in-sync.com/redirect",
					DomainsV2:   domainsV2("a.com", "b.com"),
					Attributes: map[string][]string{
						"dept": {"eng", "qa"},
					},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					ObservedGeneration: 3,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				// GetOrganizationByAlias is backed by the list endpoint: it never returns
				// Attributes. Domains are shuffled and Enabled/Members are server-populated
				// fields the spec never sets: none of that should trigger an update.
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "in-sync-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:          ptr.To("in-sync-id"),
						Name:        ptr.To("In Sync Org"),
						Alias:       ptr.To("in-sync-org"),
						Description: ptr.To("In sync description"),
						RedirectUrl: ptr.To("https://in-sync.com/redirect"),
						Enabled:     ptr.To(true),
						Domains: &[]keycloakapi.OrganizationDomainRepresentation{
							{Name: ptr.To("b.com"), Verified: ptr.To(true)},
							{Name: ptr.To("a.com")},
						},
						Members: &[]keycloakapi.MemberRepresentation{{}},
					}, (*keycloakapi.Response)(nil), nil).Once()

				// Brief fields already match, and the spec declares attributes: the full
				// representation is fetched by ID to compare them, with values shuffled.
				client.On("GetOrganization", mock.Anything, testRealmName, "in-sync-id").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("in-sync-id"),
						Alias: ptr.To("in-sync-org"),
						Attributes: &map[string][]string{
							"dept": {"qa", "eng"},
						},
					}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "in-sync-id",
		},
		{
			name: "brief field drift forces update without fetching the full representation",
			organization: &keycloakApi.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:        "Brief Drift Org",
					Alias:       "brief-drift-org",
					Description: "New description",
					DomainsV2:   domainsV2("a.com"),
					Attributes: map[string][]string{
						"dept": {"eng"},
					},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					ObservedGeneration: 1,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				// Description differs: the update fires from the brief comparison alone.
				// GetOrganization (by ID) is not stubbed; an unexpected call fails the test.
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "brief-drift-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:          ptr.To("brief-drift-id"),
						Name:        ptr.To("Brief Drift Org"),
						Alias:       ptr.To("brief-drift-org"),
						Description: ptr.To("Old description"),
						Domains: &[]keycloakapi.OrganizationDomainRepresentation{
							{Name: ptr.To("a.com")},
						},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("UpdateOrganization", mock.Anything, testRealmName, "brief-drift-id", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "brief-drift-id",
		},
		{
			name: "spec without attributes needs no full representation fetch",
			organization: &keycloakApi.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "No Attrs Org",
					Alias:     "no-attrs-org",
					DomainsV2: domainsV2("a.com"),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					ObservedGeneration: 1,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				// Brief fields already match and the spec declares no attributes: neither
				// GetOrganization (by ID) nor UpdateOrganization should be called.
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "no-attrs-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("no-attrs-id"),
						Name:  ptr.To("No Attrs Org"),
						Alias: ptr.To("no-attrs-org"),
						Domains: &[]keycloakapi.OrganizationDomainRepresentation{
							{Name: ptr.To("a.com")},
						},
					}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "no-attrs-id",
		},
		{
			name: "GetOrganization by id error is propagated",
			organization: &keycloakApi.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "By Id Error Org",
					Alias:     "by-id-error-org",
					DomainsV2: domainsV2("a.com"),
					Attributes: map[string][]string{
						"dept": {"eng"},
					},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					ObservedGeneration: 1,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "by-id-error-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("by-id-error-id"),
						Name:  ptr.To("By Id Error Org"),
						Alias: ptr.To("by-id-error-org"),
						Domains: &[]keycloakapi.OrganizationDomainRepresentation{
							{Name: ptr.To("a.com")},
						},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("GetOrganization", mock.Anything, testRealmName, "by-id-error-id").
					Return((*keycloakapi.OrganizationRepresentation)(nil), (*keycloakapi.Response)(nil), errors.New("network error")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "generation bump forces update even when in sync and keeps the disabled state",
			organization: &keycloakApi.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{Generation: 4},
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "In Sync Org",
					Alias:     "in-sync-org-gen",
					DomainsV2: domainsV2("a.com"),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					ObservedGeneration: 3,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "in-sync-org-gen").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:      ptr.To("gen-bump-id"),
						Name:    ptr.To("In Sync Org"),
						Alias:   ptr.To("in-sync-org-gen"),
						Enabled: ptr.To(false),
						Domains: &[]keycloakapi.OrganizationDomainRepresentation{
							{Name: ptr.To("a.com")},
						},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("UpdateOrganization", mock.Anything, testRealmName, "gen-bump-id",
					mock.MatchedBy(func(org keycloakapi.OrganizationRepresentation) bool {
						return org.Enabled != nil && !*org.Enabled
					})).
					Return((*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "gen-bump-id",
		},
		{
			name: "domain removed from spec forces update",
			organization: &keycloakApi.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "Domain Drop Org",
					Alias:     "domain-drop-org",
					DomainsV2: domainsV2("a.com"),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					ObservedGeneration: 1,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "domain-drop-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("domain-drop-id"),
						Name:  ptr.To("Domain Drop Org"),
						Alias: ptr.To("domain-drop-org"),
						Domains: &[]keycloakapi.OrganizationDomainRepresentation{
							{Name: ptr.To("a.com")},
							{Name: ptr.To("b.com")},
						},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("UpdateOrganization", mock.Anything, testRealmName, "domain-drop-id", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "domain-drop-id",
		},
		{
			name: "duplicated domain in spec is in sync with the deduped existing domain",
			organization: &keycloakApi.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "Dup Domain Org",
					Alias:     "dup-domain-org",
					DomainsV2: domainsV2("a.com", "a.com"),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					ObservedGeneration: 1,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				// Keycloak dedups domains server-side: only one "a.com" comes back.
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "dup-domain-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("dup-domain-id"),
						Name:  ptr.To("Dup Domain Org"),
						Alias: ptr.To("dup-domain-org"),
						Domains: &[]keycloakapi.OrganizationDomainRepresentation{
							{Name: ptr.To("a.com")},
						},
					}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "dup-domain-id",
		},
		{
			name: "attribute drift discovered via the by-id fetch forces update",
			organization: &keycloakApi.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "Attr Change Org",
					Alias:     "attr-change-org",
					DomainsV2: domainsV2("a.com"),
					Attributes: map[string][]string{
						"dept": {"eng"},
					},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					ObservedGeneration: 1,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				// Brief fields match, so the full representation is fetched by ID; its
				// attributes differ from the spec.
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "attr-change-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("attr-change-id"),
						Name:  ptr.To("Attr Change Org"),
						Alias: ptr.To("attr-change-org"),
						Domains: &[]keycloakapi.OrganizationDomainRepresentation{
							{Name: ptr.To("a.com")},
						},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("GetOrganization", mock.Anything, testRealmName, "attr-change-id").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("attr-change-id"),
						Alias: ptr.To("attr-change-org"),
						Attributes: &map[string][]string{
							"dept": {"qa"},
						},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("UpdateOrganization", mock.Anything, testRealmName, "attr-change-id", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "attr-change-id",
		},
		{
			name: "error when GetOrganizationByAlias fails with non-not-found error",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "Test Organization",
					Alias:     "test-org",
					DomainsV2: domainsV2("example.com"),
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "test-org").
					Return((*keycloakapi.OrganizationRepresentation)(nil), (*keycloakapi.Response)(nil), errors.New("network error")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "error when CreateOrganization fails",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "Test Organization",
					Alias:     "test-org",
					DomainsV2: domainsV2("example.com"),
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				// First call: GetOrganizationByAlias returns not found
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "test-org").
					Return((*keycloakapi.OrganizationRepresentation)(nil), (*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: 404, Message: "organization not found"}).Once()

				// Second call: CreateOrganization fails
				client.On("CreateOrganization", mock.Anything, testRealmName, mock.Anything).
					Return((*keycloakapi.Response)(nil), errors.New("creation failed")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "error when UpdateOrganization fails",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "Updated Organization",
					Alias:     "existing-org",
					DomainsV2: domainsV2("example.com"),
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				// First call: GetOrganizationByAlias returns existing organization
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "existing-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("existing-org-456"),
						Alias: ptr.To("existing-org"),
					}, (*keycloakapi.Response)(nil), nil).Once()

				// Second call: UpdateOrganization fails
				client.On("UpdateOrganization", mock.Anything, testRealmName, "existing-org-456", mock.Anything).
					Return((*keycloakapi.Response)(nil), errors.New("update failed")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "error when GetOrganizationByAlias fails after creation",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "Test Organization",
					Alias:     "test-org",
					DomainsV2: domainsV2("example.com"),
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				// First call: GetOrganizationByAlias returns not found
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "test-org").
					Return((*keycloakapi.OrganizationRepresentation)(nil), (*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: 404, Message: "organization not found"}).Once()

				// Second call: CreateOrganization succeeds
				client.On("CreateOrganization", mock.Anything, testRealmName, mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Once()

				// Third call: GetOrganizationByAlias fails after creation
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "test-org").
					Return((*keycloakapi.OrganizationRepresentation)(nil), (*keycloakapi.Response)(nil), errors.New("failed to retrieve created organization")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "organization with minimal required fields",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "Minimal Org",
					Alias:     "minimal-org",
					DomainsV2: domainsV2("minimal.com"),
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				// First call: GetOrganizationByAlias returns not found
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "minimal-org").
					Return((*keycloakapi.OrganizationRepresentation)(nil), (*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: 404, Message: "organization not found"}).Once()

				// Second call: CreateOrganization succeeds
				client.On("CreateOrganization", mock.Anything, testRealmName, mock.MatchedBy(func(org keycloakapi.OrganizationRepresentation) bool {
					return ptr.Deref(org.Name, "") == "Minimal Org" &&
						ptr.Deref(org.Alias, "") == "minimal-org" &&
						len(ptr.Deref(org.Domains, nil)) == 1
				})).Return((*keycloakapi.Response)(nil), nil).Once()

				// Third call: GetOrganizationByAlias returns the created organization
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "minimal-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("minimal-org-789"),
						Alias: ptr.To("minimal-org"),
					}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "minimal-org-789",
		},
		{
			name: "organization with existing ID in status",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "Existing Org",
					Alias:     "existing-org-with-id",
					DomainsV2: domainsV2("existing.com"),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "existing-id-123",
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				// First call: GetOrganizationByAlias returns existing organization
				client.On("GetOrganizationByAlias", mock.Anything, testRealmName, "existing-org-with-id").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("existing-id-123"),
						Alias: ptr.To("existing-org-with-id"),
					}, (*keycloakapi.Response)(nil), nil).Once()

				// Second call: UpdateOrganization succeeds
				client.On("UpdateOrganization", mock.Anything, testRealmName, "existing-id-123", mock.MatchedBy(func(org keycloakapi.OrganizationRepresentation) bool {
					return ptr.Deref(org.Name, "") == "Existing Org" &&
						ptr.Deref(org.Alias, "") == "existing-org-with-id"
				})).Return((*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "existing-id-123",
		},
		domainSettingsCase("update sends stored domains as fetched", true,
			[]string{"a.example.test", "b.example.test"},
			[]keycloakapi.OrganizationDomainRepresentation{
				{Name: ptr.To("a.example.test"), Verified: ptr.To(true)},
				{Name: ptr.To("b.example.test")},
			},
			&[]keycloakapi.OrganizationDomainRepresentation{
				{Name: ptr.To("a.example.test"), Verified: ptr.To(true)},
				{Name: ptr.To("b.example.test")},
			}),
		domainSettingsCase("added domain is sent by name only", true,
			[]string{"a.example.test", "b.example.test", "c.example.test"},
			[]keycloakapi.OrganizationDomainRepresentation{
				{Name: ptr.To("a.example.test"), Verified: ptr.To(true)},
				{Name: ptr.To("b.example.test")},
			},
			&[]keycloakapi.OrganizationDomainRepresentation{
				{Name: ptr.To("a.example.test"), Verified: ptr.To(true)},
				{Name: ptr.To("b.example.test")},
				{Name: ptr.To("c.example.test")},
			}),
		domainSettingsCase("removed domain is not sent", true,
			[]string{"a.example.test"},
			[]keycloakapi.OrganizationDomainRepresentation{
				{Name: ptr.To("a.example.test"), Verified: ptr.To(true)},
				{Name: ptr.To("b.example.test"), Verified: ptr.To(true)},
			},
			&[]keycloakapi.OrganizationDomainRepresentation{
				{Name: ptr.To("a.example.test"), Verified: ptr.To(true)},
			}),
		domainSettingsCase("spec domain in mixed case or padded matches the stored domain", true,
			[]string{"A.Example.test", " b.example.test"},
			[]keycloakapi.OrganizationDomainRepresentation{
				{Name: ptr.To("a.example.test"), Verified: ptr.To(true)},
				{Name: ptr.To("b.example.test"), Verified: ptr.To(true)},
			},
			&[]keycloakapi.OrganizationDomainRepresentation{
				{Name: ptr.To("a.example.test"), Verified: ptr.To(true)},
				{Name: ptr.To("b.example.test"), Verified: ptr.To(true)},
			}),
		domainSettingsCase("duplicated spec domain sends the fetched copy", true,
			[]string{"a.example.test", "A.example.test"},
			[]keycloakapi.OrganizationDomainRepresentation{
				{Name: ptr.To("a.example.test"), Verified: ptr.To(true)},
			},
			&[]keycloakapi.OrganizationDomainRepresentation{
				{Name: ptr.To("a.example.test"), Verified: ptr.To(true)},
				{Name: ptr.To("a.example.test"), Verified: ptr.To(true)},
			}),
		domainSettingsCase("fetched organization without domains", true,
			[]string{"a.example.test"},
			nil,
			&[]keycloakapi.OrganizationDomainRepresentation{{Name: ptr.To("a.example.test")}}),
		domainSettingsCase("mixed-case spec domain in sync with the stored domain is not written", false,
			[]string{" A.Example.test"},
			[]keycloakapi.OrganizationDomainRepresentation{{Name: ptr.To("a.example.test")}},
			nil),
		domainSettingsCase("case-variant duplicates in spec are in sync with the stored domain", false,
			[]string{"a.example.test", "A.example.test"},
			[]keycloakapi.OrganizationDomainRepresentation{{Name: ptr.To("a.example.test")}},
			nil),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgClient := tt.keycloakClient(t)
			kc := &keycloakapi.KeycloakClient{}
			kc.Organizations = orgClient

			handler := NewCreateOrganization(kc)
			err := handler.ServeRequest(context.Background(), tt.organization, tt.realmName)

			tt.wantErr(t, err)

			if err == nil {
				require.Equal(t, tt.expectedOrgID, tt.organization.Status.OrganizationID)
			}
		})
	}
}

func TestSpecToOrganizationRepresentation(t *testing.T) {
	tests := []struct {
		name   string
		org    *keycloakApi.KeycloakOrganization
		verify func(t *testing.T, rep keycloakapi.OrganizationRepresentation)
	}{
		{
			name: "full spec with all fields",
			org: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:        "Test Organization",
					Alias:       "test-org",
					Description: "A description",
					RedirectURL: "https://example.com/redirect",
					DomainsV2:   domainsV2("example.com", "test.com"),
					Attributes: map[string][]string{
						"dept": {"eng"},
						"loc":  {"us", "eu"},
					},
				},
			},
			verify: func(t *testing.T, rep keycloakapi.OrganizationRepresentation) {
				t.Helper()

				require.Equal(t, "Test Organization", ptr.Deref(rep.Name, ""))
				require.Equal(t, "test-org", ptr.Deref(rep.Alias, ""))
				require.Equal(t, "A description", ptr.Deref(rep.Description, ""))
				require.Equal(t, "https://example.com/redirect", ptr.Deref(rep.RedirectUrl, ""))

				require.NotNil(t, rep.Domains)
				require.Len(t, *rep.Domains, 2)

				domainNames := make([]string, len(*rep.Domains))
				for i, d := range *rep.Domains {
					domainNames[i] = ptr.Deref(d.Name, "")
				}

				require.ElementsMatch(t, []string{"example.com", "test.com"}, domainNames)

				require.NotNil(t, rep.Attributes)
				attrs := *rep.Attributes
				require.Equal(t, []string{"eng"}, attrs["dept"])
				require.ElementsMatch(t, []string{"us", "eu"}, attrs["loc"])
			},
		},
		{
			name: "minimal spec - optional fields absent",
			org: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "Minimal Org",
					Alias:     "minimal-org",
					DomainsV2: domainsV2("minimal.com"),
				},
			},
			verify: func(t *testing.T, rep keycloakapi.OrganizationRepresentation) {
				t.Helper()

				require.Equal(t, "Minimal Org", ptr.Deref(rep.Name, ""))
				require.Equal(t, "minimal-org", ptr.Deref(rep.Alias, ""))
				require.Equal(t, "", ptr.Deref(rep.Description, ""))
				require.Equal(t, "", ptr.Deref(rep.RedirectUrl, ""))
				require.Nil(t, rep.Attributes)
				require.NotNil(t, rep.Domains)
				require.Len(t, *rep.Domains, 1)
				require.Equal(t, "minimal.com", ptr.Deref((*rep.Domains)[0].Name, ""))
			},
		},
		{
			name: "names read from domainsV2; legacy domains ignored",
			org: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "V2 Org",
					Alias:     "v2-org",
					DomainsV2: domainsV2("v2.com"),
					//nolint:staticcheck // legacy field under test
					Domains: []string{"legacy.com"},
				},
			},
			verify: func(t *testing.T, rep keycloakapi.OrganizationRepresentation) {
				t.Helper()

				require.NotNil(t, rep.Domains)
				require.Len(t, *rep.Domains, 1)
				require.Equal(t, "v2.com", ptr.Deref((*rep.Domains)[0].Name, ""))
			},
		},
		{
			name: "nil domains",
			org: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:      "No Domains Org",
					Alias:     "no-domains-org",
					DomainsV2: nil,
				},
			},
			verify: func(t *testing.T, rep keycloakapi.OrganizationRepresentation) {
				t.Helper()
				require.Nil(t, rep.Domains)
			},
		},
		{
			name: "nil attributes",
			org: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:       "No Attrs Org",
					Alias:      "no-attrs-org",
					DomainsV2:  domainsV2("no-attrs.com"),
					Attributes: nil,
				},
			},
			verify: func(t *testing.T, rep keycloakapi.OrganizationRepresentation) {
				t.Helper()
				require.Nil(t, rep.Attributes)
			},
		},
		{
			name: "empty attributes map",
			org: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:       "Empty Attrs Org",
					Alias:      "empty-attrs-org",
					DomainsV2:  domainsV2("empty-attrs.com"),
					Attributes: map[string][]string{},
				},
			},
			verify: func(t *testing.T, rep keycloakapi.OrganizationRepresentation) {
				t.Helper()
				require.Nil(t, rep.Attributes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep := specToOrganizationRepresentation(tt.org)
			tt.verify(t, rep)
		})
	}
}

// domainSettingsCase builds an update-path case for an existing organization. forced bumps
// the generation past the observed one. A nil wantDomains means no UpdateOrganization call is expected.
func domainSettingsCase(
	name string,
	forced bool,
	specDomains []string,
	stored []keycloakapi.OrganizationDomainRepresentation,
	wantDomains *[]keycloakapi.OrganizationDomainRepresentation,
) serveRequestCase {
	const (
		alias = "domain-settings-org"
		orgID = "domain-settings-id"
	)

	generation := int64(1)
	if forced {
		generation = 2
	}

	return serveRequestCase{
		name: name,
		organization: &keycloakApi.KeycloakOrganization{
			ObjectMeta: metav1.ObjectMeta{Generation: generation},
			Spec: keycloakApi.KeycloakOrganizationSpec{
				Name:      "Domain Settings Org",
				Alias:     alias,
				DomainsV2: domainsV2(specDomains...),
			},
			Status: keycloakApi.KeycloakOrganizationStatus{ObservedGeneration: 1},
		},
		realmName: testRealmName,
		keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
			client := keycloakapimocks.NewMockOrganizationsClient(t)

			existing := &keycloakapi.OrganizationRepresentation{
				Id:    ptr.To(orgID),
				Name:  ptr.To("Domain Settings Org"),
				Alias: ptr.To(alias),
			}
			if stored != nil {
				existing.Domains = &stored
			}

			client.On("GetOrganizationByAlias", mock.Anything, testRealmName, alias).
				Return(existing, (*keycloakapi.Response)(nil), nil).Once()

			if wantDomains != nil {
				client.On("UpdateOrganization", mock.Anything, testRealmName, orgID, mock.Anything).
					Run(func(args mock.Arguments) {
						sent, ok := args.Get(3).(keycloakapi.OrganizationRepresentation)
						require.True(t, ok)
						assert.Equal(t, wantDomains, sent.Domains)
					}).
					Return((*keycloakapi.Response)(nil), nil).Once()
			}

			return client
		},
		wantErr:       require.NoError,
		expectedOrgID: orgID,
	}
}

func TestPreserveDomainSettings(t *testing.T) {
	t.Parallel()

	verifiedA := keycloakapi.OrganizationDomainRepresentation{Name: ptr.To("a.example.test"), Verified: ptr.To(true)}

	tests := []struct {
		name    string
		current *[]keycloakapi.OrganizationDomainRepresentation
		desired []string
		want    *[]keycloakapi.OrganizationDomainRepresentation
	}{
		{name: "empty desired", current: &[]keycloakapi.OrganizationDomainRepresentation{verifiedA}, desired: nil, want: nil},
		{
			name: "fetched element without name is skipped",
			current: &[]keycloakapi.OrganizationDomainRepresentation{
				{Verified: ptr.To(true)},
			},
			desired: []string{"a.example.test"},
			want:    &[]keycloakapi.OrganizationDomainRepresentation{{Name: ptr.To("a.example.test")}},
		},
		{
			name:    "spec order is kept",
			current: &[]keycloakapi.OrganizationDomainRepresentation{verifiedA, {Name: ptr.To("b.example.test")}},
			desired: []string{"b.example.test", "a.example.test"},
			want: &[]keycloakapi.OrganizationDomainRepresentation{
				{Name: ptr.To("b.example.test")},
				verifiedA,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, preserveDomainSettings(tt.current, tt.desired))
		})
	}
}
