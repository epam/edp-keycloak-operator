package chain

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/destination"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
	secretrefmocks "github.com/epam/edp-keycloak-operator/pkg/secretref/mocks"
)

const testExecID = "exec-id-456"

const (
	testNamespace      = "test-ns"
	keyDefaultProvider = "defaultProvider"
	keyAPIToken        = "apiToken"
)

// noRefs is the resolver for specs without secret references: nothing to resolve, no tokens.
type noRefs struct{}

func (noRefs) MapConfigSecretsRefs(_ context.Context, _ string, _ map[string]string, _ string) (map[string]string, error) {
	return map[string]string{}, nil
}

// resolving returns a resolver mock that rewrites every "$..." value in config to resolved[value]
// and returns tokenOf(value) as the version token of each resolved key. The expectation is
// optional (.Maybe()).
func resolving(t *testing.T, resolved map[string]string) *secretrefmocks.MockRefClient {
	t.Helper()

	m := secretrefmocks.NewMockRefClient(t)
	m.EXPECT().MapConfigSecretsRefs(mock.Anything, configSecretsField, mock.Anything, testNamespace).
		RunAndReturn(func(_ context.Context, _ string, config map[string]string, _ string) (map[string]string, error) {
			tokens := map[string]string{}

			for k, v := range config {
				if secretref.HasSecretRef(v) {
					config[k] = resolved[v]
					tokens[k] = tokenOf(v)
				}
			}

			return tokens, nil
		}).
		Maybe()

	return m
}

func tokenOf(ref string) string {
	return "version-of-" + ref
}

func locationResponse(id string) *keycloakapi.Response {
	return &keycloakapi.Response{
		HTTPResponse: &http.Response{
			Header: http.Header{
				"Location": []string{"http://localhost/admin/realms/test-realm/authentication/executions/" + id},
			},
		},
	}
}

// setIdpRedirectorSpec fills the spec with one configured identity-provider-redirector execution.
func setIdpRedirectorSpec(flow *keycloakApi.KeycloakAuthFlow) {
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{
			Authenticator: "identity-provider-redirector",
			Requirement:   "ALTERNATIVE",
			Priority:      10,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{
				Alias:  "idp-config",
				Config: map[string]string{keyDefaultProvider: "github"},
			},
		},
	}
}

// nonFlowExec builds a top-level, non-flow execution as returned by GetFlowExecutions.
func nonFlowExec(id, providerID, requirement string, priority int32, configID *string) keycloakapi.AuthenticationExecutionInfoRepresentation {
	return keycloakapi.AuthenticationExecutionInfoRepresentation{
		Id:                   ptr.To(id),
		ProviderId:           ptr.To(providerID),
		AuthenticationFlow:   ptr.To(false),
		Level:                ptr.To(int32(0)),
		Requirement:          ptr.To(requirement),
		Priority:             ptr.To(priority),
		AuthenticationConfig: configID,
	}
}

