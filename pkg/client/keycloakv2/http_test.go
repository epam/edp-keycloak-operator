package keycloakv2

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetResourceIDFromResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resp     *Response
		expected string
	}{
		{
			name:     "nil response",
			resp:     nil,
			expected: "",
		},
		{
			name: "nil HTTP response",
			resp: &Response{
				HTTPResponse: nil,
			},
			expected: "",
		},
		{
			name: "missing Location header",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{},
				},
			},
			expected: "",
		},
		{
			name: "empty Location header",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{""},
					},
				},
			},
			expected: "",
		},
		{
			name: "valid absolute URL with group ID",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8086/admin/realms/test/groups/abc-123"},
					},
				},
			},
			expected: "abc-123",
		},
		{
			name: "valid absolute URL with client ID",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8086/admin/realms/test/clients/uuid-456"},
					},
				},
			},
			expected: "uuid-456",
		},
		{
			name: "URL with trailing slash",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8086/admin/realms/test/groups/abc-123/"},
					},
				},
			},
			expected: "abc-123",
		},
		{
			name: "relative URL path",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"/admin/realms/test/groups/def-789"},
					},
				},
			},
			expected: "def-789",
		},
		{
			name: "URL with query string",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8086/admin/realms/test/groups/xyz-999?param=value"},
					},
				},
			},
			expected: "xyz-999",
		},
		{
			name: "URL with fragment",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8086/admin/realms/test/groups/frag-111#section"},
					},
				},
			},
			expected: "frag-111",
		},
		{
			name: "URL with both query string and fragment",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8086/admin/realms/test/groups/combo-555?foo=bar#anchor"},
					},
				},
			},
			expected: "combo-555",
		},
		{
			name: "URL with multiple trailing slashes",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8086/admin/realms/test/groups/multi-222///"},
					},
				},
			},
			expected: "multi-222",
		},
		{
			name: "URL with UUID",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8086/admin/realms/test/users/550e8400-e29b-41d4-a716-446655440000"},
					},
				},
			},
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "URL with special characters in ID",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8086/admin/realms/test/groups/special_id-with.chars"},
					},
				},
			},
			expected: "special_id-with.chars",
		},
		{
			name: "just a slash",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"/"},
					},
				},
			},
			expected: "",
		},
		{
			name: "URL with URL-encoded characters",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8086/admin/realms/test/groups/id%20with%20spaces"},
					},
				},
			},
			expected: "id with spaces",
		},
		{
			name: "URL with encoded slash in ID",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8086/admin/realms/test/groups/abc%2F123"},
					},
				},
			},
			expected: "abc/123",
		},
		{
			name: "URL with multiple encoded slashes in ID",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8086/admin/realms/test/groups/path%2Fto%2Fresource"},
					},
				},
			},
			expected: "path/to/resource",
		},
		{
			name: "invalid URL",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"://invalid-url"},
					},
				},
			},
			expected: "",
		},
		{
			name: "URL with port number",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8080/admin/realms/test/groups/port-123"},
					},
				},
			},
			expected: "port-123",
		},
		{
			name: "just domain without path",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"http://localhost:8086"},
					},
				},
			},
			expected: "",
		},
		{
			name: "HTTPS URL",
			resp: &Response{
				HTTPResponse: &http.Response{
					Header: http.Header{
						"Location": []string{"https://keycloak.example.com/admin/realms/prod/clients/secure-123"},
					},
				},
			},
			expected: "secure-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := GetResourceIDFromResponse(tt.resp)
			assert.Equal(t, tt.expected, result)
		})
	}
}
