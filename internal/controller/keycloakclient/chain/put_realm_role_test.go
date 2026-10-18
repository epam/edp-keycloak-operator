package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapiMocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func roleNotFound(m *keycloakapiMocks.MockRolesClient, name string) {
	m.On("GetRealmRole", mock.Anything, testRealmName, name).
		Return((*keycloakapi.RoleRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).Once()
}

func roleFound(m *keycloakapiMocks.MockRolesClient, name, id string) {
	m.On("GetRealmRole", mock.Anything, testRealmName, name).
		Return(&keycloakapi.RoleRepresentation{Id: ptr.To(id), Name: ptr.To(name)}, (*keycloakapi.Response)(nil), nil).Once()
}

func createsRole(m *keycloakapiMocks.MockRolesClient, name string, err error) {
	m.On("CreateRealmRole", mock.Anything, testRealmName, mock.MatchedBy(func(r keycloakapi.RoleRepresentation) bool {
		return r.Name != nil && *r.Name == name && r.Composite == nil
	})).Return((*keycloakapi.Response)(nil), err).Once()
}

// addsToComposite expects child to be added into parent, not parent into child.
func addsToComposite(m *keycloakapiMocks.MockRolesClient, parent, child, childID string, err error) {
	m.On("AddRealmRoleComposites", mock.Anything, testRealmName, parent,
		mock.MatchedBy(func(roles []keycloakapi.RoleRepresentation) bool {
			return len(roles) == 1 && roles[0].Id != nil && *roles[0].Id == childID &&
				roles[0].Name != nil && *roles[0].Name == child
		})).Return((*keycloakapi.Response)(nil), err).Once()
}

//nolint:staticcheck // covers the deprecated realmRoles field
func TestPutRealmRole_Serve(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name            string
		realmRoles      *[]keycloakApi.RealmRole
		mockSetup       func(*keycloakapiMocks.MockRolesClient)
		expectedError   []string
		unexpectedError string
	}{
		{
			name:       "success - created role is added into its composite",
			realmRoles: &[]keycloakApi.RealmRole{{Name: "app-admin", Composite: "administrator"}},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "app-admin")
				roleFound(m, "administrator", "admin-id")
				createsRole(m, "app-admin", nil)
				roleFound(m, "app-admin", "app-admin-id")
				addsToComposite(m, "administrator", "app-admin", "app-admin-id", nil)
			},
		},
		{
			name:       "error - missing composite fails before the role is created",
			realmRoles: &[]keycloakApi.RealmRole{{Name: "app-admin", Composite: "ghost"}},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "app-admin")
				roleNotFound(m, "ghost")
			},
			expectedError: []string{`composite realm roles "ghost" of "app-admin" not found in realm "` + testRealmName + `"; create them with a KeycloakRealmRole`},
		},
		{
			name:       "error - composite lookup fails",
			realmRoles: &[]keycloakApi.RealmRole{{Name: "app-admin", Composite: "administrator"}},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "app-admin")
				m.On("GetRealmRole", mock.Anything, testRealmName, "administrator").
					Return((*keycloakapi.RoleRepresentation)(nil), (*keycloakapi.Response)(nil), errors.New("keycloak api error")).Once()
			},
			expectedError: []string{"error getting composite realm role administrator"},
		},
		{
			name:       "error - failed composite add deletes the created role",
			realmRoles: &[]keycloakApi.RealmRole{{Name: "app-admin", Composite: "administrator"}},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "app-admin")
				roleFound(m, "administrator", "admin-id")
				createsRole(m, "app-admin", nil)
				roleFound(m, "app-admin", "app-admin-id")
				addsToComposite(m, "administrator", "app-admin", "app-admin-id", errors.New("add failed"))
				m.On("DeleteRealmRole", mock.Anything, testRealmName, "app-admin").Return((*keycloakapi.Response)(nil), nil).Once()
			},
			expectedError: []string{"error adding realm role app-admin to composite realm role administrator"},
		},
		{
			name:       "error - failed rollback is reported with the add error",
			realmRoles: &[]keycloakApi.RealmRole{{Name: "app-admin", Composite: "administrator"}},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "app-admin")
				roleFound(m, "administrator", "admin-id")
				createsRole(m, "app-admin", nil)
				roleFound(m, "app-admin", "app-admin-id")
				addsToComposite(m, "administrator", "app-admin", "app-admin-id", errors.New("add failed"))
				m.On("DeleteRealmRole", mock.Anything, testRealmName, "app-admin").
					Return((*keycloakapi.Response)(nil), errors.New("delete failed")).Once()
			},
			expectedError: []string{
				"error adding realm role app-admin to composite realm role administrator",
				"unable to delete realm role app-admin after failed composite add",
			},
		},
		{
			name:       "error - created role without id is deleted",
			realmRoles: &[]keycloakApi.RealmRole{{Name: "app-admin", Composite: "administrator"}},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "app-admin")
				roleFound(m, "administrator", "admin-id")
				createsRole(m, "app-admin", nil)
				m.On("GetRealmRole", mock.Anything, testRealmName, "app-admin").
					Return(&keycloakapi.RoleRepresentation{Name: ptr.To("app-admin")}, (*keycloakapi.Response)(nil), nil).Once()
				m.On("DeleteRealmRole", mock.Anything, testRealmName, "app-admin").Return((*keycloakapi.Response)(nil), nil).Once()
			},
			expectedError: []string{`created realm role "app-admin"`},
		},
		{
			name: "success - composite listed later in spec is created first",
			realmRoles: &[]keycloakApi.RealmRole{
				{Name: "app-admin", Composite: "app-base"},
				{Name: "app-base", Composite: ""},
			},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "app-admin")
				roleNotFound(m, "app-base")
				createsRole(m, "app-base", nil)
				createsRole(m, "app-admin", nil)
				roleFound(m, "app-admin", "app-admin-id")
				addsToComposite(m, "app-base", "app-admin", "app-admin-id", nil)
			},
		},
		{
			name: "error - entries that are each other's composite",
			realmRoles: &[]keycloakApi.RealmRole{
				{Name: "role-a", Composite: "role-b"},
				{Name: "role-b", Composite: "role-a"},
			},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "role-a")
				roleNotFound(m, "role-b")
			},
			expectedError: []string{`realm roles ["role-a" "role-b"] cannot be ordered: composite cycle`},
		},
		{
			name: "error - cycle fails before any role is created",
			realmRoles: &[]keycloakApi.RealmRole{
				{Name: "role-x", Composite: ""},
				{Name: "role-a", Composite: "role-b"},
				{Name: "role-b", Composite: "role-a"},
			},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "role-x")
				roleNotFound(m, "role-a")
				roleNotFound(m, "role-b")
			},
			expectedError: []string{`realm roles ["role-a" "role-b"] cannot be ordered: composite cycle`},
		},
		{
			name:       "error - entry that is its own composite",
			realmRoles: &[]keycloakApi.RealmRole{{Name: "role-a", Composite: "role-a"}},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "role-a")
			},
			expectedError: []string{`realm roles ["role-a"] cannot be ordered: composite cycle`},
		},
		{
			name: "duplicate entries are created once",
			realmRoles: &[]keycloakApi.RealmRole{
				{Name: "role-a", Composite: ""},
				{Name: "role-a", Composite: ""},
			},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "role-a")
				createsRole(m, "role-a", nil)
			},
		},
		{
			name: "error - missing composite fails before any role is created",
			realmRoles: &[]keycloakApi.RealmRole{
				{Name: "role-x", Composite: ""},
				{Name: "app-admin", Composite: "ghost"},
				{Name: "app-user", Composite: "ghost2"},
			},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "role-x")
				roleNotFound(m, "app-admin")
				roleNotFound(m, "app-user")
				roleNotFound(m, "ghost")
				roleNotFound(m, "ghost2")
			},
			expectedError: []string{`composite realm roles "ghost" of "app-admin", "ghost2" of "app-user" not found`},
		},
		{
			name: "success - shared composite is looked up once",
			realmRoles: &[]keycloakApi.RealmRole{
				{Name: "role-a", Composite: "base"},
				{Name: "role-b", Composite: "base"},
			},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "role-a")
				roleNotFound(m, "role-b")
				roleFound(m, "base", "base-id")
				for _, name := range []string{"role-a", "role-b"} {
					createsRole(m, name, nil)
					roleFound(m, name, name+"-id")
					addsToComposite(m, "base", name, name+"-id", nil)
				}
			},
		},
		{
			name:       "error - rollback tolerates a role that is already gone",
			realmRoles: &[]keycloakApi.RealmRole{{Name: "app-admin", Composite: "administrator"}},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "app-admin")
				roleFound(m, "administrator", "admin-id")
				createsRole(m, "app-admin", nil)
				roleFound(m, "app-admin", "app-admin-id")
				addsToComposite(m, "administrator", "app-admin", "app-admin-id", errors.New("add failed"))
				m.On("DeleteRealmRole", mock.Anything, testRealmName, "app-admin").
					Return((*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).Once()
			},
			expectedError:   []string{"error adding realm role app-admin to composite realm role administrator"},
			unexpectedError: "unable to delete realm role",
		},
		{
			name: "success - multiple roles",
			realmRoles: &[]keycloakApi.RealmRole{
				{Name: "role1", Composite: "composite1"},
				{Name: "role2", Composite: "composite2"},
			},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				for _, r := range []struct{ name, parent string }{{"role1", "composite1"}, {"role2", "composite2"}} {
					roleNotFound(m, r.name)
					roleFound(m, r.parent, r.parent+"-id")
					createsRole(m, r.name, nil)
					roleFound(m, r.name, r.name+"-id")
					addsToComposite(m, r.parent, r.name, r.name+"-id", nil)
				}
			},
		},
		{
			name:       "success - no realm roles (nil)",
			realmRoles: nil,
			mockSetup:  func(m *keycloakapiMocks.MockRolesClient) {},
		},
		{
			name:       "success - no realm roles (empty slice)",
			realmRoles: &[]keycloakApi.RealmRole{},
			mockSetup:  func(m *keycloakapiMocks.MockRolesClient) {},
		},
		{
			name:       "existing role is left unchanged",
			realmRoles: &[]keycloakApi.RealmRole{{Name: "existing-role", Composite: "composite"}},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleFound(m, "existing-role", "id")
			},
		},
		{
			name:       "error - GetRealmRole fails",
			realmRoles: &[]keycloakApi.RealmRole{{Name: "test-role", Composite: "composite"}},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				m.On("GetRealmRole", mock.Anything, testRealmName, "test-role").
					Return((*keycloakapi.RoleRepresentation)(nil), (*keycloakapi.Response)(nil), errors.New("keycloak api error")).Once()
			},
			expectedError: []string{"error checking realm role test-role"},
		},
		{
			name:       "error - CreateRealmRole fails",
			realmRoles: &[]keycloakApi.RealmRole{{Name: "test-role", Composite: "composite"}},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "test-role")
				roleFound(m, "composite", "composite-id")
				createsRole(m, "test-role", errors.New("creation failed"))
			},
			expectedError: []string{"error creating realm role test-role"},
		},
		{
			name: "multiple roles - first exists, second is created",
			realmRoles: &[]keycloakApi.RealmRole{
				{Name: "existing-role", Composite: "composite1"},
				{Name: "new-role", Composite: "composite2"},
			},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleFound(m, "existing-role", "id")
				roleNotFound(m, "new-role")
				roleFound(m, "composite2", "composite2-id")
				createsRole(m, "new-role", nil)
				roleFound(m, "new-role", "new-role-id")
				addsToComposite(m, "composite2", "new-role", "new-role-id", nil)
			},
		},
		{
			name:       "empty composite name - role created without composite",
			realmRoles: &[]keycloakApi.RealmRole{{Name: "test-role", Composite: ""}},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				roleNotFound(m, "test-role")
				createsRole(m, "test-role", nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keycloakClient := &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{Name: testClientName, Namespace: testClientNamespace},
				Spec:       keycloakApi.KeycloakClientSpec{RealmRoles: tt.realmRoles},
			}

			rolesMock := keycloakapiMocks.NewMockRolesClient(t)
			tt.mockSetup(rolesMock)

			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(keycloakClient).
				WithStatusSubresource(keycloakClient).
				Build()

			service := NewPutRealmRole(&keycloakapi.KeycloakClient{Roles: rolesMock}, k8sClient)
			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())

			err := service.Serve(ctx, keycloakClient, testRealmName, &ClientContext{})

			if len(tt.expectedError) == 0 {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)

			for _, want := range tt.expectedError {
				assert.Contains(t, err.Error(), want)
			}

			if tt.unexpectedError != "" {
				assert.NotContains(t, err.Error(), tt.unexpectedError)
			}
		})
	}
}