func TestSyncAuthFlowExecutions_Serve_NoExistingNoSpec(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_SkipFlowTypeExec(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Alias: "sub-flow", AuthenticatorFlow: true},
	}

	// Flow-type exec is filtered from the non-flow sync; spec entry already matches → no update
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			{Id: ptr.To("flow-exec"), DisplayName: ptr.To("sub-flow"), AuthenticationFlow: ptr.To(true), Level: ptr.To(int32(0)), Priority: ptr.To(int32(0))},
		}, nil, nil).Once()

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_InSync(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth", Requirement: "REQUIRED", Priority: 10},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			nonFlowExec("exec-1", "basic-auth", "REQUIRED", 10, nil),
		}, nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_RequirementDrift(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth", Requirement: "REQUIRED", Priority: 10},
	}

	existing := nonFlowExec("exec-1", "basic-auth", "DISABLED", 10, nil)

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	updated := existing
	updated.Requirement = ptr.To("REQUIRED")
	updated.Priority = ptr.To(int32(10))

	mockFlows.EXPECT().UpdateFlowExecution(context.Background(), testRealmName, testFlowAlias, updated).
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_PriorityDrift(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth", Requirement: "REQUIRED", Priority: 30},
	}

	existing := nonFlowExec("exec-1", "basic-auth", "REQUIRED", 10, nil)

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	updated := existing
	updated.Priority = ptr.To(int32(30))

	mockFlows.EXPECT().UpdateFlowExecution(context.Background(), testRealmName, testFlowAlias, updated).
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_ConfigDrift(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-1"

	flow := &keycloakApi.KeycloakAuthFlow{}
	setIdpRedirectorSpec(flow)

	existing := nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	existingConfig := map[string]string{keyDefaultProvider: "gitlab"}
	mockFlows.EXPECT().GetAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(&keycloakapi.AuthenticatorConfigRepresentation{
			Id:     ptr.To(configID),
			Alias:  ptr.To("idp-config"),
			Config: &existingConfig,
		}, nil, nil)

	desiredConfig := map[string]string{keyDefaultProvider: "github"}
	mockFlows.EXPECT().UpdateAuthenticatorConfig(context.Background(), testRealmName, configID,
		keycloakapi.AuthenticatorConfigRepresentation{
			Id:     ptr.To(configID),
			Alias:  ptr.To("idp-config"),
			Config: &desiredConfig,
		},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

// Keycloak returns the mask sentinel for every key the provider does not declare as a
// non-secret property; such keys must not count as drift.
func TestSyncAuthFlowExecutions_Serve_MaskedConfigKeyInSync(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-1"

	flow := &keycloakApi.KeycloakAuthFlow{}
	setIdpRedirectorSpec(flow)
	flow.Spec.AuthenticationExecutions[0].AuthenticatorConfig.Config[keyAPIToken] = "plain-token"

	existing := nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	existingConfig := map[string]string{
		keyDefaultProvider: "github",
		keyAPIToken:        keycloakapi.MaskedSecretValue,
	}
	mockFlows.EXPECT().GetAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(&keycloakapi.AuthenticatorConfigRepresentation{
			Id:     ptr.To(configID),
			Alias:  ptr.To("idp-config"),
			Config: &existingConfig,
		}, nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
	assert.Empty(t, flow.Status.ConfigSecretsHash)
}

func TestSyncAuthFlowExecutions_Serve_MaskedConfigKeyAliasDrift(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-1"

	flow := &keycloakApi.KeycloakAuthFlow{}
	setIdpRedirectorSpec(flow)
	flow.Spec.AuthenticationExecutions[0].AuthenticatorConfig.Config[keyAPIToken] = "plain-token"

	existing := nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	existingConfig := map[string]string{
		keyDefaultProvider: "github",
		keyAPIToken:        keycloakapi.MaskedSecretValue,
	}
	mockFlows.EXPECT().GetAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(&keycloakapi.AuthenticatorConfigRepresentation{
			Id:     ptr.To(configID),
			Alias:  ptr.To("old-alias"),
			Config: &existingConfig,
		}, nil, nil)

	desiredConfig := map[string]string{keyDefaultProvider: "github", keyAPIToken: "plain-token"}
	mockFlows.EXPECT().UpdateAuthenticatorConfig(context.Background(), testRealmName, configID,
		keycloakapi.AuthenticatorConfigRepresentation{
			Id:     ptr.To(configID),
			Alias:  ptr.To("idp-config"),
			Config: &desiredConfig,
		},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

// Only keys whose fetched value is the masked sentinel are skipped. Every clear key is compared
// against the resolved spec value.
func TestSyncAuthFlowExecutions_Serve_ConfigCompare(t *testing.T) {
	const (
		configID = "cfg-1"
		specRef  = "$s:k"
	)

	// Stored hash of the secret versions: equal to the freshly computed one, so no update is forced.
	refHash := secretref.ValuesHashSingle(map[string]string{"0/" + keyAPIToken: tokenOf(specRef)})

	tests := []struct {
		name       string
		specConfig map[string]string
		fetched    *map[string]string
		storedHash string
		wantUpdate bool
		// wantSent is the PUT body config; defaults to specConfig.
		wantSent map[string]string
	}{
		{
			name:       "clear key drifted while masked sibling present",
			specConfig: map[string]string{"defaultProvider": "github", "apiToken": "plain-token"},
			fetched: &map[string]string{
				"defaultProvider": "gitlab",
				"apiToken":        keycloakapi.MaskedSecretValue,
			},
			wantUpdate: true,
		},
		{
			name:       "spec key absent from fetched config while masked sibling present",
			specConfig: map[string]string{"defaultProvider": "github", "apiToken": "plain-token"},
			fetched:    &map[string]string{"apiToken": keycloakapi.MaskedSecretValue},
			wantUpdate: true,
		},
		{
			name:       "fetched config has nil Config map",
			specConfig: map[string]string{"defaultProvider": "github"},
			fetched:    nil,
			wantUpdate: true,
		},
		{
			name:       "secret ref resolved value equals fetched clear value",
			specConfig: map[string]string{keyAPIToken: specRef},
			fetched:    &map[string]string{keyAPIToken: testResolved},
			storedHash: refHash,
			wantUpdate: false,
		},
		{
			name:       "secret ref resolved value differs from fetched clear value",
			specConfig: map[string]string{keyAPIToken: specRef},
			fetched:    &map[string]string{keyAPIToken: "other"},
			storedHash: refHash,
			wantUpdate: true,
			wantSent:   map[string]string{keyAPIToken: testResolved},
		},
		{
			name:       "secret ref key masked in fetched config",
			specConfig: map[string]string{keyAPIToken: specRef},
			fetched:    &map[string]string{keyAPIToken: keycloakapi.MaskedSecretValue},
			storedHash: refHash,
			wantUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFlows := mocks.NewMockAuthFlowsClient(t)
			kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

			flow := &keycloakApi.KeycloakAuthFlow{}
			flow.Namespace = testNamespace
			flow.Status.ConfigSecretsHash = tt.storedHash
			setIdpRedirectorSpec(flow)
			flow.Spec.AuthenticationExecutions[0].AuthenticatorConfig.Config = tt.specConfig

			existing := nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID))

			mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
				Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

			mockFlows.EXPECT().GetAuthenticatorConfig(context.Background(), testRealmName, configID).
				Return(&keycloakapi.AuthenticatorConfigRepresentation{
					Id:     ptr.To(configID),
					Alias:  ptr.To("idp-config"),
					Config: tt.fetched,
				}, nil, nil).Once()

			if tt.wantUpdate {
				wantSent := tt.wantSent
				if wantSent == nil {
					wantSent = tt.specConfig
				}

				mockFlows.EXPECT().UpdateAuthenticatorConfig(context.Background(), testRealmName, configID,
					keycloakapi.AuthenticatorConfigRepresentation{
						Id:     ptr.To(configID),
						Alias:  ptr.To("idp-config"),
						Config: &wantSent,
					},
				).Return(nil, nil).Once()
			}

			h := NewSyncAuthFlowExecutions(kc, resolving(t, map[string]string{specRef: testResolved}))
			require.NoError(t, h.Serve(context.Background(), flow, testRealmName))
		})
	}
}

func TestSyncAuthFlowExecutions_Serve_ConfigRemovedFromSpec(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-2"

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth", Requirement: "REQUIRED", Priority: 10},
	}

	existing := nonFlowExec("exec-1", "basic-auth", "REQUIRED", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	mockFlows.EXPECT().DeleteAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_ExecutionRemovedFromSpec(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-3"

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	existing := nonFlowExec("exec-removed", "old-auth", "REQUIRED", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	mockFlows.EXPECT().DeleteAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(nil, nil)

	mockFlows.EXPECT().DeleteExecution(context.Background(), testRealmName, "exec-removed").
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_ExecutionRemovedFromSpec_ConfigDeleteNotFound(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-4"

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	existing := nonFlowExec("exec-removed", "old-auth", "REQUIRED", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	mockFlows.EXPECT().DeleteAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(nil, keycloakapi.ErrNotFound)

	mockFlows.EXPECT().DeleteExecution(context.Background(), testRealmName, "exec-removed").
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_AddExecution(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Status.ID = testFlowID
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth", AuthenticatorFlow: false},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	mockFlows.EXPECT().AddExecutionToFlow(
		context.Background(), testRealmName,
		keycloakapi.AuthenticationExecutionRepresentation{
			Authenticator: ptr.To("basic-auth"),
			ParentFlow:    ptr.To(testFlowID),
			Requirement:   ptr.To(""),
			Priority:      ptr.To(int32(0)),
		},
	).Return(locationResponse(testExecID), nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_AddExecutionWithConfig(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Status.ID = testFlowID
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{
			Authenticator:     "identity-provider-redirector",
			AuthenticatorFlow: false,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{
				Alias:  "idp-config",
				Config: map[string]string{keyDefaultProvider: "github"},
			},
		},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	mockFlows.EXPECT().AddExecutionToFlow(
		context.Background(), testRealmName,
		keycloakapi.AuthenticationExecutionRepresentation{
			Authenticator: ptr.To("identity-provider-redirector"),
			ParentFlow:    ptr.To(testFlowID),
			Requirement:   ptr.To(""),
			Priority:      ptr.To(int32(0)),
		},
	).Return(locationResponse(testExecID), nil)

	cfg := map[string]string{keyDefaultProvider: "github"}
	mockFlows.EXPECT().CreateExecutionConfig(
		context.Background(), testRealmName, testExecID,
		keycloakapi.AuthenticatorConfigRepresentation{
			Alias:  ptr.To("idp-config"),
			Config: &cfg,
		},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_AddExecutionPropagatesPriority(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Status.ID = testFlowID
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth", AuthenticatorFlow: false, Priority: 5, Requirement: "REQUIRED"},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	mockFlows.EXPECT().AddExecutionToFlow(
		context.Background(), testRealmName,
		keycloakapi.AuthenticationExecutionRepresentation{
			Authenticator: ptr.To("basic-auth"),
			ParentFlow:    ptr.To(testFlowID),
			Requirement:   ptr.To("REQUIRED"),
			Priority:      ptr.To(int32(5)),
		},
	).Return(locationResponse(testExecID), nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_ForceUpdateOnGenerationChange(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-5"

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Generation = 2
	flow.Status.ObservedGeneration = 1
	setIdpRedirectorSpec(flow)

	existing := nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	// Requirement and priority match spec: exact scalar diffs need no forced write.
	desiredConfig := map[string]string{keyDefaultProvider: "github"}

	// Forced by generation drift: config updated without a prior read.
	mockFlows.EXPECT().UpdateAuthenticatorConfig(context.Background(), testRealmName, configID,
		keycloakapi.AuthenticatorConfigRepresentation{
			Id:     ptr.To(configID),
			Alias:  ptr.To("idp-config"),
			Config: &desiredConfig,
		},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

// Forced writes bypass the masked-key skip: the PUT body carries the spec value of every key.
func TestSyncAuthFlowExecutions_Serve_ForceUpdateSendsMaskedKeyValue(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-5"

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Generation = 2
	flow.Status.ObservedGeneration = 1
	setIdpRedirectorSpec(flow)
	flow.Spec.AuthenticationExecutions[0].AuthenticatorConfig.Config["apiToken"] = "plain-token"

	existing := nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	var sent keycloakapi.AuthenticatorConfigRepresentation

	mockFlows.EXPECT().UpdateAuthenticatorConfig(context.Background(), testRealmName, configID, mock.Anything).
		Run(func(_ context.Context, _, _ string, rep keycloakapi.AuthenticatorConfigRepresentation) {
			sent = rep
		}).
		Return(nil, nil).Once()

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	require.NoError(t, h.Serve(context.Background(), flow, testRealmName))

	require.NotNil(t, sent.Config)
	assert.Equal(t, map[string]string{"defaultProvider": "github", "apiToken": "plain-token"}, *sent.Config)
	assert.Equal(t, ptr.To(configID), sent.Id)
	assert.Equal(t, ptr.To("idp-config"), sent.Alias)
}

func TestSyncAuthFlowExecutions_Serve_DanglingConfigRecreated(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-gone"

	flow := &keycloakApi.KeycloakAuthFlow{}
	setIdpRedirectorSpec(flow)

	existing := nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	// Execution references a config that no longer exists: read 404s, update 404s, config is recreated.
	mockFlows.EXPECT().GetAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(nil, nil, keycloakapi.ErrNotFound)

	mockFlows.EXPECT().UpdateAuthenticatorConfig(context.Background(), testRealmName, configID, mock.Anything).
		Return(nil, keycloakapi.ErrNotFound)

	desiredConfig := map[string]string{keyDefaultProvider: "github"}

	mockFlows.EXPECT().CreateExecutionConfig(context.Background(), testRealmName, "exec-1",
		keycloakapi.AuthenticatorConfigRepresentation{
			Alias:  ptr.To("idp-config"),
			Config: &desiredConfig,
		},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_DuplicateAuthenticatorInSync(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "otp-form", Requirement: "REQUIRED", Priority: 10},
		{Authenticator: "otp-form", Requirement: "ALTERNATIVE", Priority: 20},
	}

	existing1 := nonFlowExec("exec-1", "otp-form", "REQUIRED", 10, nil)
	existing2 := nonFlowExec("exec-2", "otp-form", "ALTERNATIVE", 20, nil)

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing1, existing2}, nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_AdjustChildFlowPriority(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Alias: "sub-flow", AuthenticatorFlow: true, Priority: 10, Requirement: "REQUIRED"},
	}

	// Single fetch serves both the non-flow sync and the child-flow priority adjustment.
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			{
				DisplayName:        ptr.To("sub-flow"),
				AuthenticationFlow: ptr.To(true),
				Level:              ptr.To(int32(0)),
				Priority:           ptr.To(int32(20)), // differs from spec (10)
				Requirement:        ptr.To("DISABLED"),
			},
		}, nil, nil).Once()

	updatedExec := keycloakapi.AuthenticationExecutionInfoRepresentation{
		DisplayName:        ptr.To("sub-flow"),
		AuthenticationFlow: ptr.To(true),
		Level:              ptr.To(int32(0)),
		Priority:           ptr.To(int32(10)),
		Requirement:        ptr.To("REQUIRED"),
	}

	mockFlows.EXPECT().UpdateFlowExecution(context.Background(), testRealmName, testFlowAlias, updatedExec).
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_GetFlowExecutionsError(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return(nil, nil, errors.New("api error"))

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get flow executions")
}

func TestSyncAuthFlowExecutions_Serve_DeleteExecutionError(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			nonFlowExec("bad-exec", "basic-auth", "REQUIRED", 0, nil),
		}, nil, nil)

	mockFlows.EXPECT().DeleteExecution(context.Background(), testRealmName, "bad-exec").
		Return(nil, errors.New("delete failed"))

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete execution")
}

func TestSyncAuthFlowExecutions_Serve_AddExecutionError(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Status.ID = testFlowID
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth"},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	mockFlows.EXPECT().AddExecutionToFlow(
		context.Background(), testRealmName,
		keycloakapi.AuthenticationExecutionRepresentation{
			Authenticator: ptr.To("basic-auth"),
			ParentFlow:    ptr.To(testFlowID),
			Requirement:   ptr.To(""),
			Priority:      ptr.To(int32(0)),
		},
	).Return(nil, errors.New("add failed"))

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add execution")
}

func TestSyncAuthFlowExecutions_Serve_EmptyFlowID(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	// Status.ID intentionally left empty
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth"},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	h := NewSyncAuthFlowExecutions(kc, noRefs{})
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flow ID is empty")
}

const (
	testSecretRef = "$sms-secret:token"
	testResolved  = "resolved-token"
)

// refFlow returns a flow in the test namespace with one identity-provider-redirector execution
// whose config carries a secret reference in key refKey next to a plain defaultProvider.
func refFlow(refKey string) *keycloakApi.KeycloakAuthFlow {
	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Namespace = testNamespace
	flow.Status.ID = testFlowID
	setIdpRedirectorSpec(flow)
	flow.Spec.AuthenticationExecutions[0].AuthenticatorConfig.Config[refKey] = testSecretRef

	return flow
}

func TestSyncAuthFlowExecutions_SecretRef_ResolvedOnCreateConfig(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := refFlow(keyAPIToken)

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, nil),
		}, nil, nil)

	resolvedConfig := map[string]string{keyDefaultProvider: "github", keyAPIToken: testResolved}
	mockFlows.EXPECT().CreateExecutionConfig(context.Background(), testRealmName, "exec-1",
		keycloakapi.AuthenticatorConfigRepresentation{Alias: ptr.To("idp-config"), Config: &resolvedConfig},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, resolving(t, map[string]string{testSecretRef: testResolved}))
	require.NoError(t, h.Serve(context.Background(), flow, testRealmName))

	assert.Equal(t, secretref.ValuesHashSingle(map[string]string{"0/apiToken": tokenOf(testSecretRef)}), flow.Status.ConfigSecretsHash)
	assert.Equal(t, testSecretRef, flow.Spec.AuthenticationExecutions[0].AuthenticatorConfig.Config[keyAPIToken],
		"spec must not be mutated")
}

func TestSyncAuthFlowExecutions_SecretRef_ResolvedOnAddExecution(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := refFlow(keyAPIToken)

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)
	mockFlows.EXPECT().AddExecutionToFlow(context.Background(), testRealmName, mock.Anything).
		Return(locationResponse(testExecID), nil)

	resolvedConfig := map[string]string{keyDefaultProvider: "github", keyAPIToken: testResolved}
	mockFlows.EXPECT().CreateExecutionConfig(context.Background(), testRealmName, testExecID,
		keycloakapi.AuthenticatorConfigRepresentation{Alias: ptr.To("idp-config"), Config: &resolvedConfig},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, resolving(t, map[string]string{testSecretRef: testResolved}))
	require.NoError(t, h.Serve(context.Background(), flow, testRealmName))
}

func TestSyncAuthFlowExecutions_SecretRef_DanglingConfigRecreated(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-gone"

	// Empty stored hash with a reference present forces the update.
	flow := refFlow(keyAPIToken)

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID)),
		}, nil, nil)

	resolvedConfig := map[string]string{keyDefaultProvider: "github", keyAPIToken: testResolved}

	mockFlows.EXPECT().UpdateAuthenticatorConfig(context.Background(), testRealmName, configID, mock.Anything).
		Return(nil, keycloakapi.ErrNotFound)
	mockFlows.EXPECT().CreateExecutionConfig(context.Background(), testRealmName, "exec-1",
		keycloakapi.AuthenticatorConfigRepresentation{Alias: ptr.To("idp-config"), Config: &resolvedConfig},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, resolving(t, map[string]string{testSecretRef: testResolved}))
	require.NoError(t, h.Serve(context.Background(), flow, testRealmName))
}

