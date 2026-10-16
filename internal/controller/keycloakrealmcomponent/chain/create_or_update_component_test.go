package chain

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

const (
	testComponentName = "test-component"
	testComponentID   = "comp-id-123"
	testRealmName     = "test-realm"
	testRealmID       = "realm-internal-id"
	testProviderID    = "ldap"
	testProviderType  = "org.keycloak.storage.UserStorageProvider"
	testNamespace     = "test-ns"
)

// noSecretRefsHash is the handler-computed hash for a config with no secret refs.
var noSecretRefsHash = secretref.ValuesHash(nil)

type fakeSecretRefClient struct {
	err error
}

func (f *fakeSecretRefClient) MapComponentConfigSecretsRefs(_ context.Context, _ string, _ map[string][]string, _ string) (map[string][]string, error) {
	return nil, f.err
}

// mutatingSecretRefClient simulates secret-ref resolution by rewriting config values in place,
// mirroring the real SecretRef client's mutate-in-place contract; versions are the returned
// secret version tokens.
type mutatingSecretRefClient struct {
	mutate   func(config map[string][]string)
	versions map[string][]string
}

func (f *mutatingSecretRefClient) MapComponentConfigSecretsRefs(_ context.Context, _ string, config map[string][]string, _ string) (map[string][]string, error) {
	if f.mutate != nil {
		f.mutate(config)
	}

	return f.versions, nil
}

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))

	return s
}

func baseComponent() *keycloakApi.KeycloakRealmComponent {
	return &keycloakApi.KeycloakRealmComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testComponentName,
			Namespace: testNamespace,
		},
		Spec: keycloakApi.KeycloakComponentSpec{
			Name:         testComponentName,
			ProviderID:   testProviderID,
			ProviderType: testProviderType,
			RealmRef:     testRealmRef(),
		},
	}
}

func testRealmRef() common.RealmRef {
	return common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: "test-realm-cr"}
}

// byName is the scoped lookup for testComponentName the handler issues when status.ID does not resolve.
func byName(parentID string) *keycloakapi.GetComponentsParams {
	return &keycloakapi.GetComponentsParams{Name: ptr.To(testComponentName), Parent: ptr.To(parentID)}
}

func createdResponse(id string) *keycloakapi.Response {
	return &keycloakapi.Response{
		HTTPResponse: &http.Response{
			Header: http.Header{
				"Location": []string{"http://localhost/admin/realms/test-realm/components/" + id},
			},
		},
	}
}

func inSyncExisting() keycloakapi.ComponentRepresentation {
	return keycloakapi.ComponentRepresentation{
		Id:           ptr.To(testComponentID),
		Name:         ptr.To(testComponentName),
		ProviderId:   ptr.To(testProviderID),
		ProviderType: ptr.To(testProviderType),
		ParentId:     ptr.To(testRealmID),
	}
}

func desiredRepr(id *string, parentID string) keycloakapi.ComponentRepresentation {
	return keycloakapi.ComponentRepresentation{
		Id:           id,
		Name:         ptr.To(testComponentName),
		ProviderId:   ptr.To(testProviderID),
		ProviderType: ptr.To(testProviderType),
		Config:       ptr.To(keycloakapi.MultivaluedHashMapStringString{}),
		ParentId:     ptr.To(parentID),
	}
}

