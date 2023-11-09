package objectmeta

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPreserveResourcesOnDeletion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		object v1.Object
		want   bool
	}{
		{
			name: "should return true if annotation is set",
			object: &v1.ObjectMeta{
				Annotations: map[string]string{
					PreserveResourcesOnDeletionAnnotation: "true",
				},
			},
			want: true,
		},
		{
			name: "should return false if annotation is not set",
			object: &v1.ObjectMeta{
				Annotations: map[string]string{},
			},
			want: false,
		},
		{
			name: "should return false if annotation is set to false",
			object: &v1.ObjectMeta{
				Annotations: map[string]string{
					PreserveResourcesOnDeletionAnnotation: "false",
				},
			},
			want: false,
		},
		{
			name: "should return false if annotation is set to not empty string",
			object: &v1.ObjectMeta{
				Annotations: map[string]string{
					PreserveResourcesOnDeletionAnnotation: "some string",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := PreserveResourcesOnDeletion(tt.object)
			assert.Equal(t, tt.want, got)
		})
	}
}