// A reference in a key Keycloak returns in clear is compared by its resolved value, so drift
// there is repaired without a generation or secret change.
func TestSyncAuthFlowExecutions_SecretRef_ClearKeyDriftRepaired(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-1"

	flow := refFlow(keyDefaultProvider)
	flow.Status.ConfigSecretsHash = secretref.ValuesHashSingle(map[string]string{"0/defaultProvider": tokenOf(testSecretRef)})

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID)),
		}, nil, nil)

	existingConfig := map[string]string{keyDefaultProvider: "edited-by-hand"}
	mockFlows.EXPECT().GetAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(&keycloakapi.AuthenticatorConfigRepresentation{
			Id: ptr.To(configID), Alias: ptr.To("idp-config"), Config: &existingConfig,
		}, nil, nil)

	resolvedConfig := map[string]string{keyDefaultProvider: testResolved}
	mockFlows.EXPECT().UpdateAuthenticatorConfig(context.Background(), testRealmName, configID,
		keycloakapi.AuthenticatorConfigRepresentation{Id: ptr.To(configID), Alias: ptr.To("idp-config"), Config: &resolvedConfig},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, resolving(t, map[string]string{testSecretRef: testResolved}))
	require.NoError(t, h.Serve(context.Background(), flow, testRealmName))
}

