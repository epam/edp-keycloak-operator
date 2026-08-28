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

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

const (
	testComponentName = "test-component"
	testComponentID   = "comp-id-123"
	testRealmName     = "test-realm"
	testProviderID    = "ldap"
	testProviderType  = "org.keycloak.storage.UserStorageProvider"
	testNamespace     = "test-ns"
)

type fakeSecretRefClient struct {
	err error
}

func (f *fakeSecretRefClient) MapComponentConfigSecretsRefs(_ context.Context, _ map[string][]string, _ string) error {
	return f.err
}

// mutatingSecretRefClient simulates secret-ref resolution by rewriting config values in place,
// mirroring the real SecretRef client's mutate-in-place contract.
type mutatingSecretRefClient struct {
	mutate func(config map[string][]string)
}

func (f *mutatingSecretRefClient) MapComponentConfigSecretsRefs(_ context.Context, config map[string][]string, _ string) error {
	if f.mutate != nil {
		f.mutate(config)
	}

	return nil
}

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))

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
		},
	}
}

func TestCreateOrUpdateComponent_Serve_CreateNew(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(nil, nil)

	mockComponents.EXPECT().
		CreateComponent(context.Background(), testRealmName, keycloakapi.ComponentRepresentation{
			Name:         ptr.To(testComponentName),
			ProviderId:   ptr.To(testProviderID),
			ProviderType: ptr.To(testProviderType),
			Config:       ptr.To(keycloakapi.MultivaluedHashMapStringString{}),
		}).
		Return(&keycloakapi.Response{
			HTTPResponse: &http.Response{
				Header: http.Header{
					"Location": []string{"http://localhost/admin/realms/test-realm/components/" + testComponentID},
				},
			},
		}, nil)

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testComponentID, component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_UpdateExisting(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	// Pre-seed the hash so it can't itself be the reason the update fires: the test isolates
	// the providerId drift.
	component.Status.ConfigSecretsHash = secretref.ConfigSecretsHash(component.Spec.Config, component.Spec.Config)

	// Fetched providerId differs from spec: drift forces the update.
	existing := &keycloakapi.ComponentRepresentation{
		Id:           ptr.To(testComponentID),
		Name:         ptr.To(testComponentName),
		ProviderId:   ptr.To("stale-provider"),
		ProviderType: ptr.To(testProviderType),
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(existing, nil)

	mockComponents.EXPECT().
		UpdateComponent(context.Background(), testRealmName, testComponentID, keycloakapi.ComponentRepresentation{
			Id:           ptr.To(testComponentID),
			Name:         ptr.To(testComponentName),
			ProviderId:   ptr.To(testProviderID),
			ProviderType: ptr.To(testProviderType),
			Config:       ptr.To(keycloakapi.MultivaluedHashMapStringString{}),
		}).
		Return(nil, nil)

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testComponentID, component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_TopLevelParentIdAutoFillDoesNotForceUpdate(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	component.Status.ConfigSecretsHash = secretref.ConfigSecretsHash(component.Spec.Config, component.Spec.Config)

	// No spec.ParentRef, but Keycloak auto-fills ParentId with the realm's internal ID for
	// top-level components: this must not be mistaken for spec drift.
	existing := &keycloakapi.ComponentRepresentation{
		Id:           ptr.To(testComponentID),
		Name:         ptr.To(testComponentName),
		ProviderId:   ptr.To(testProviderID),
		ProviderType: ptr.To(testProviderType),
		ParentId:     ptr.To("realm-internal-id"),
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(existing, nil)

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testComponentID, component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_SkipUpdateWhenInSync(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	component.Generation = 5
	component.Status.ObservedGeneration = 5
	component.Spec.Config = map[string][]string{"key1": {"val1", "val2"}}
	component.Status.ConfigSecretsHash = secretref.ConfigSecretsHash(component.Spec.Config, component.Spec.Config)

	// Fetched config value order differs from spec: order-insensitive comparison still matches.
	existing := &keycloakapi.ComponentRepresentation{
		Id:           ptr.To(testComponentID),
		Name:         ptr.To(testComponentName),
		ProviderId:   ptr.To(testProviderID),
		ProviderType: ptr.To(testProviderType),
		Config: ptr.To(keycloakapi.MultivaluedHashMapStringString{
			"key1": {"val2", "val1"},
		}),
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(existing, nil)

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testComponentID, component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_ForceUpdateOnGenerationBump(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	component.Generation = 6
	component.Status.ObservedGeneration = 5
	component.Spec.Config = map[string][]string{"key1": {"val1", "val2"}}
	component.Status.ConfigSecretsHash = secretref.ConfigSecretsHash(component.Spec.Config, component.Spec.Config)

	existing := &keycloakapi.ComponentRepresentation{
		Id:           ptr.To(testComponentID),
		Name:         ptr.To(testComponentName),
		ProviderId:   ptr.To(testProviderID),
		ProviderType: ptr.To(testProviderType),
		Config: ptr.To(keycloakapi.MultivaluedHashMapStringString{
			"key1": {"val1", "val2"},
		}),
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(existing, nil)

	mockComponents.EXPECT().
		UpdateComponent(context.Background(), testRealmName, testComponentID, mock.Anything).
		Return(nil, nil)

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
}

func TestCreateOrUpdateComponent_Serve_SecretRefConfigKeySkipped(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	component.Generation = 2
	component.Status.ObservedGeneration = 2
	component.Spec.Config = map[string][]string{"bindCredential": {"$secret:key"}}
	component.Status.ConfigSecretsHash = secretref.ConfigSecretsHash(
		map[string][]string{"bindCredential": {"$secret:key"}},
		map[string][]string{"bindCredential": {"resolved-value"}},
	)

	// Keycloak masks the secret-typed value on GET regardless of the resolved value.
	existing := &keycloakapi.ComponentRepresentation{
		Id:           ptr.To(testComponentID),
		Name:         ptr.To(testComponentName),
		ProviderId:   ptr.To(testProviderID),
		ProviderType: ptr.To(testProviderType),
		Config: ptr.To(keycloakapi.MultivaluedHashMapStringString{
			"bindCredential": {keycloakapi.MaskedSecretValue},
		}),
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(existing, nil)

	secretRefClient := &mutatingSecretRefClient{mutate: func(config map[string][]string) {
		config["bindCredential"] = []string{"resolved-value"}
	}}

	h := NewCreateOrUpdateComponent(fakeClient, kClient, secretRefClient)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
}

func TestCreateOrUpdateComponent_Serve_HashDriftForcesUpdate(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	component.Generation = 2
	component.Status.ObservedGeneration = 2
	component.Spec.Config = map[string][]string{"bindCredential": {"$secret:key"}}
	component.Status.ConfigSecretsHash = "stale-hash"

	existing := &keycloakapi.ComponentRepresentation{
		Id:           ptr.To(testComponentID),
		Name:         ptr.To(testComponentName),
		ProviderId:   ptr.To(testProviderID),
		ProviderType: ptr.To(testProviderType),
		Config: ptr.To(keycloakapi.MultivaluedHashMapStringString{
			"bindCredential": {keycloakapi.MaskedSecretValue},
		}),
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(existing, nil)

	mockComponents.EXPECT().
		UpdateComponent(context.Background(), testRealmName, testComponentID, mock.Anything).
		Return(nil, nil)

	secretRefClient := &mutatingSecretRefClient{mutate: func(config map[string][]string) {
		config["bindCredential"] = []string{"resolved-value"}
	}}

	h := NewCreateOrUpdateComponent(fakeClient, kClient, secretRefClient)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
	assert.NotEqual(t, "stale-hash", component.Status.ConfigSecretsHash)
}

func TestCreateOrUpdateComponent_Serve_MaskedFetchedValueSkipped(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	component.Generation = 3
	component.Status.ObservedGeneration = 3
	component.Spec.Config = map[string][]string{"bindCredential": {"plain-secret-value"}}
	component.Status.ConfigSecretsHash = secretref.ConfigSecretsHash(component.Spec.Config, component.Spec.Config)

	// A plain-literal (non secret-ref) value is still masked by Keycloak on GET.
	existing := &keycloakapi.ComponentRepresentation{
		Id:           ptr.To(testComponentID),
		Name:         ptr.To(testComponentName),
		ProviderId:   ptr.To(testProviderID),
		ProviderType: ptr.To(testProviderType),
		Config: ptr.To(keycloakapi.MultivaluedHashMapStringString{
			"bindCredential": {keycloakapi.MaskedSecretValue},
		}),
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(existing, nil)

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
}

func TestCreateOrUpdateComponent_Serve_FindByNameError(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(nil, errors.New("api error"))

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find component by name")
}

func TestCreateOrUpdateComponent_Serve_CreateError(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(nil, nil)

	mockComponents.EXPECT().
		CreateComponent(context.Background(), testRealmName, keycloakapi.ComponentRepresentation{
			Name:         ptr.To(testComponentName),
			ProviderId:   ptr.To(testProviderID),
			ProviderType: ptr.To(testProviderType),
			Config:       ptr.To(keycloakapi.MultivaluedHashMapStringString{}),
		}).
		Return(nil, errors.New("create error"))

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create realm component")
}

func TestCreateOrUpdateComponent_Serve_UpdateError(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	// Pre-seed the hash so it can't itself be the reason the update fires: the test isolates
	// the providerId drift.
	component.Status.ConfigSecretsHash = secretref.ConfigSecretsHash(component.Spec.Config, component.Spec.Config)

	// Fetched providerId differs from spec: drift forces the update.
	existing := &keycloakapi.ComponentRepresentation{
		Id:           ptr.To(testComponentID),
		Name:         ptr.To(testComponentName),
		ProviderId:   ptr.To("stale-provider"),
		ProviderType: ptr.To(testProviderType),
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(existing, nil)

	mockComponents.EXPECT().
		UpdateComponent(context.Background(), testRealmName, testComponentID, keycloakapi.ComponentRepresentation{
			Id:           ptr.To(testComponentID),
			Name:         ptr.To(testComponentName),
			ProviderId:   ptr.To(testProviderID),
			ProviderType: ptr.To(testProviderType),
			Config:       ptr.To(keycloakapi.MultivaluedHashMapStringString{}),
		}).
		Return(nil, errors.New("update error"))

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update realm component")
}

func TestCreateOrUpdateComponent_Serve_SecretRefError(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{err: errors.New("secret error")})
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to map config secrets")
}

func TestCreateOrUpdateComponent_Serve_ParentRefRealmKind(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	mockRealms := mocks.NewMockRealmClient(t)
	kClient := &keycloakapi.KeycloakClient{
		RealmComponents: mockComponents,
		Realms:          mockRealms,
	}

	parentRealmCR := &keycloakApi.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "parent-realm-cr",
			Namespace: testNamespace,
		},
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "parent-realm",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(parentRealmCR).Build()

	component := baseComponent()
	component.Spec.ParentRef = &keycloakApi.ParentComponent{
		Kind: keycloakApi.KeycloakRealmKind,
		Name: "parent-realm-cr",
	}

	parentRealmID := "realm-uuid-456"

	mockRealms.EXPECT().
		GetRealm(context.Background(), "parent-realm").
		Return(&keycloakapi.RealmRepresentation{Id: &parentRealmID}, nil, nil)

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(nil, nil)

	mockComponents.EXPECT().
		CreateComponent(context.Background(), testRealmName, keycloakapi.ComponentRepresentation{
			Name:         ptr.To(testComponentName),
			ProviderId:   ptr.To(testProviderID),
			ProviderType: ptr.To(testProviderType),
			Config:       ptr.To(keycloakapi.MultivaluedHashMapStringString{}),
			ParentId:     &parentRealmID,
		}).
		Return(&keycloakapi.Response{
			HTTPResponse: &http.Response{
				Header: http.Header{
					"Location": []string{"http://localhost/admin/realms/test-realm/components/" + testComponentID},
				},
			},
		}, nil)

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testComponentID, component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_ParentRefComponentKind(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	parentID := "parent-comp-id-789"
	component := baseComponent()
	component.Spec.ParentRef = &keycloakApi.ParentComponent{
		Kind: keycloakApi.KeycloakRealmComponentKind,
		Name: "parent-component",
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, "parent-component").
		Return(&keycloakapi.ComponentRepresentation{Id: &parentID}, nil)

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(nil, nil)

	mockComponents.EXPECT().
		CreateComponent(context.Background(), testRealmName, keycloakapi.ComponentRepresentation{
			Name:         ptr.To(testComponentName),
			ProviderId:   ptr.To(testProviderID),
			ProviderType: ptr.To(testProviderType),
			Config:       ptr.To(keycloakapi.MultivaluedHashMapStringString{}),
			ParentId:     &parentID,
		}).
		Return(&keycloakapi.Response{
			HTTPResponse: &http.Response{
				Header: http.Header{
					"Location": []string{"http://localhost/admin/realms/test-realm/components/" + testComponentID},
				},
			},
		}, nil)

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
}

func TestCreateOrUpdateComponent_Serve_ParentComponentNotFound(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	component.Spec.ParentRef = &keycloakApi.ParentComponent{
		Kind: keycloakApi.KeycloakRealmComponentKind,
		Name: "missing-parent",
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, "missing-parent").
		Return(nil, nil)

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parent component")
	assert.Contains(t, err.Error(), "not found")
}

func TestCreateOrUpdateComponent_Serve_UnsupportedParentKind(t *testing.T) {
	kClient := &keycloakapi.KeycloakClient{}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	component.Spec.ParentRef = &keycloakApi.ParentComponent{
		Kind: "UnsupportedKind",
		Name: "something",
	}

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not supported")
}

func TestCreateOrUpdateComponent_Serve_ParentRealmKind_K8sGetError(t *testing.T) {
	kClient := &keycloakapi.KeycloakClient{}
	// empty fake client — realm CR is not present, so k8sClient.Get will return NotFound
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	component.Spec.ParentRef = &keycloakApi.ParentComponent{
		Kind: keycloakApi.KeycloakRealmKind,
		Name: "missing-realm-cr",
	}

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to get parent realm")
}

func TestCreateOrUpdateComponent_Serve_ParentComponentKind_NilID(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	component.Spec.ParentRef = &keycloakApi.ParentComponent{
		Kind: keycloakApi.KeycloakRealmComponentKind,
		Name: "parent-no-id",
	}

	// returns a component representation with nil Id
	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, "parent-no-id").
		Return(&keycloakapi.ComponentRepresentation{Name: ptr.To("parent-no-id")}, nil)

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no ID")
}
