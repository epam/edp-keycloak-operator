package chain

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
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
	kClient := &keycloakv2.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(nil, nil)

	mockComponents.EXPECT().
		CreateComponent(context.Background(), testRealmName, keycloakv2.ComponentRepresentation{
			Name:         ptr.To(testComponentName),
			ProviderId:   ptr.To(testProviderID),
			ProviderType: ptr.To(testProviderType),
			Config:       ptr.To(keycloakv2.MultivaluedHashMapStringString{}),
		}).
		Return(&keycloakv2.Response{
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
	kClient := &keycloakv2.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()

	existing := &keycloakv2.ComponentRepresentation{
		Id:           ptr.To(testComponentID),
		Name:         ptr.To(testComponentName),
		ProviderId:   ptr.To(testProviderID),
		ProviderType: ptr.To(testProviderType),
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(existing, nil)

	mockComponents.EXPECT().
		UpdateComponent(context.Background(), testRealmName, testComponentID, keycloakv2.ComponentRepresentation{
			Id:           ptr.To(testComponentID),
			Name:         ptr.To(testComponentName),
			ProviderId:   ptr.To(testProviderID),
			ProviderType: ptr.To(testProviderType),
			Config:       ptr.To(keycloakv2.MultivaluedHashMapStringString{}),
		}).
		Return(nil, nil)

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testComponentID, component.Status.ID)
}

func TestCreateOrUpdateComponent_Serve_FindByNameError(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakv2.KeycloakClient{RealmComponents: mockComponents}
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
	kClient := &keycloakv2.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(nil, nil)

	mockComponents.EXPECT().
		CreateComponent(context.Background(), testRealmName, keycloakv2.ComponentRepresentation{
			Name:         ptr.To(testComponentName),
			ProviderId:   ptr.To(testProviderID),
			ProviderType: ptr.To(testProviderType),
			Config:       ptr.To(keycloakv2.MultivaluedHashMapStringString{}),
		}).
		Return(nil, errors.New("create error"))

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create realm component")
}

func TestCreateOrUpdateComponent_Serve_UpdateError(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakv2.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()

	existing := &keycloakv2.ComponentRepresentation{
		Id:           ptr.To(testComponentID),
		Name:         ptr.To(testComponentName),
		ProviderId:   ptr.To(testProviderID),
		ProviderType: ptr.To(testProviderType),
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(existing, nil)

	mockComponents.EXPECT().
		UpdateComponent(context.Background(), testRealmName, testComponentID, keycloakv2.ComponentRepresentation{
			Id:           ptr.To(testComponentID),
			Name:         ptr.To(testComponentName),
			ProviderId:   ptr.To(testProviderID),
			ProviderType: ptr.To(testProviderType),
			Config:       ptr.To(keycloakv2.MultivaluedHashMapStringString{}),
		}).
		Return(nil, errors.New("update error"))

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update realm component")
}

func TestCreateOrUpdateComponent_Serve_SecretRefError(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakv2.KeycloakClient{RealmComponents: mockComponents}
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
	kClient := &keycloakv2.KeycloakClient{
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
		Return(&keycloakv2.RealmRepresentation{Id: &parentRealmID}, nil, nil)

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(nil, nil)

	mockComponents.EXPECT().
		CreateComponent(context.Background(), testRealmName, keycloakv2.ComponentRepresentation{
			Name:         ptr.To(testComponentName),
			ProviderId:   ptr.To(testProviderID),
			ProviderType: ptr.To(testProviderType),
			Config:       ptr.To(keycloakv2.MultivaluedHashMapStringString{}),
			ParentId:     &parentRealmID,
		}).
		Return(&keycloakv2.Response{
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
	kClient := &keycloakv2.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	parentID := "parent-comp-id-789"
	component := baseComponent()
	component.Spec.ParentRef = &keycloakApi.ParentComponent{
		Kind: keycloakApi.KeycloakRealmComponentKind,
		Name: "parent-component",
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, "parent-component").
		Return(&keycloakv2.ComponentRepresentation{Id: &parentID}, nil)

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(nil, nil)

	mockComponents.EXPECT().
		CreateComponent(context.Background(), testRealmName, keycloakv2.ComponentRepresentation{
			Name:         ptr.To(testComponentName),
			ProviderId:   ptr.To(testProviderID),
			ProviderType: ptr.To(testProviderType),
			Config:       ptr.To(keycloakv2.MultivaluedHashMapStringString{}),
			ParentId:     &parentID,
		}).
		Return(&keycloakv2.Response{
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
	kClient := &keycloakv2.KeycloakClient{RealmComponents: mockComponents}
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
	kClient := &keycloakv2.KeycloakClient{}
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
	kClient := &keycloakv2.KeycloakClient{}
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
	kClient := &keycloakv2.KeycloakClient{RealmComponents: mockComponents}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	component := baseComponent()
	component.Spec.ParentRef = &keycloakApi.ParentComponent{
		Kind: keycloakApi.KeycloakRealmComponentKind,
		Name: "parent-no-id",
	}

	// returns a component representation with nil Id
	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, "parent-no-id").
		Return(&keycloakv2.ComponentRepresentation{Name: ptr.To("parent-no-id")}, nil)

	h := NewCreateOrUpdateComponent(fakeClient, kClient, &fakeSecretRefClient{})
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no ID")
}
