package keycloakv2

import "net/http"

type Response struct {
	Body         []byte
	HTTPResponse *http.Response
}
