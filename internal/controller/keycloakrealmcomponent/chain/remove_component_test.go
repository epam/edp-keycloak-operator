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

func TestRemoveComponent_Serve(t *testing.T) {
	realmFound := func(realms *mocks.MockRealmClient) {
		realms.EXPECT().GetRealm(context.Background(), testRealmName).
			Return(&keycloakapi.RealmRepresentation{Id: ptr.To(testRealmID)}, nil, nil)
	}

	tests := []struct {
		name        string
		statusID    string
		parentRef   *keycloakApi.ParentComponent
		annotations map[string]string
		setup       func(components *mocks.MockRealmComponentsClient, realms *mocks.MockRealmClient)
		wantErr     string
	}{
		{
			name:     "deletes by status.id",
			statusID: testComponentID,
			setup: func(components *mocks.MockRealmComponentsClient, _ *mocks.MockRealmClient) {
				components.EXPECT().DeleteComponent(context.Background(), testRealmName, testComponentID).Return(nil, nil)
			},
		},
		{
			name:     "status.id already gone in Keycloak",
			statusID: testComponentID,
			setup: func(components *mocks.MockRealmComponentsClient, _ *mocks.MockRealmClient) {
				components.EXPECT().DeleteComponent(context.Background(), testRealmName, testComponentID).
					Return(nil, &keycloakapi.ApiError{Code: 404, Message: "Not Found"})
			},
		},
		{
			name:     "delete error",
			statusID: testComponentID,
			setup: func(components *mocks.MockRealmComponentsClient, _ *mocks.MockRealmClient) {
				components.EXPECT().DeleteComponent(context.Background(), testRealmName, testComponentID).
					Return(nil, errors.New("delete error"))
			},
			wantErr: "failed to delete realm component",
		},
		{
			name:        "preserve annotation skips Keycloak",
			statusID:    testComponentID,
			annotations: map[string]string{objectmeta.PreserveResourcesOnDeletionAnnotation: "true"},
		},
		{
			name: "no status.id: found by name under the realm",
			setup: func(components *mocks.MockRealmComponentsClient, realms *mocks.MockRealmClient) {
				realmFound(realms)
				components.EXPECT().GetComponents(context.Background(), testRealmName, byName(testRealmID)).
					Return([]keycloakapi.ComponentRepresentation{{Id: ptr.To(testComponentID), Name: ptr.To(testComponentName)}}, nil, nil)
				components.EXPECT().DeleteComponent(context.Background(), testRealmName, testComponentID).Return(nil, nil)
			},
		},
		{
			name: "no status.id: nothing under the realm",
			setup: func(components *mocks.MockRealmComponentsClient, realms *mocks.MockRealmClient) {
				realmFound(realms)
				components.EXPECT().GetComponents(context.Background(), testRealmName, byName(testRealmID)).Return(nil, nil, nil)
			},
		},
		{
			name: "no status.id: lookup error",
			setup: func(components *mocks.MockRealmComponentsClient, realms *mocks.MockRealmClient) {
				realmFound(realms)
				components.EXPECT().GetComponents(context.Background(), testRealmName, byName(testRealmID)).
					Return(nil, nil, errors.New("lookup error"))
			},
			wantErr: "failed to find component for deletion",
		},
		{
			name: "no status.id: realm lookup error",
			setup: func(_ *mocks.MockRealmComponentsClient, realms *mocks.MockRealmClient) {
				realms.EXPECT().GetRealm(context.Background(), testRealmName).Return(nil, nil, errors.New("realm error"))
			},
			wantErr: "failed to resolve realm for deletion",
		},
		{
			name:      "no status.id: empty parentRef.kind is the realm",
			parentRef: &keycloakApi.ParentComponent{Name: "test-realm-cr"},
			setup: func(components *mocks.MockRealmComponentsClient, realms *mocks.MockRealmClient) {
				realmFound(realms)
				components.EXPECT().GetComponents(context.Background(), testRealmName, byName(testRealmID)).Return(nil, nil, nil)
			},
		},
		{
			name:      "no status.id: component parent skips the name lookup",
			parentRef: &keycloakApi.ParentComponent{Kind: keycloakApi.KeycloakRealmComponentKind, Name: "parent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := mocks.NewMockRealmComponentsClient(t)
			realms := mocks.NewMockRealmClient(t)

			if tt.setup != nil {
				tt.setup(components, realms)
			}

			component := &keycloakApi.KeycloakRealmComponent{
				ObjectMeta: metav1.ObjectMeta{Name: testComponentName, Annotations: tt.annotations},
				Spec:       keycloakApi.KeycloakComponentSpec{Name: testComponentName, ParentRef: tt.parentRef},
				Status:     keycloakApi.KeycloakComponentStatus{ID: tt.statusID},
			}

			kClient := &keycloakapi.KeycloakClient{RealmComponents: components, Realms: realms}
			err := NewRemoveComponent(kClient).Serve(context.Background(), component, testRealmName)

			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