func TestSyncAuthFlowExecutions_SecretRef_MaskedKeyInSync(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-1"

	flow := refFlow(keyAPIToken)
	flow.Status.ConfigSecretsHash = secretref.ValuesHashSingle(map[string]string{"0/apiToken": tokenOf(testSecretRef)})

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID)),
		}, nil, nil)

	existingConfig := map[string]string{keyDefaultProvider: "github", keyAPIToken: keycloakapi.MaskedSecretValue}
	mockFlows.EXPECT().GetAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(&keycloakapi.AuthenticatorConfigRepresentation{
			Id: ptr.To(configID), Alias: ptr.To("idp-config"), Config: &existingConfig,
		}, nil, nil)

	h := NewSyncAuthFlowExecutions(kc, resolving(t, map[string]string{testSecretRef: testResolved}))
	require.NoError(t, h.Serve(context.Background(), flow, testRealmName))
}

func TestSyncAuthFlowExecutions_SecretRef_HashDriftForcesUpdate(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-1"

	flow := refFlow(keyAPIToken)
	flow.Status.ConfigSecretsHash = secretref.ValuesHashSingle(map[string]string{"0/apiToken": "version-before-rotation"})

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID)),
		}, nil, nil)

	// Forced: no GET, straight to the update with the new value.
	resolvedConfig := map[string]string{keyDefaultProvider: "github", keyAPIToken: testResolved}
	mockFlows.EXPECT().UpdateAuthenticatorConfig(context.Background(), testRealmName, configID,
		keycloakapi.AuthenticatorConfigRepresentation{Id: ptr.To(configID), Alias: ptr.To("idp-config"), Config: &resolvedConfig},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, resolving(t, map[string]string{testSecretRef: testResolved}))
	require.NoError(t, h.Serve(context.Background(), flow, testRealmName))

	assert.Equal(t, secretref.ValuesHashSingle(map[string]string{"0/apiToken": tokenOf(testSecretRef)}), flow.Status.ConfigSecretsHash)
}