func TestCreateOrUpdateComponent_Serve_CreateNew(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return(nil, nil, nil)

	mockComponents.EXPECT().
		CreateComponent(context.Background(), testRealmName, desiredRepr(nil, testRealmID)).
		Return(createdResponse(testComponentID), nil)

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, testRealmID)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testComponentID, component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_UpdateExisting(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()
	// Pre-seed the hash so it can't itself be the reason the update fires: the test isolates
	// the providerId drift.
	component.Status.ConfigSecretsHash = noSecretRefsHash

	existing := inSyncExisting()
	existing.ProviderId = ptr.To("stale-provider")

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return([]keycloakapi.ComponentRepresentation{existing}, nil, nil)

	mockComponents.EXPECT().
		UpdateComponent(context.Background(), testRealmName, testComponentID, desiredRepr(ptr.To(testComponentID), testRealmID)).
		Return(nil, nil)

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, testRealmID)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testComponentID, component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_TopLevelParentIdAutoFillDoesNotForceUpdate(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	mockRealms := mocks.NewMockRealmClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents, Realms: mockRealms}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	component.Status.ConfigSecretsHash = noSecretRefsHash

	// No spec.parentRef: the parent resolves to the realm's internal ID, which is what
	// Keycloak auto-fills for top-level components.
	mockRealms.EXPECT().
		GetRealm(context.Background(), testRealmName).
		Return(&keycloakapi.RealmRepresentation{Id: ptr.To(testRealmID)}, nil, nil)

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return([]keycloakapi.ComponentRepresentation{inSyncExisting()}, nil, nil)

	parent, err := ResolveParent(context.Background(), fakeClient, kClient, component, testRealmName)
	require.NoError(t, err)

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, parent.ID)
	err = h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testComponentID, component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_SkipUpdateWhenInSync(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()
	component.Generation = 5
	component.Status.ObservedGeneration = 5
	component.Spec.Config = map[string][]string{"key1": {"val1", "val2"}}
	component.Status.ConfigSecretsHash = noSecretRefsHash

	// Fetched config value order differs from spec: order-insensitive comparison still matches.
	existing := inSyncExisting()
	existing.Config = ptr.To(keycloakapi.MultivaluedHashMapStringString{"key1": {"val2", "val1"}})

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return([]keycloakapi.ComponentRepresentation{existing}, nil, nil)

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, testRealmID)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testComponentID, component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_ParentIdAlwaysCompared(t *testing.T) {
	tests := []struct {
		name           string
		existingParent *string
		desiredParent  string
	}{
		{name: "top-level component attached elsewhere moves back to the realm", existingParent: ptr.To("old-parent-id"), desiredParent: testRealmID},
		{name: "missing parentId is written", existingParent: nil, desiredParent: testRealmID},
		{name: "re-parent to another component", existingParent: ptr.To("old-parent-id"), desiredParent: "new-parent-id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockComponents := mocks.NewMockRealmComponentsClient(t)
			kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

			component := baseComponent()
			// Generation and hash are in sync: the ParentId mismatch is the only reason to update.
			component.Generation = 4
			component.Status.ObservedGeneration = 4
			component.Status.ConfigSecretsHash = noSecretRefsHash

			existing := inSyncExisting()
			existing.Config = ptr.To(keycloakapi.MultivaluedHashMapStringString{})
			existing.ParentId = tt.existingParent

			mockComponents.EXPECT().
				GetComponents(context.Background(), testRealmName, byName(tt.desiredParent)).
				Return([]keycloakapi.ComponentRepresentation{existing}, nil, nil)

			mockComponents.EXPECT().
				UpdateComponent(context.Background(), testRealmName, testComponentID, desiredRepr(ptr.To(testComponentID), tt.desiredParent)).
				Return(nil, nil)

			h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, tt.desiredParent)
			require.NoError(t, h.Serve(context.Background(), component, testRealmName))
		})
	}
}

