package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

type defaultRoleWrite int

const (
	writeNone defaultRoleWrite = iota
	writeAdd
	writeAddError
	writeDelete
	writeDeleteNotFound
	writeDeleteError
)

func TestMakeDefault_Serve(t *testing.T) {
	const (
		roleID          = "role-id-123"
		defaultRoleName = "default-roles-test-realm"
	)

	rep := []keycloakapi.RoleRepresentation{{Id: ptr.To(roleID), Name: ptr.To(testRoleName)}}
	member := []keycloakapi.RoleRepresentation{{Id: ptr.To(roleID), Name: ptr.To(testRoleName)}}
	other := []keycloakapi.RoleRepresentation{{Id: ptr.To("other"), Name: ptr.To("other")}}
	sameNameClientRole := []keycloakapi.RoleRepresentation{
		{Id: ptr.To("client-role-id"), Name: ptr.To(testRoleName), ClientRole: ptr.To(true)},
	}
	otherAndMember := []keycloakapi.RoleRepresentation{other[0], member[0]}
	nilID := []keycloakapi.RoleRepresentation{{Name: ptr.To(testRoleName)}}
	realmWithDefault := func(role *keycloakapi.RoleRepresentation) func() (*keycloakapi.RealmRepresentation, error) {
		return func() (*keycloakapi.RealmRepresentation, error) {
			return &keycloakapi.RealmRepresentation{DefaultRole: role}, nil
		}
	}

	tests := []struct {
		name          string
		realm         string
		isDefault     bool
		composites    []keycloakapi.RoleRepresentation
		compositesErr error
		write         defaultRoleWrite
		// getRealm replaces the default GetRealm response; the handler then stops after GetRealm.
		getRealm func() (*keycloakapi.RealmRepresentation, error)
		wantErr  []string
		wantIs   error
	}{
		{name: "default, not a member: add", isDefault: true, composites: other, write: writeAdd},
		{name: "default, member: no write", isDefault: true, composites: member},
		{name: "not default, member: remove", composites: otherAndMember, write: writeDelete},
		{name: "not default, not a member: no write", composites: []keycloakapi.RoleRepresentation{}},
		{name: "default, client role with the same name: add", isDefault: true, composites: sameNameClientRole, write: writeAdd},
		{name: "not default, client role with the same name: no write", composites: sameNameClientRole},
		{name: "default, composite without id: add", isDefault: true, composites: nilID, write: writeAdd},
		{name: "not default, composite without id: no write", composites: nilID},
		{
			name:       "upper-case realm: default role name from the realm",
			realm:      "Test-Realm",
			isDefault:  true,
			composites: other,
			write:      writeAdd,
		},
		{
			name:       "add error",
			isDefault:  true,
			composites: other,
			write:      writeAddError,
			wantErr:    []string{"failed to add role to default-roles"},
		},
		{
			name:       "remove error",
			composites: member,
			write:      writeDeleteError,
			wantErr:    []string{"failed to remove role from default-roles"},
		},
		{name: "remove not found is done", composites: member, write: writeDeleteNotFound},
		{
			name:          "get composites error",
			isDefault:     true,
			compositesErr: keycloakapi.ErrNotFound,
			wantErr:       []string{"failed to get default-roles composites"},
		},
		{
			name:      "get realm error",
			isDefault: true,
			getRealm: func() (*keycloakapi.RealmRepresentation, error) {
				return nil, errors.New("api error")
			},
			wantErr: []string{"failed to get realm", "test-realm"},
		},
		{
			name:      "nil realm",
			isDefault: true,
			getRealm: func() (*keycloakapi.RealmRepresentation, error) {
				return nil, nil
			},
			wantErr: []string{`failed to get realm "test-realm"`},
			wantIs:  keycloakapi.ErrNilRealmRepresentation,
		},
		{
			name:      "nil default role",
			isDefault: true,
			getRealm:  realmWithDefault(nil),
			wantErr:   []string{`realm "test-realm" has no default role in the Keycloak response`},
		},
		{
			name:      "nil default role name",
			isDefault: true,
			getRealm:  realmWithDefault(&keycloakapi.RoleRepresentation{}),
			wantErr:   []string{`realm "test-realm" has no default role in the Keycloak response`},
		},
		{
			name:      "empty default role name",
			isDefault: true,
			getRealm:  realmWithDefault(&keycloakapi.RoleRepresentation{Name: ptr.To("")}),
			wantErr:   []string{`realm "test-realm" has no default role in the Keycloak response`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			realmName := tt.realm
			if realmName == "" {
				realmName = "test-realm"
			}

			realms := mocks.NewMockRealmClient(t)
			roles := mocks.NewMockRolesClient(t)

			if tt.getRealm != nil {
				kcRealm, err := tt.getRealm()
				realms.EXPECT().GetRealm(context.Background(), realmName).Return(kcRealm, nil, err)
			} else {
				realms.EXPECT().GetRealm(context.Background(), realmName).
					Return(&keycloakapi.RealmRepresentation{
						DefaultRole: &keycloakapi.RoleRepresentation{Name: ptr.To(defaultRoleName)},
					}, nil, nil)
				roles.EXPECT().GetRealmRoleComposites(context.Background(), realmName, defaultRoleName).
					Return(tt.composites, nil, tt.compositesErr)
			}

			add := roles.EXPECT().AddRealmRoleComposites
			del := roles.EXPECT().DeleteRealmRoleComposites

			switch tt.write {
			case writeAdd:
				add(context.Background(), realmName, defaultRoleName, rep).Return(nil, nil)
			case writeAddError:
				add(context.Background(), realmName, defaultRoleName, rep).Return(nil, errors.New("api error"))
			case writeDelete:
				del(context.Background(), realmName, defaultRoleName, rep).Return(nil, nil)
			case writeDeleteNotFound:
				del(context.Background(), realmName, defaultRoleName, rep).Return(nil, keycloakapi.ErrNotFound)
			case writeDeleteError:
				del(context.Background(), realmName, defaultRoleName, rep).Return(nil, errors.New("api error"))
			case writeNone:
			}

			role := &keycloakApi.KeycloakRealmRole{}
			role.Spec.Name = testRoleName
			role.Spec.IsDefault = tt.isDefault

			h := NewMakeDefault(&keycloakapi.KeycloakClient{Realms: realms, Roles: roles})
			err := h.Serve(context.Background(), role, realmName, &RoleContext{RoleID: roleID})

			if len(tt.wantErr) == 0 {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)

			for _, s := range tt.wantErr {
				assert.Contains(t, err.Error(), s)
			}

			if tt.wantIs != nil {
				assert.ErrorIs(t, err, tt.wantIs)
			}
		})
	}
}