// A resolve failure in any execution happens before the first Keycloak read or write, and the
// error names the spec index, counting flow-type entries.
func TestSyncAuthFlowExecutions_SecretRef_ResolveErrorBeforeAnyWrite(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Namespace = testNamespace
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Alias: "child", AuthenticatorFlow: true, Priority: 5},
		{Authenticator: "basic-auth", Requirement: "REQUIRED", Priority: 10,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{Alias: "a", Config: map[string]string{"k": "plain"}}},
		{Authenticator: "sms-auth", Requirement: "REQUIRED", Priority: 20,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{Alias: "b", Config: map[string]string{keyAPIToken: "$missing:key"}}},
	}

	refs := secretrefmocks.NewMockRefClient(t)
	refs.EXPECT().MapConfigSecretsRefs(mock.Anything, configSecretsField, map[string]string{"k": "plain"}, testNamespace).
		Return(map[string]string{}, nil)
	refs.EXPECT().MapConfigSecretsRefs(mock.Anything, configSecretsField, map[string]string{keyAPIToken: "$missing:key"}, testNamespace).
		Return(nil, errors.New("failed to get secret missing"))

	h := NewSyncAuthFlowExecutions(kc, refs)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `execution 2 ("sms-auth"): unable to map config secrets: failed to get secret missing`)
	assert.Empty(t, flow.Status.ConfigSecretsHash)
}

