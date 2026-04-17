package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

func TestRemoveComponent_Serve_SuccessWithStatusID(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.APIClient{RealmComponents: mockComponents}

	component := &keycloakApi.KeycloakRealmComponent{
		ObjectMeta: metav1.ObjectMeta{Name: testComponentName},
		Status:     keycloakApi.KeycloakComponentStatus{ID: testComponentID},
	}

	mockComponents.EXPECT().
		DeleteComponent(context.Background(), testRealmName, testComponentID).
		Return(nil, nil)

	h := NewRemoveComponent(kClient)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
}

func TestRemoveComponent_Serve_SuccessFallbackByName(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.APIClient{RealmComponents: mockComponents}

	component := &keycloakApi.KeycloakRealmComponent{
		ObjectMeta: metav1.ObjectMeta{Name: testComponentName},
		Spec:       keycloakApi.KeycloakComponentSpec{Name: testComponentName},
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(&keycloakapi.ComponentRepresentation{Id: ptr.To(testComponentID)}, nil)

	mockComponents.EXPECT().
		DeleteComponent(context.Background(), testRealmName, testComponentID).
		Return(nil, nil)

	h := NewRemoveComponent(kClient)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
}

func TestRemoveComponent_Serve_ComponentNotFoundByName(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.APIClient{RealmComponents: mockComponents}

	component := &keycloakApi.KeycloakRealmComponent{
		ObjectMeta: metav1.ObjectMeta{Name: testComponentName},
		Spec:       keycloakApi.KeycloakComponentSpec{Name: testComponentName},
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(nil, nil)

	h := NewRemoveComponent(kClient)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
}

func TestRemoveComponent_Serve_DeleteNotFound(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.APIClient{RealmComponents: mockComponents}

	component := &keycloakApi.KeycloakRealmComponent{
		ObjectMeta: metav1.ObjectMeta{Name: testComponentName},
		Status:     keycloakApi.KeycloakComponentStatus{ID: testComponentID},
	}

	mockComponents.EXPECT().
		DeleteComponent(context.Background(), testRealmName, testComponentID).
		Return(nil, &keycloakapi.ApiError{Code: 404, Message: "Not Found"})

	h := NewRemoveComponent(kClient)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
}

func TestRemoveComponent_Serve_PreserveAnnotation(t *testing.T) {
	kClient := &keycloakapi.APIClient{}

	component := &keycloakApi.KeycloakRealmComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name: testComponentName,
			Annotations: map[string]string{
				objectmeta.PreserveResourcesOnDeletionAnnotation: "true",
			},
		},
		Status: keycloakApi.KeycloakComponentStatus{ID: testComponentID},
	}

	h := NewRemoveComponent(kClient)
	err := h.Serve(context.Background(), component, testRealmName)
	require.NoError(t, err)
	// no mock expectations — verify no API calls were made
}

func TestRemoveComponent_Serve_FindByNameError(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.APIClient{RealmComponents: mockComponents}

	component := &keycloakApi.KeycloakRealmComponent{
		ObjectMeta: metav1.ObjectMeta{Name: testComponentName},
		Spec:       keycloakApi.KeycloakComponentSpec{Name: testComponentName},
	}

	mockComponents.EXPECT().
		FindComponentByName(context.Background(), testRealmName, testComponentName).
		Return(nil, errors.New("lookup error"))

	h := NewRemoveComponent(kClient)
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to find component for deletion")
}

func TestRemoveComponent_Serve_DeleteError(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.APIClient{RealmComponents: mockComponents}

	component := &keycloakApi.KeycloakRealmComponent{
		ObjectMeta: metav1.ObjectMeta{Name: testComponentName},
		Status:     keycloakApi.KeycloakComponentStatus{ID: testComponentID},
	}

	mockComponents.EXPECT().
		DeleteComponent(context.Background(), testRealmName, testComponentID).
		Return(nil, errors.New("delete error"))

	h := NewRemoveComponent(kClient)
	err := h.Serve(context.Background(), component, testRealmName)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to delete realm component")
}
