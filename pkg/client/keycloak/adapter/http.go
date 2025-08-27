package adapter

import "net/http"

const (
	contentTypeHeader = "Content-Type"
	contentTypeJson   = "application/json"
)

// setJSONContentType sets the Content-Type header to application/json
func setJSONContentType(w http.ResponseWriter) {
	w.Header().Set(contentTypeHeader, contentTypeJson)
}
