package fakehttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

type Server interface {
	Start()
	Close()
	GetURL() string
}

type ServerBuilder struct {
	fakeServer *DefaultServer
}

func NewServerBuilder() *ServerBuilder {
	return &ServerBuilder{fakeServer: NewDefaultServer()}
}

// AddStringResponder registers new handler at a given endpoint
// that returns a given response string with a http.StatusOK header.
func (b *ServerBuilder) AddStringResponder(endpoint, response string) *ServerBuilder {
	b.fakeServer.addStringResponder(http.StatusOK, endpoint, response)

	return b
}

// AddStringResponderWithCode registers new handler at a given endpoint
// that returns a given response string and status code header.
func (b *ServerBuilder) AddStringResponderWithCode(code int, endpoint, response string) *ServerBuilder {
	b.fakeServer.addStringResponder(code, endpoint, response)

	return b
}

// AddJsonResponderWithCode registers new handler at a given endpoint
// that returns a given response object as json and status code header.
func (b *ServerBuilder) AddJsonResponderWithCode(code int, endpoint string, response any) *ServerBuilder {
	b.fakeServer.addJsonResponder(code, endpoint, response)

	return b
}

// AddKeycloakAuthResponders registers handlers for standard Keycloak authentication endpoints
// for the master realm. This is useful for testing Keycloak client initialization.
// It adds responses for:
// - /realms/master/protocol/openid-connect/token (returns access token)
// - /admin/realms/master (returns empty JSON)
func (b *ServerBuilder) AddKeycloakAuthResponders() *ServerBuilder {
	return b.AddKeycloakAuthRespondersForRealm("master")
}

// AddKeycloakAuthRespondersForRealm registers handlers for Keycloak authentication endpoints
// for a custom realm. This is useful for testing Keycloak client initialization.
// It adds responses for:
// - /realms/{realm}/protocol/openid-connect/token (returns access token)
// - /admin/realms/{realm} (returns empty JSON)
func (b *ServerBuilder) AddKeycloakAuthRespondersForRealm(realm string) *ServerBuilder {
	tokenResponse := map[string]string{
		"access_token":  "test-access-token",
		"refresh_token": "test-refresh-token",
		"token_type":    "Bearer",
	}

	b.fakeServer.addJsonResponder(http.StatusOK, "/realms/"+realm+"/protocol/openid-connect/token", tokenResponse)
	b.fakeServer.addJsonResponder(http.StatusOK, "/admin/realms/"+realm, map[string]interface{}{})

	return b
}

// BuildAndStart returns a running Server. Don't forget to close it when done using Server.Close.
func (b *ServerBuilder) BuildAndStart() Server {
	b.fakeServer.Start()

	return b.fakeServer
}

type DefaultServer struct {
	mux        *http.ServeMux
	testServer *httptest.Server
}

func NewDefaultServer() *DefaultServer {
	return &DefaultServer{mux: http.NewServeMux()}
}

func (s *DefaultServer) addStringResponder(status int, endpoint, response string) {
	s.mux.HandleFunc(endpoint, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(status)

		if _, err := writer.Write([]byte(response)); err != nil {
			panic(err)
		}
	})
}

func (s *DefaultServer) addJsonResponder(status int, endpoint string, response any) {
	s.mux.HandleFunc(endpoint, func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)

		jsonResp, err := json.Marshal(response)
		if err != nil {
			panic(err)
		}

		if _, err := writer.Write(jsonResp); err != nil {
			panic(err)
		}
	})
}

func (s *DefaultServer) Start() {
	s.testServer = httptest.NewServer(s.mux)
}

func (s *DefaultServer) Close() {
	if s.testServer == nil {
		panic("attempted to close a server that was never initialized; try to start the server before closing it")
	}

	s.testServer.Close()
}

func (s *DefaultServer) GetURL() string {
	if s.testServer == nil {
		panic("attempted to get URL of a server that was never initialized; try to start the server before getting it's URL")
	}

	return s.testServer.URL
}