func TestCreateOrUpdateComponent_Serve_FoundByStatusID(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()
	component.Generation = 2
	component.Status.ObservedGeneration = 2
	component.Status.ConfigSecretsHash = noSecretRefsHash
	component.Status.ID = testComponentID

	// Renamed and attached to another parent in Keycloak: status.ID still identifies it.
	existing := inSyncExisting()
	existing.Name = ptr.To("old-name")
	existing.ParentId = ptr.To("old-parent-id")

	mockComponents.EXPECT().
		GetComponent(context.Background(), testRealmName, testComponentID).
		Return(&existing, nil, nil)

	mockComponents.EXPECT().
		UpdateComponent(context.Background(), testRealmName, testComponentID, desiredRepr(ptr.To(testComponentID), "new-parent-id")).
		Return(nil, nil)

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, "new-parent-id")
	require.NoError(t, h.Serve(context.Background(), component, testRealmName))
	assert.Equal(t, testComponentID, component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_StaleStatusIDFallsBackToScopedLookup(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()
	component.Status.ConfigSecretsHash = noSecretRefsHash
	component.Status.ID = "stale-id"

	mockComponents.EXPECT().
		GetComponent(context.Background(), testRealmName, "stale-id").
		Return(nil, nil, &keycloakapi.ApiError{Code: http.StatusNotFound, Message: "Not Found"})

	other := inSyncExisting()
	other.Id = ptr.To("other-id")
	other.Name = ptr.To(testComponentName + "-other")

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return([]keycloakapi.ComponentRepresentation{other, inSyncExisting()}, nil, nil)

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, testRealmID)
	require.NoError(t, h.Serve(context.Background(), component, testRealmName))
	assert.Equal(t, testComponentID, component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_StaleStatusIDNotFoundCreates(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()
	component.Status.ID = "stale-id"

	mockComponents.EXPECT().
		GetComponent(context.Background(), testRealmName, "stale-id").
		Return(nil, nil, keycloakapi.ErrNotFound)

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return(nil, nil, nil)

	mockComponents.EXPECT().
		CreateComponent(context.Background(), testRealmName, desiredRepr(nil, testRealmID)).
		Return(createdResponse("new-id"), nil)

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, testRealmID)
	require.NoError(t, h.Serve(context.Background(), component, testRealmName))
	assert.Equal(t, "new-id", component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_GetByStatusIDError(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()
	component.Status.ID = testComponentID

	mockComponents.EXPECT().
		GetComponent(context.Background(), testRealmName, testComponentID).
		Return(nil, nil, errors.New("api error"))

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, testRealmID)
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get component "+testComponentID)
}

func TestCreateOrUpdateComponent_Serve_SameNameUnderDifferentParents(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	childUnder := func(id, parentID string) keycloakapi.ComponentRepresentation {
		c := inSyncExisting()
		c.Id = ptr.To(id)
		c.ParentId = ptr.To(parentID)

		return c
	}

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName("parent-a")).
		Return([]keycloakapi.ComponentRepresentation{childUnder("child-a", "parent-a")}, nil, nil)
	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName("parent-b")).
		Return([]keycloakapi.ComponentRepresentation{childUnder("child-b", "parent-b")}, nil, nil)

	for parentID, wantID := range map[string]string{"parent-a": "child-a", "parent-b": "child-b"} {
		t.Run(parentID, func(t *testing.T) {
			component := baseComponent()
			component.Status.ConfigSecretsHash = noSecretRefsHash

			h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, parentID)
			require.NoError(t, h.Serve(context.Background(), component, testRealmName))
			assert.Equal(t, wantID, component.Status.ID)
		})
	}
}

func TestCreateOrUpdateComponent_Serve_ForceUpdateOnGenerationBump(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()
	component.Generation = 6
	component.Status.ObservedGeneration = 5
	component.Spec.Config = map[string][]string{"key1": {"val1", "val2"}}
	component.Status.ConfigSecretsHash = noSecretRefsHash

	existing := inSyncExisting()
	existing.Config = ptr.To(keycloakapi.MultivaluedHashMapStringString{"key1": {"val1", "val2"}})

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return([]keycloakapi.ComponentRepresentation{existing}, nil, nil)

	mockComponents.EXPECT().
		UpdateComponent(context.Background(), testRealmName, testComponentID, mock.Anything).
		Return(nil, nil)

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, testRealmID)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
}

func TestCreateOrUpdateComponent_Serve_SecretRefConfigKeySkipped(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()
	component.Generation = 2
	component.Status.ObservedGeneration = 2
	component.Spec.Config = map[string][]string{"bindCredential": {"$secret:key"}}
	secretVersions := map[string][]string{"bindCredential": {"secret:secret:key@uid-1@100"}}
	component.Status.ConfigSecretsHash = secretref.ValuesHash(secretVersions)

	// Keycloak masks the secret-typed value on GET regardless of the resolved value.
	existing := inSyncExisting()
	existing.Config = ptr.To(keycloakapi.MultivaluedHashMapStringString{
		"bindCredential": {keycloakapi.MaskedSecretValue},
	})

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return([]keycloakapi.ComponentRepresentation{existing}, nil, nil)

	secretRefClient := &mutatingSecretRefClient{
		mutate: func(config map[string][]string) {
			config["bindCredential"] = []string{"resolved-value"}
		},
		versions: secretVersions,
	}

	h := NewCreateOrUpdateComponent(kClient, secretRefClient, testRealmID)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
}