func TestOrderByComposite(t *testing.T) {
	role := func(name, composite string) keycloakApi.RealmRole {
		return keycloakApi.RealmRole{Name: name, Composite: composite}
	}

	tests := []struct {
		name    string
		roles   []keycloakApi.RealmRole
		want    []string
		wantErr string
	}{
		{name: "no entries", roles: nil, want: []string{}},
		{name: "independent entries keep spec order", roles: []keycloakApi.RealmRole{role("a", ""), role("b", "x")}, want: []string{"a", "b"}},
		{name: "composite listed later comes first", roles: []keycloakApi.RealmRole{role("a", "b"), role("b", "")}, want: []string{"b", "a"}},
		{name: "chain", roles: []keycloakApi.RealmRole{role("a", "b"), role("b", "c"), role("c", "")}, want: []string{"c", "b", "a"}},
		{name: "cycle", roles: []keycloakApi.RealmRole{role("a", "b"), role("b", "a")}, wantErr: `realm roles ["a" "b"] cannot be ordered: composite cycle`},
		{name: "self reference", roles: []keycloakApi.RealmRole{role("a", "a")}, wantErr: `realm roles ["a"] cannot be ordered: composite cycle`},
		{name: "dependent of a cycle is unresolved", roles: []keycloakApi.RealmRole{role("a", "b"), role("b", "a"), role("c", "a")}, wantErr: `realm roles ["a" "b" "c"] cannot be ordered: composite cycle`},
		{name: "empty composite is not a dependency on an empty name", roles: []keycloakApi.RealmRole{role("", ""), role("a", "")}, want: []string{"", "a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := orderByComposite(tt.roles)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)

			names := make([]string, 0, len(got))
			for _, r := range got {
				names = append(names, r.Name)
			}

			assert.Equal(t, tt.want, names)
		})
	}
}
