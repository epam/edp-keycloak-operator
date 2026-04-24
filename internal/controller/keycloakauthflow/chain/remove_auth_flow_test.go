package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"k8s.io/utils/ptr"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))

	return s
}

func TestRemoveAuthFlow_Serve_PreserveOnDeletion(t *testing.T) {
	kc := &keycloakapi.KeycloakClient{}
	k8sClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	flow := &keycloakApi.KeycloakAuthFlow{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"edp.epam.com/preserve-resources-on-deletion": "true",
			},
		},
	}
	flow.Spec.Alias = testFlowAlias

	h := NewRemoveAuthFlow(kc, k8sClient)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestRemoveAuthFlow_Serve_TopLevel_Success(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	mockRealms := mocks.NewMockRealmClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows, Realms: mockRealms}
	k8sClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Status.ID = testFlowID

	// unsetBrowserFlow: realm.BrowserFlow != alias → no update
	mockRealms.EXPECT().GetRealm(context.Background(), testRealmName).
		Return(&keycloakapi.RealmRepresentation{BrowserFlow: ptr.To("other-flow")}, nil, nil)

	// DeleteAuthFlow
	mockFlows.EXPECT().DeleteAuthFlow(context.Background(), testRealmName, testFlowID).
		Return(nil, nil)

	h := NewRemoveAuthFlow(kc, k8sClient)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestRemoveAuthFlow_Serve_TopLevel_BrowserFlowUnset(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	mockRealms := mocks.NewMockRealmClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows, Realms: mockRealms}
	k8sClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Status.ID = testFlowID

	// unsetBrowserFlow: realm.BrowserFlow == alias → must replace
	mockRealms.EXPECT().GetRealm(context.Background(), testRealmName).
		Return(&keycloakapi.RealmRepresentation{BrowserFlow: ptr.To(testFlowAlias)}, nil, nil)

	// GetAuthFlows called inside unsetBrowserFlow
	mockFlows.EXPECT().GetAuthFlows(context.Background(), testRealmName).
		Return([]keycloakapi.AuthFlowRepresentation{
			{Alias: ptr.To(testFlowAlias), Id: ptr.To(testFlowID)},
			{Alias: ptr.To("other-flow"), Id: ptr.To("other-id")},
		}, nil, nil)

	mockRealms.EXPECT().UpdateRealm(context.Background(), testRealmName,
		keycloakapi.RealmRepresentation{BrowserFlow: ptr.To("other-flow")},
	).Return(nil, nil)

	// DeleteAuthFlow
	mockFlows.EXPECT().DeleteAuthFlow(context.Background(), testRealmName, testFlowID).
		Return(nil, nil)

	h := NewRemoveAuthFlow(kc, k8sClient)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestRemoveAuthFlow_Serve_TopLevel_EmptyStatusID(t *testing.T) {
	kc := &keycloakapi.KeycloakClient{}
	k8sClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	// Status.ID intentionally left empty — deletion should be skipped

	h := NewRemoveAuthFlow(kc, k8sClient)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err) // graceful skip
}

func TestRemoveAuthFlow_Serve_ChildFlow_Success(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}
	k8sClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testChildAlias
	flow.Spec.ParentName = testParentFlow

	// GetFlowExecutions: finds execution matching alias
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testParentFlow).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			{DisplayName: ptr.To(testChildAlias), Id: ptr.To("child-exec-id")},
		}, nil, nil)

	mockFlows.EXPECT().DeleteExecution(context.Background(), testRealmName, "child-exec-id").
		Return(nil, nil)

	h := NewRemoveAuthFlow(kc, k8sClient)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestRemoveAuthFlow_Serve_ChildFlow_ParentNotFound(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}
	k8sClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testChildAlias
	flow.Spec.ParentName = testParentFlow

	// Parent flow returns 404 → graceful skip
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testParentFlow).
		Return(nil, nil, &keycloakapi.ApiError{Code: 404})

	h := NewRemoveAuthFlow(kc, k8sClient)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestRemoveAuthFlow_Serve_ChildFlow_NotInParent(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}
	k8sClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testChildAlias
	flow.Spec.ParentName = testParentFlow

	// Parent exists but child alias not present → graceful skip
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testParentFlow).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			{DisplayName: ptr.To("other-child"), Id: ptr.To("other-id")},
		}, nil, nil)

	h := NewRemoveAuthFlow(kc, k8sClient)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestRemoveAuthFlow_Serve_ChildK8sFlowExists(t *testing.T) {
	kc := &keycloakapi.KeycloakClient{}

	childFlow := keycloakApi.KeycloakAuthFlow{}
	childFlow.Spec.Alias = "dependent-child"
	childFlow.Spec.ParentName = testFlowAlias
	childFlow.Spec.RealmRef.Name = testRealmName
	childFlow.Spec.RealmRef.Kind = "KeycloakRealm"

	s := newScheme(t)
	k8sClient := fake.NewClientBuilder().WithScheme(s).WithObjects(&childFlow).Build()

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.RealmRef.Name = testRealmName
	flow.Spec.RealmRef.Kind = "KeycloakRealm"

	h := NewRemoveAuthFlow(kc, k8sClient)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete flow")
}

func TestRemoveAuthFlow_Serve_GetFlowExecutionsError(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}
	k8sClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testChildAlias
	flow.Spec.ParentName = testParentFlow

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testParentFlow).
		Return(nil, nil, errors.New("api error"))

	h := NewRemoveAuthFlow(kc, k8sClient)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get parent flow executions")
}