// With an enforcing allowlist, a reference outside a credential key is denied before any
// Keycloak read or write, even when another execution is clean.
func TestSyncAuthFlowExecutions_SecretRef_GuardDeniesNonCredentialKey(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Namespace = testNamespace
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "identity-provider-redirector", Requirement: "ALTERNATIVE", Priority: 10,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{Alias: "a", Config: map[string]string{keyDefaultProvider: "github"}}},
		{Authenticator: "sms-auth", Requirement: "REQUIRED", Priority: 20,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{Alias: "b", Config: map[string]string{keyAPIToken: testSecretRef}}},
	}

	guard, err := destination.New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	secretRef, err := secretref.NewSecretRef(fake.NewClientBuilder().Build(), guard)
	require.NoError(t, err)

	h := NewSyncAuthFlowExecutions(kc, secretRef)
	err = h.Serve(context.Background(), flow, testRealmName)
	require.ErrorIs(t, err, destination.ErrNotAllowed)
	assert.Contains(t, err.Error(), `execution 1 ("sms-auth")`)
	assert.Contains(t, err.Error(), configSecretsField+".apiToken")
}

// With an enforcing allowlist, a reference in a credential key resolves and is written.
func TestSyncAuthFlowExecutions_SecretRef_GuardAllowsCredentialKey(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Namespace = testNamespace
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "registration-recaptcha-action", Requirement: "REQUIRED", Priority: 10,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{
				Alias:  "recaptcha",
				Config: map[string]string{"site.key": "site", "secret.key": "$recaptcha:secretKey"},
			}},
	}

	guard, err := destination.New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "recaptcha", Namespace: testNamespace},
		Data:       map[string][]byte{"secretKey": []byte("real-secret")},
	}
	secretRef, err := secretref.NewSecretRef(fake.NewClientBuilder().WithObjects(secret).Build(), guard)
	require.NoError(t, err)

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			nonFlowExec("exec-1", "registration-recaptcha-action", "REQUIRED", 10, nil),
		}, nil, nil)

	resolvedConfig := map[string]string{"site.key": "site", "secret.key": "real-secret"}
	mockFlows.EXPECT().CreateExecutionConfig(context.Background(), testRealmName, "exec-1",
		keycloakapi.AuthenticatorConfigRepresentation{Alias: ptr.To("recaptcha"), Config: &resolvedConfig},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, secretRef)
	require.NoError(t, h.Serve(context.Background(), flow, testRealmName))
	assert.NotEmpty(t, flow.Status.ConfigSecretsHash)
}

