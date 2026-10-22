package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapiMocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

const (
	smOwnerUUID = "client-uuid"
	smAPIUUID   = "api-uuid"
)

func smRole(name string) keycloakapi.RoleRepresentation {
	return keycloakapi.RoleRepresentation{Id: ptr.To(name + "-id"), Name: ptr.To(name)}
}

func smRoles(names ...string) []keycloakapi.RoleRepresentation {
	out := make([]keycloakapi.RoleRepresentation, 0, len(names))
	for _, n := range names {
		out = append(out, smRole(n))
	}

	return out
}

func smNotFound() error {
	return &keycloakapi.ApiError{Code: 404, Message: "not found"}
}

type smMocks struct {
	clients *keycloakapiMocks.MockClientsClient
	roles   *keycloakapiMocks.MockRolesClient
}

func TestScopeMappings_Serve(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	nilResp := (*keycloakapi.Response)(nil)
	apiClient := &keycloakapi.ClientRepresentation{Id: ptr.To(smAPIUUID), ClientId: ptr.To("api")}
	boom := errors.New("boom")

	tests := []struct {
		name          string
		spec          *keycloakApi.ScopeMappings
		addOnly       bool
		setup         func(m smMocks)
		expectedError string
		wantReason    string
		wantStatus    metav1.ConditionStatus
	}{
		{
			name: "absent field makes no call and sets no condition",
			spec: nil,
		},
		{
			name: "adds a missing realm role",
			spec: &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{"rr"})},
			setup: func(m smMocks) {
				m.clients.On("GetRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID).
					Return([]keycloakapi.RoleRepresentation{}, nilResp, nil)
				m.roles.On("GetRealmRole", mock.Anything, testRealmName, "rr").
					Return(ptr.To(smRole("rr")), nilResp, nil)
				m.clients.On("AddRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smRoles("rr")).
					Return(nilResp, nil).Once()
			},
			wantReason: ReasonScopeMappingsSynced,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name: "settled realm role makes no write",
			spec: &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{"rr"})},
			setup: func(m smMocks) {
				m.clients.On("GetRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID).
					Return(smRoles("rr"), nilResp, nil)
			},
			wantReason: ReasonScopeMappingsSynced,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name: "removes an extra realm role",
			spec: &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{"rr"})},
			setup: func(m smMocks) {
				m.clients.On("GetRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID).
					Return(smRoles("rr", "x"), nilResp, nil)
				m.clients.On("DeleteRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smRoles("x")).
					Return(nilResp, nil).Once()
			},
			wantReason: ReasonScopeMappingsSynced,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name:    "addOnly keeps an extra realm role",
			spec:    &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{"rr"})},
			addOnly: true,
			setup: func(m smMocks) {
				m.clients.On("GetRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID).
					Return(smRoles("rr", "x"), nilResp, nil)
			},
			wantReason: ReasonScopeMappingsSynced,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name: "empty list clears every realm role with an explicit list",
			spec: &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{})},
			setup: func(m smMocks) {
				m.clients.On("GetRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID).
					Return(smRoles("rr", "x"), nilResp, nil)
				m.clients.On("DeleteRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smRoles("rr", "x")).
					Return(nilResp, nil).Once()
			},
			wantReason: ReasonScopeMappingsSynced,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name: "empty list with nothing mapped makes no delete",
			spec: &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{})},
			setup: func(m smMocks) {
				m.clients.On("GetRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID).
					Return([]keycloakapi.RoleRepresentation{}, nilResp, nil)
			},
			wantReason: ReasonScopeMappingsSynced,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name: "repeated role name is looked up and posted once",
			spec: &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{"rr", "rr"})},
			setup: func(m smMocks) {
				m.clients.On("GetRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID).
					Return([]keycloakapi.RoleRepresentation{}, nilResp, nil)
				m.roles.On("GetRealmRole", mock.Anything, testRealmName, "rr").
					Return(ptr.To(smRole("rr")), nilResp, nil).Once()
				m.clients.On("AddRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smRoles("rr")).
					Return(nilResp, nil).Once()
			},
			wantReason: ReasonScopeMappingsSynced,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name:          "empty realm role name is rejected before any call",
			spec:          &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{""})},
			expectedError: "scopeMappings.realmRoles[0]: role name must not be empty",
			wantReason:    ReasonConfigurationError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "missing realm role names the owning CRD",
			spec: &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{"missing"})},
			setup: func(m smMocks) {
				m.clients.On("GetRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID).
					Return([]keycloakapi.RoleRepresentation{}, nilResp, nil)
				m.roles.On("GetRealmRole", mock.Anything, testRealmName, "missing").
					Return((*keycloakapi.RoleRepresentation)(nil), nilResp, smNotFound())
			},
			expectedError: `realm role "missing" not found in realm "test-realm"; create it with a KeycloakRealmRole`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "adds a missing client role",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "api", Roles: ptr.To([]string{"read"})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "api").
					Return(apiClient, nilResp, nil)
				m.clients.On("GetClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID).
					Return([]keycloakapi.RoleRepresentation{}, nilResp, nil)
				m.clients.On("GetClientRole", mock.Anything, testRealmName, smAPIUUID, "read").
					Return(ptr.To(smRole("read")), nilResp, nil)
				m.clients.On("AddClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID, smRoles("read")).
					Return(nilResp, nil).Once()
			},
			wantReason: ReasonScopeMappingsSynced,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name: "settled client role makes no write",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "api", Roles: ptr.To([]string{"read"})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "api").
					Return(apiClient, nilResp, nil)
				m.clients.On("GetClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID).
					Return(smRoles("read"), nilResp, nil)
			},
			wantReason: ReasonScopeMappingsSynced,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name: "empty roles clears the client",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "api", Roles: ptr.To([]string{})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "api").
					Return(apiClient, nilResp, nil)
				m.clients.On("GetClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID).
					Return(smRoles("read"), nilResp, nil)
				m.clients.On("DeleteClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID, smRoles("read")).
					Return(nilResp, nil).Once()
			},
			wantReason: ReasonScopeMappingsSynced,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name:       "absent roles skips the entry",
			spec:       &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{{ClientID: "api"}}},
			wantReason: ReasonScopeMappingsSynced,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name: "empty clientId is rejected before any call",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "", Roles: ptr.To([]string{"x"})},
			}},
			expectedError: "scopeMappings.clientRoles[0]: clientId must not be empty",
			wantReason:    ReasonConfigurationError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "repeated clientId is rejected before any call",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "api", Roles: ptr.To([]string{"a"})},
				{ClientID: "api", Roles: ptr.To([]string{"b"})},
			}},
			expectedError: `scopeMappings.clientRoles[1]: duplicate clientId "api"`,
			wantReason:    ReasonConfigurationError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "empty client role name is rejected before any call",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "api", Roles: ptr.To([]string{""})},
			}},
			expectedError: "scopeMappings.clientRoles[0].roles[0]: role name must not be empty",
			wantReason:    ReasonConfigurationError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "missing client names the owning CRD",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "nope", Roles: ptr.To([]string{"x"})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "nope").
					Return((*keycloakapi.ClientRepresentation)(nil), nilResp, keycloakapi.ErrNotFound)
			},
			expectedError: `client "nope" not found in realm "test-realm"; create it with a KeycloakClient`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "client without id gives the same text",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "nope", Roles: ptr.To([]string{"x"})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "nope").
					Return(&keycloakapi.ClientRepresentation{}, nilResp, nil)
			},
			expectedError: `client "nope" not found in realm "test-realm"; create it with a KeycloakClient`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "missing client role names the owning CRD",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "api", Roles: ptr.To([]string{"nope"})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "api").
					Return(apiClient, nilResp, nil)
				m.clients.On("GetClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID).
					Return([]keycloakapi.RoleRepresentation{}, nilResp, nil)
				m.clients.On("GetClientRole", mock.Anything, testRealmName, smAPIUUID, "nope").
					Return((*keycloakapi.RoleRepresentation)(nil), nilResp, smNotFound())
			},
			expectedError: `client role "nope" not found for client "api"; define it in clientRolesV2 of the KeycloakClient for that client`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "whitespace clientId is rejected before any call",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "  ", Roles: ptr.To([]string{"x"})},
			}},
			expectedError: "scopeMappings.clientRoles[0]: clientId must not be empty",
			wantReason:    ReasonConfigurationError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "client with a different clientId gives the not-found text",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "nope", Roles: ptr.To([]string{"x"})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "nope").
					Return(&keycloakapi.ClientRepresentation{Id: ptr.To(smAPIUUID), ClientId: ptr.To("other")}, nilResp, nil)
			},
			expectedError: `client "nope" not found in realm "test-realm"; create it with a KeycloakClient`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "client with id and without clientId gives the not-found text",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "nope", Roles: ptr.To([]string{"x"})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "nope").
					Return(&keycloakapi.ClientRepresentation{Id: ptr.To(smAPIUUID)}, nilResp, nil)
			},
			expectedError: `client "nope" not found in realm "test-realm"; create it with a KeycloakClient`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "client with matching clientId and without id gives the not-found text",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "nope", Roles: ptr.To([]string{"x"})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "nope").
					Return(&keycloakapi.ClientRepresentation{ClientId: ptr.To("nope")}, nilResp, nil)
			},
			expectedError: `client "nope" not found in realm "test-realm"; create it with a KeycloakClient`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "nil client without error gives the not-found text",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "nope", Roles: ptr.To([]string{"x"})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "nope").
					Return((*keycloakapi.ClientRepresentation)(nil), nilResp, nil)
			},
			expectedError: `client "nope" not found in realm "test-realm"; create it with a KeycloakClient`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "repeated client role name is looked up and posted once",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "api", Roles: ptr.To([]string{"read", "read"})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "api").
					Return(apiClient, nilResp, nil)
				m.clients.On("GetClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID).
					Return([]keycloakapi.RoleRepresentation{}, nilResp, nil)
				m.clients.On("GetClientRole", mock.Anything, testRealmName, smAPIUUID, "read").
					Return(ptr.To(smRole("read")), nilResp, nil).Once()
				m.clients.On("AddClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID, smRoles("read")).
					Return(nilResp, nil).Once()
			},
			wantReason: ReasonScopeMappingsSynced,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name:    "addOnly keeps an extra client role and adds the missing one",
			spec:    &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{{ClientID: "api", Roles: ptr.To([]string{"read"})}}},
			addOnly: true,
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "api").
					Return(apiClient, nilResp, nil)
				m.clients.On("GetClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID).
					Return(smRoles("x"), nilResp, nil)
				m.clients.On("GetClientRole", mock.Anything, testRealmName, smAPIUUID, "read").
					Return(ptr.To(smRole("read")), nilResp, nil)
				m.clients.On("AddClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID, smRoles("read")).
					Return(nilResp, nil).Once()
			},
			wantReason: ReasonScopeMappingsSynced,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name: "realm GET failure",
			spec: &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{"rr"})},
			setup: func(m smMocks) {
				m.clients.On("GetRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID).
					Return(nil, nilResp, boom)
			},
			expectedError: "unable to get realm scope mappings",
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "realm role lookup failure",
			spec: &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{"rr"})},
			setup: func(m smMocks) {
				m.clients.On("GetRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID).
					Return([]keycloakapi.RoleRepresentation{}, nilResp, nil)
				m.roles.On("GetRealmRole", mock.Anything, testRealmName, "rr").
					Return((*keycloakapi.RoleRepresentation)(nil), nilResp, boom)
			},
			expectedError: `unable to get realm role "rr"`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "realm role without id",
			spec: &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{"rr"})},
			setup: func(m smMocks) {
				m.clients.On("GetRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID).
					Return([]keycloakapi.RoleRepresentation{}, nilResp, nil)
				m.roles.On("GetRealmRole", mock.Anything, testRealmName, "rr").
					Return(&keycloakapi.RoleRepresentation{Name: ptr.To("rr")}, nilResp, nil)
			},
			expectedError: `realm role "rr" has no id`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "realm POST failure",
			spec: &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{"rr"})},
			setup: func(m smMocks) {
				m.clients.On("GetRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID).
					Return([]keycloakapi.RoleRepresentation{}, nilResp, nil)
				m.roles.On("GetRealmRole", mock.Anything, testRealmName, "rr").
					Return(ptr.To(smRole("rr")), nilResp, nil)
				m.clients.On("AddRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smRoles("rr")).
					Return(nilResp, boom)
			},
			expectedError: "unable to add realm scope mappings",
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "realm DELETE failure",
			spec: &keycloakApi.ScopeMappings{RealmRoles: ptr.To([]string{})},
			setup: func(m smMocks) {
				m.clients.On("GetRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID).
					Return(smRoles("x"), nilResp, nil)
				m.clients.On("DeleteRealmScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smRoles("x")).
					Return(nilResp, boom)
			},
			expectedError: "unable to delete realm scope mappings",
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "client lookup failure",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "api", Roles: ptr.To([]string{"read"})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "api").
					Return((*keycloakapi.ClientRepresentation)(nil), nilResp, boom)
			},
			expectedError: `unable to get client "api"`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "client GET failure",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "api", Roles: ptr.To([]string{"read"})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "api").
					Return(apiClient, nilResp, nil)
				m.clients.On("GetClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID).
					Return(nil, nilResp, boom)
			},
			expectedError: `unable to get scope mappings of client "api"`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "client POST failure",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "api", Roles: ptr.To([]string{"read"})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "api").
					Return(apiClient, nilResp, nil)
				m.clients.On("GetClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID).
					Return([]keycloakapi.RoleRepresentation{}, nilResp, nil)
				m.clients.On("GetClientRole", mock.Anything, testRealmName, smAPIUUID, "read").
					Return(ptr.To(smRole("read")), nilResp, nil)
				m.clients.On("AddClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID, smRoles("read")).
					Return(nilResp, boom)
			},
			expectedError: `unable to add scope mappings of client "api"`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
		{
			name: "client DELETE failure",
			spec: &keycloakApi.ScopeMappings{ClientRoles: []keycloakApi.UserClientRole{
				{ClientID: "api", Roles: ptr.To([]string{})},
			}},
			setup: func(m smMocks) {
				m.clients.On("GetClientByClientID", mock.Anything, testRealmName, "api").
					Return(apiClient, nilResp, nil)
				m.clients.On("GetClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID).
					Return(smRoles("read"), nilResp, nil)
				m.clients.On("DeleteClientScopeMappings", mock.Anything, testRealmName, smOwnerUUID, smAPIUUID, smRoles("read")).
					Return(nilResp, boom)
			},
			expectedError: `unable to delete scope mappings of client "api"`,
			wantReason:    ReasonKeycloakAPIError,
			wantStatus:    metav1.ConditionFalse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := smMocks{
				clients: keycloakapiMocks.NewMockClientsClient(t),
				roles:   keycloakapiMocks.NewMockRolesClient(t),
			}
			if tt.setup != nil {
				tt.setup(m)
			}

			kc := &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{Name: testClientName, Namespace: testClientNamespace},
				Spec:       keycloakApi.KeycloakClientSpec{ClientId: testClientID, ScopeMappings: tt.spec},
			}
			if tt.addOnly {
				kc.Spec.ReconciliationStrategy = keycloakApi.ReconciliationStrategyAddOnly
			}

			k8sClient := fake.NewClientBuilder().WithScheme(s).WithObjects(kc).WithStatusSubresource(kc).Build()

			err := NewScopeMappings(
				&keycloakapi.KeycloakClient{Clients: m.clients, Roles: m.roles}, k8sClient,
			).Serve(context.Background(), kc, testRealmName, &ClientContext{ClientUUID: smOwnerUUID})

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}

			cond := meta.FindStatusCondition(kc.Status.Conditions, ConditionScopeMappingsSynced)
			if tt.wantReason == "" {
				assert.Nil(t, cond)
				return
			}

			require.NotNil(t, cond)
			assert.Equal(t, tt.wantReason, cond.Reason)
			assert.Equal(t, tt.wantStatus, cond.Status)
		})
	}
}
