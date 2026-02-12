package keycloakv2

import (
	"net/http"
	"net/url"
	"strings"
)

type Response struct {
	Body         []byte
	HTTPResponse *http.Response
}

// GetResourceIDFromResponse extracts the resource ID from the Location header
// in an HTTP response. Returns empty string if Location header is not present.
//
// Location header format: http://host/admin/realms/{realm}/{resource}/{id}
// Examples:
//   - "http://localhost:8086/admin/realms/test/groups/abc-123" -> "abc-123"
//   - "http://localhost:8086/admin/realms/test/clients/uuid-456" -> "uuid-456"
func GetResourceIDFromResponse(resp *Response) string {
	if resp == nil || resp.HTTPResponse == nil {
		return ""
	}

	location := resp.HTTPResponse.Header.Get("Location")
	if location == "" {
		return ""
	}

	// Parse URL to handle query strings, fragments, and escaped characters
	u, err := url.Parse(location)
	if err != nil {
		return ""
	}

	// Use EscapedPath to preserve encoded path separators (e.g., %2F)
	// This ensures we split on actual path boundaries, not encoded slashes within IDs
	path := strings.TrimRight(u.EscapedPath(), "/")

	parts := strings.Split(path, "/")

	lastSegment := parts[len(parts)-1]
	if lastSegment == "" {
		return ""
	}

	// Unescape the last segment to get the decoded ID
	decoded, err := url.PathUnescape(lastSegment)
	if err != nil {
		// If unescaping fails, return the encoded version
		return lastSegment
	}

	return decoded
}
