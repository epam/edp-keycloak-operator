package keycloakapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestFindGroupInList(t *testing.T) {
	tests := []struct {
		name   string
		groups []GroupRepresentation
		search string
		wantID string
	}{
		{
			name:   "match returns the group",
			groups: []GroupRepresentation{{Id: ptr.To("id-1"), Name: ptr.To("developers")}},
			search: "developers",
			wantID: "id-1",
		},
		{
			name:   "no match returns nil",
			groups: []GroupRepresentation{{Id: ptr.To("id-1"), Name: ptr.To("developers")}},
			search: "ops",
		},
		{
			name:   "nil name is skipped",
			groups: []GroupRepresentation{{Id: ptr.To("id-1")}},
			search: "developers",
		},
		{
			name:   "name match without an id is not a usable group",
			groups: []GroupRepresentation{{Name: ptr.To("developers")}},
			search: "developers",
		},
		{
			name: "an unusable match does not shadow a usable one",
			groups: []GroupRepresentation{
				{Name: ptr.To("developers")},
				{Id: ptr.To("id-2"), Name: ptr.To("developers")},
			},
			search: "developers",
			wantID: "id-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findGroupInList(tt.groups, tt.search)

			if tt.wantID == "" {
				assert.Nil(t, got)
				return
			}

			assert.NotNil(t, got)
			assert.Equal(t, tt.wantID, ptr.Deref(got.Id, ""))
		})
	}
}
