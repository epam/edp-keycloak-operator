package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The user controller reads an omitted role list as "not managed" and an empty one as "remove
// every role". The controller also rewrites the whole resource when it adds its finalizer or
// migrates attributes, so an empty list has to survive a marshal/unmarshal cycle: with omitempty
// on the json tag it would come back as omitted and silently downgrade to "not managed".
func TestKeycloakRealmUserSpec_EmptyRoleListsSurviveSerialization(t *testing.T) {
	spec := KeycloakRealmUserSpec{
		Username: "testuser",
		Roles:    []string{},
		ClientRoles: []UserClientRole{
			{ClientID: "client1", Roles: []string{}},
		},
	}

	raw, err := json.Marshal(spec)
	require.NoError(t, err)

	var decoded KeycloakRealmUserSpec

	require.NoError(t, json.Unmarshal(raw, &decoded))

	assert.NotNil(t, decoded.Roles, "empty spec.roles must not decode back as omitted")
	assert.Empty(t, decoded.Roles)

	require.Len(t, decoded.ClientRoles, 1)
	assert.NotNil(t, decoded.ClientRoles[0].Roles, "empty spec.clientRoles[].roles must not decode back as omitted")
	assert.Empty(t, decoded.ClientRoles[0].Roles)
}

// An omitted role list must stay omitted, so that "not managed" is not confused with "remove
// every role" in the other direction either.
func TestKeycloakRealmUserSpec_OmittedRoleListsStayOmitted(t *testing.T) {
	raw, err := json.Marshal(KeycloakRealmUserSpec{
		Username:    "testuser",
		ClientRoles: []UserClientRole{{ClientID: "client1"}},
	})
	require.NoError(t, err)

	var decoded KeycloakRealmUserSpec

	require.NoError(t, json.Unmarshal(raw, &decoded))

	assert.Nil(t, decoded.Roles)
	require.Len(t, decoded.ClientRoles, 1)
	assert.Nil(t, decoded.ClientRoles[0].Roles)
}
