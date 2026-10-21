package chain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/epam/edp-keycloak-operator/api/common"
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

	ownerDeleting := func() *keycloakApi.KeycloakRealmComponent {
		owner := ownerComponent(testNamespace, "owner", testComponentID)
		owner.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		owner.Finalizers = []string{common.FinalizerName}

		return owner
	}

	foundByNameOwned := func(components *mocks.MockRealmComponentsClient, realms *mocks.MockRealmClient) {
		realmFound(realms)
		components.EXPECT().GetComponents(context.Background(), testRealmName, byName(testRealmID)).
			Return([]keycloakapi.ComponentRepresentation{{
				Id:           ptr.To(testComponentID),
				Name:         ptr.To(testComponentName),
				ProviderType: ptr.To(testProviderType),
			}}, nil, nil)
	}

	refusal := "is managed by KeycloakRealmComponent " + testNamespace + "/owner"

	tests := []struct {
		name        string
		statusID    string
		objects     []client.Object
		listErr     bool
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
					Return([]keycloakapi.ComponentRepresentation{{
						Id:           ptr.To(testComponentID),
						Name:         ptr.To(testComponentName),
						ProviderType: ptr.To(testProviderType),
					}}, nil, nil)
				components.EXPECT().DeleteComponent(context.Background(), testRealmName, testComponentID).Return(nil, nil)
			},
		},
		{
			name: "no status.id: same name of another providerType is not deleted",
			setup: func(components *mocks.MockRealmComponentsClient, realms *mocks.MockRealmClient) {
				realmFound(realms)
				components.EXPECT().GetComponents(context.Background(), testRealmName, byName(testRealmID)).
					Return([]keycloakapi.ComponentRepresentation{{
						Id:           ptr.To(otherTypeID),
						Name:         ptr.To(testComponentName),
						ProviderType: ptr.To(otherProviderType),
					}}, nil, nil)
			},
		},
		{
			name: "no status.id: deletes the component of its own providerType",
			setup: func(components *mocks.MockRealmComponentsClient, realms *mocks.MockRealmClient) {
				realmFound(realms)
				components.EXPECT().GetComponents(context.Background(), testRealmName, byName(testRealmID)).
					Return([]keycloakapi.ComponentRepresentation{
						{
							Id:           ptr.To(otherTypeID),
							Name:         ptr.To(testComponentName),
							ProviderType: ptr.To(otherProviderType),
						},
						{
							Id:           ptr.To(testComponentID),
							Name:         ptr.To(testComponentName),
							ProviderType: ptr.To(testProviderType),
						},
					}, nil, nil)
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
		{
			name:     "status.id recorded by another CR is not deleted",
			statusID: testComponentID,
			objects:  []client.Object{ownerComponent(testNamespace, "owner", testComponentID)},
			wantErr:  refusal,
		},
		{
			name:    "no status.id: component found by name and recorded by another CR is skipped without delete",
			objects: []client.Object{ownerComponent(testNamespace, "owner", testComponentID)},
			setup:   foundByNameOwned,
		},
		{
			name:        "preserve annotation skips the ownership check with status.id",
			statusID:    testComponentID,
			objects:     []client.Object{ownerComponent(testNamespace, "owner", testComponentID)},
			annotations: map[string]string{objectmeta.PreserveResourcesOnDeletionAnnotation: "true"},
		},
		{
			name:        "preserve annotation skips the lookup and ownership check without status.id",
			objects:     []client.Object{ownerComponent(testNamespace, "owner", testComponentID)},
			annotations: map[string]string{objectmeta.PreserveResourcesOnDeletionAnnotation: "true"},
		},
		{
			name:     "owner being deleted still blocks the delete",
			statusID: testComponentID,
			objects:  []client.Object{ownerDeleting()},
			wantErr:  refusal,
		},
		{
			name:     "ownership list error blocks the delete",
			statusID: testComponentID,
			listErr:  true,
			wantErr:  "unable to check realm component ownership for deletion",
		},
		{
			name:     "same-ID candidate of an unresolvable realm blocks the delete and is named",
			statusID: testComponentID,
			objects: []client.Object{func() *keycloakApi.KeycloakRealmComponent {
				owner := ownerComponent("other-ns", "owner", testComponentID)
				owner.Spec.RealmRef = common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: "other-realm"}

				return owner
			}()},
			wantErr: "KeycloakRealmComponent other-ns/owner",
		},
		{
			name:     "status.id with another CR recording a different ID is deleted",
			statusID: testComponentID,
			objects:  []client.Object{ownerComponent(testNamespace, "other", "another-id")},
			setup: func(components *mocks.MockRealmComponentsClient, _ *mocks.MockRealmClient) {
				components.EXPECT().DeleteComponent(context.Background(), testRealmName, testComponentID).Return(nil, nil)
			},
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
				ObjectMeta: metav1.ObjectMeta{Name: testComponentName, Namespace: testNamespace, Annotations: tt.annotations},
				Spec: keycloakApi.KeycloakComponentSpec{
					Name:         testComponentName,
					ProviderType: testProviderType,
					RealmRef:     testRealmRef(),
					ParentRef:    tt.parentRef,
				},
				Status: keycloakApi.KeycloakComponentStatus{ID: tt.statusID},
			}

			kClient := &keycloakapi.KeycloakClient{RealmComponents: components, Realms: realms}

			k8sClient := newFakeK8sClient(t, tt.objects...)
			if tt.listErr {
				k8sClient = fake.NewClientBuilder().
					WithScheme(newScheme(t)).
					WithInterceptorFuncs(interceptor.Funcs{
						List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
							return errors.New("list failed")
						},
					}).
					Build()
			}

			err := NewRemoveComponent(k8sClient, kClient).Serve(context.Background(), component, testRealmName)

			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.ErrorContains(t, err, tt.wantErr)

			if tt.wantErr == refusal {
				require.ErrorContains(t, err, "id "+testComponentID)
				require.ErrorContains(t, err, objectmeta.PreserveResourcesOnDeletionAnnotation+"=true on KeycloakRealmComponent "+
					testNamespace+"/"+testComponentName)
			}
		})
	}
}