// Flow-type entries are child flows managed by their own CR; their config is never resolved.
func TestSyncAuthFlowExecutions_SecretRef_FlowTypeEntryNotResolved(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Namespace = testNamespace
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Alias: "child", AuthenticatorFlow: true, Requirement: "ALTERNATIVE", Priority: 5,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{Alias: "x", Config: map[string]string{keyAPIToken: "$never:resolved"}}},
		{Authenticator: "basic-auth", Requirement: "REQUIRED", Priority: 10},
	}

	childExec := keycloakapi.AuthenticationExecutionInfoRepresentation{
		Id: ptr.To("child-1"), DisplayName: ptr.To("child"), AuthenticationFlow: ptr.To(true),
		Level: ptr.To(int32(0)), Requirement: ptr.To("ALTERNATIVE"), Priority: ptr.To(int32(5)),
	}
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			childExec, nonFlowExec("exec-1", "basic-auth", "REQUIRED", 10, nil),
		}, nil, nil)

	// The only non-flow entry has no config: the resolver must not be called at all.
	refs := secretrefmocks.NewMockRefClient(t)

	h := NewSyncAuthFlowExecutions(kc, refs)
	require.NoError(t, h.Serve(context.Background(), flow, testRealmName))
}

// The same config key in two executions yields two hash entries keyed by spec index, and
// the sort by priority keeps each resolved config with its own execution.
func TestSyncAuthFlowExecutions_SecretRef_TwoExecutionsKeepOwnConfig(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Namespace = testNamespace
	flow.Status.ID = testFlowID
	flow.Spec.Alias = testFlowAlias
	// Listed in reverse priority order on purpose.
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "sms-auth", Requirement: "REQUIRED", Priority: 20,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{Alias: "second", Config: map[string]string{keyAPIToken: "$b:key"}}},
		{Authenticator: "sms-auth", Requirement: "REQUIRED", Priority: 10,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{Alias: "first", Config: map[string]string{keyAPIToken: "$a:key"}}},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	add := func(priority int32) keycloakapi.AuthenticationExecutionRepresentation {
		return keycloakapi.AuthenticationExecutionRepresentation{
			Authenticator: ptr.To("sms-auth"), ParentFlow: ptr.To(testFlowID),
			Requirement: ptr.To("REQUIRED"), Priority: ptr.To(priority),
		}
	}
	mockFlows.EXPECT().AddExecutionToFlow(context.Background(), testRealmName, add(10)).Return(locationResponse("exec-10"), nil)
	mockFlows.EXPECT().AddExecutionToFlow(context.Background(), testRealmName, add(20)).Return(locationResponse("exec-20"), nil)

	firstConfig := map[string]string{keyAPIToken: "value-a"}
	secondConfig := map[string]string{keyAPIToken: "value-b"}

	mockFlows.EXPECT().CreateExecutionConfig(context.Background(), testRealmName, "exec-10",
		keycloakapi.AuthenticatorConfigRepresentation{Alias: ptr.To("first"), Config: &firstConfig}).Return(nil, nil)
	mockFlows.EXPECT().CreateExecutionConfig(context.Background(), testRealmName, "exec-20",
		keycloakapi.AuthenticatorConfigRepresentation{Alias: ptr.To("second"), Config: &secondConfig}).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc, resolving(t, map[string]string{"$a:key": "value-a", "$b:key": "value-b"}))
	require.NoError(t, h.Serve(context.Background(), flow, testRealmName))

	assert.Equal(t, secretref.ValuesHashSingle(map[string]string{"0/apiToken": tokenOf("$b:key"), "1/apiToken": tokenOf("$a:key")}),
		flow.Status.ConfigSecretsHash)
}

func TestSyncAuthFlowExecutions_SecretRef_HashNotStampedWhenWriteFails(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Namespace = testNamespace
	flow.Status.ID = testFlowID
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "sms-auth", Requirement: "REQUIRED", Priority: 10,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{Alias: "first", Config: map[string]string{keyAPIToken: "$a:key"}}},
		{Authenticator: "sms-auth", Requirement: "REQUIRED", Priority: 20,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{Alias: "second", Config: map[string]string{keyAPIToken: "$b:key"}}},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			nonFlowExec("exec-10", "sms-auth", "REQUIRED", 10, nil),
			nonFlowExec("exec-20", "sms-auth", "REQUIRED", 20, nil),
		}, nil, nil)

	mockFlows.EXPECT().CreateExecutionConfig(context.Background(), testRealmName, "exec-10", mock.Anything).Return(nil, nil)
	mockFlows.EXPECT().CreateExecutionConfig(context.Background(), testRealmName, "exec-20", mock.Anything).
		Return(nil, errors.New("keycloak down"))

	h := NewSyncAuthFlowExecutions(kc, resolving(t, map[string]string{"$a:key": "value-a", "$b:key": "value-b"}))
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Empty(t, flow.Status.ConfigSecretsHash)
}