func TestCreateOrUpdateComponent_Serve_HashDriftForcesUpdate(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()
	component.Generation = 2
	component.Status.ObservedGeneration = 2
	component.Spec.Config = map[string][]string{"bindCredential": {"$secret:key"}}
	component.Status.ConfigSecretsHash = "stale-hash"

	existing := inSyncExisting()
	existing.Config = ptr.To(keycloakapi.MultivaluedHashMapStringString{
		"bindCredential": {keycloakapi.MaskedSecretValue},
	})

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return([]keycloakapi.ComponentRepresentation{existing}, nil, nil)

	mockComponents.EXPECT().
		UpdateComponent(context.Background(), testRealmName, testComponentID, mock.Anything).
		Return(nil, nil)

	secretRefClient := &mutatingSecretRefClient{
		mutate: func(config map[string][]string) {
			config["bindCredential"] = []string{"resolved-value"}
		},
		versions: map[string][]string{"bindCredential": {"secret:secret:key@uid-1@101"}},
	}

	h := NewCreateOrUpdateComponent(kClient, secretRefClient, testRealmID)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
	assert.NotEqual(t, "stale-hash", component.Status.ConfigSecretsHash)
}

func TestCreateOrUpdateComponent_Serve_MaskedFetchedValueSkipped(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()
	component.Generation = 3
	component.Status.ObservedGeneration = 3
	component.Spec.Config = map[string][]string{"bindCredential": {"plain-secret-value"}}
	component.Status.ConfigSecretsHash = noSecretRefsHash

	// A plain-literal (non secret-ref) value is still masked by Keycloak on GET.
	existing := inSyncExisting()
	existing.Config = ptr.To(keycloakapi.MultivaluedHashMapStringString{
		"bindCredential": {keycloakapi.MaskedSecretValue},
	})

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return([]keycloakapi.ComponentRepresentation{existing}, nil, nil)

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, testRealmID)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
}

func TestCreateOrUpdateComponent_Serve_ScopedLookupError(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return(nil, nil, errors.New("api error"))

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, testRealmID)
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find component")
}

func TestCreateOrUpdateComponent_Serve_CreateError(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return(nil, nil, nil)

	mockComponents.EXPECT().
		CreateComponent(context.Background(), testRealmName, desiredRepr(nil, testRealmID)).
		Return(nil, errors.New("create error"))

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, testRealmID)
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create realm component")
}

func TestCreateOrUpdateComponent_Serve_CreateWithoutLocationHeader(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return(nil, nil, nil)

	// Keycloak reports the new id in the Location header only. Without it status.ID would
	// stay empty and every later update would address an empty component id.
	mockComponents.EXPECT().
		CreateComponent(context.Background(), testRealmName, desiredRepr(nil, testRealmID)).
		Return(&keycloakapi.Response{HTTPResponse: &http.Response{Header: http.Header{}}}, nil)

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, testRealmID)
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Location header missing or empty")
	assert.Empty(t, component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_UpdateError(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	component := baseComponent()
	// Pre-seed the hash so it can't itself be the reason the update fires: the test isolates
	// the providerId drift.
	component.Status.ConfigSecretsHash = noSecretRefsHash

	existing := inSyncExisting()
	existing.ProviderId = ptr.To("stale-provider")

	mockComponents.EXPECT().
		GetComponents(context.Background(), testRealmName, byName(testRealmID)).
		Return([]keycloakapi.ComponentRepresentation{existing}, nil, nil)

	mockComponents.EXPECT().
		UpdateComponent(context.Background(), testRealmName, testComponentID, desiredRepr(ptr.To(testComponentID), testRealmID)).
		Return(nil, errors.New("update error"))

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{}, testRealmID)
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update realm component")
}

func TestCreateOrUpdateComponent_Serve_SecretRefError(t *testing.T) {
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mocks.NewMockRealmComponentsClient(t)}

	component := baseComponent()

	h := NewCreateOrUpdateComponent(kClient, &fakeSecretRefClient{err: errors.New("secret error")}, testRealmID)
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to map config secrets")
}
