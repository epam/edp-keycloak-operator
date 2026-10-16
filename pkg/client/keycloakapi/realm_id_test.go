package keycloakapi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

func TestRealmID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		realm   *keycloakapi.RealmRepresentation
		want    string
		wantErr error
	}{
		{name: "nil representation", realm: nil, wantErr: keycloakapi.ErrNilRealmRepresentation},
		{name: "nil id", realm: &keycloakapi.RealmRepresentation{}, wantErr: keycloakapi.ErrRealmIDMissing},
		{name: "empty id", realm: &keycloakapi.RealmRepresentation{Id: ptr.To("")}, wantErr: keycloakapi.ErrRealmIDMissing},
		{name: "id set", realm: &keycloakapi.RealmRepresentation{Id: ptr.To("realm-id")}, want: "realm-id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := keycloakapi.RealmID(tt.realm)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
