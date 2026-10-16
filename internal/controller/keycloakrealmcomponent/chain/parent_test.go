package chain

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

const (
	testParentCRName = "corporate-ldap"
	testParentID     = "parent-comp-id-789"
	testKeycloakURL  = "http://kc.example"
)

// parentComponentCR returns a parent KeycloakRealmComponent CR in testNamespace, in the same
// realm as baseComponent(), whose metadata.name (referenced by parentRef.name) differs from its
// Keycloak name (spec.name). id is its status.ID; empty means not reconciled yet.
func parentComponentCR(name, keycloakName, id string) *keycloakApi.KeycloakRealmComponent {
	return &keycloakApi.KeycloakRealmComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
			UID:       "parent-uid",
		},
		Spec: keycloakApi.KeycloakComponentSpec{
			Name:         keycloakName,
			ProviderID:   testProviderID,
			ProviderType: testProviderType,
			RealmRef:     testRealmRef(),
		},
		Status: keycloakApi.KeycloakComponentStatus{
			ID: id,
		},
	}
}

// realmStack returns a Keycloak and a KeycloakRealm CR named realmCR in testNamespace.
func realmStack(realmCR, realmName string) []client.Object {
	return []client.Object{
		&keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{Name: "keycloak", Namespace: testNamespace},
			Spec:       keycloakApi.KeycloakSpec{Url: testKeycloakURL},
		},
		&keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: realmCR, Namespace: testNamespace},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName:   realmName,
				KeycloakRef: common.KeycloakRef{Kind: keycloakApi.KeycloakKind, Name: "keycloak"},
			},
		},
	}
}

func componentChild(parentName string) *keycloakApi.KeycloakRealmComponent {
	component := baseComponent()
	component.Spec.ParentRef = &keycloakApi.ParentComponent{
		Kind: keycloakApi.KeycloakRealmComponentKind,
		Name: parentName,
	}

	return component
}

func TestResolveParent_RealmParent(t *testing.T) {
	tests := []struct {
		name      string
		parentRef *keycloakApi.ParentComponent
	}{
		{name: "no parentRef"},
		{name: "KeycloakRealm parentRef matching realmRef", parentRef: &keycloakApi.ParentComponent{
			Kind: keycloakApi.KeycloakRealmKind, Name: testRealmRef().Name,
		}},
		{name: "empty kind defaults to KeycloakRealm", parentRef: &keycloakApi.ParentComponent{
			Name: testRealmRef().Name,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRealms := mocks.NewMockRealmClient(t)
			kClient := &keycloakapi.KeycloakClient{Realms: mockRealms}

			component := baseComponent()
			component.Spec.ParentRef = tt.parentRef

			mockRealms.EXPECT().
				GetRealm(context.Background(), testRealmName).
				Return(&keycloakapi.RealmRepresentation{Id: ptr.To(testRealmID)}, nil, nil)

			got, err := ResolveParent(context.Background(), fake.NewClientBuilder().WithScheme(newScheme(t)).Build(),
				kClient, component, testRealmName)
			require.NoError(t, err)
			assert.Equal(t, ResolvedParent{ID: testRealmID}, got)
		})
	}
}

func TestResolveParent_RealmParentErrors(t *testing.T) {
	tests := []struct {
		name    string
		realm   *keycloakapi.RealmRepresentation
		err     error
		wantErr string
	}{
		{name: "GetRealm error", err: errors.New("boom"), wantErr: `unable to get realm "test-realm" from Keycloak`},
		{name: "empty realm ID", realm: &keycloakapi.RealmRepresentation{}, wantErr: `realm "test-realm" has no ID`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRealms := mocks.NewMockRealmClient(t)
			kClient := &keycloakapi.KeycloakClient{Realms: mockRealms}

			mockRealms.EXPECT().GetRealm(context.Background(), testRealmName).Return(tt.realm, nil, tt.err)

			_, err := ResolveParent(context.Background(), fake.NewClientBuilder().WithScheme(newScheme(t)).Build(),
				kClient, baseComponent(), testRealmName)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestResolveParent_RealmParentMismatch(t *testing.T) {
	// no expectations: Keycloak must not be called for a foreign realm parent
	kClient := &keycloakapi.KeycloakClient{Realms: mocks.NewMockRealmClient(t)}

	component := baseComponent()
	component.Spec.ParentRef = &keycloakApi.ParentComponent{Kind: keycloakApi.KeycloakRealmKind, Name: "other-realm-cr"}

	_, err := ResolveParent(context.Background(), fake.NewClientBuilder().WithScheme(newScheme(t)).Build(),
		kClient, component, testRealmName)
	require.ErrorContains(t, err, `parentRef "other-realm-cr" does not match spec.realmRef "test-realm-cr"`)
}

func TestResolveParent_ComponentParent(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	parent := parentComponentCR(testParentCRName, "Corporate LDAP", testParentID)
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(parent).Build()

	mockComponents.EXPECT().
		GetComponent(context.Background(), testRealmName, testParentID).
		Return(&keycloakapi.ComponentRepresentation{Id: ptr.To(testParentID)}, nil, nil)

	got, err := ResolveParent(context.Background(), fakeClient, kClient, componentChild(testParentCRName), testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testParentID, got.ID)
	require.NotNil(t, got.CR)
	assert.Equal(t, testParentCRName, got.CR.Name)
}

func TestResolveParent_ComponentParentResolvedRealmMatch(t *testing.T) {
	mockComponents := mocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	// The parent references the same Keycloak realm through a ClusterKeycloakRealm.
	parent := parentComponentCR(testParentCRName, "Corporate LDAP", testParentID)
	parent.Spec.RealmRef = common.RealmRef{Kind: keycloakAlpha.ClusterKeycloakRealmKind, Name: "cluster-realm"}

	objs := append(realmStack(testRealmRef().Name, testRealmName),
		parent,
		&keycloakAlpha.ClusterKeycloak{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster-keycloak"},
			Spec:       keycloakAlpha.ClusterKeycloakSpec{Url: testKeycloakURL + "/"},
		},
		&keycloakAlpha.ClusterKeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster-realm"},
			Spec:       keycloakAlpha.ClusterKeycloakRealmSpec{ClusterKeycloakRef: "cluster-keycloak", RealmName: testRealmName},
		},
	)
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(objs...).Build()

	mockComponents.EXPECT().
		GetComponent(context.Background(), testRealmName, testParentID).
		Return(&keycloakapi.ComponentRepresentation{Id: ptr.To(testParentID)}, nil, nil)

	got, err := ResolveParent(context.Background(), fakeClient, kClient, componentChild(testParentCRName), testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testParentID, got.ID)
}

func TestResolveParent_ComponentParentErrors(t *testing.T) {
	deleting := parentComponentCR("deleting-parent", "Deleting Parent", testParentID)
	deleting.DeletionTimestamp = ptr.To(metav1.Now())
	deleting.Finalizers = []string{common.FinalizerName}

	otherRealm := parentComponentCR("other-realm-parent", "Other Realm Parent", testParentID)
	otherRealm.Spec.RealmRef.Name = "other-realm-cr"

	unresolvable := parentComponentCR("unresolvable-parent", "Unresolvable Parent", testParentID)
	unresolvable.Spec.RealmRef.Name = "missing-realm-cr"

	tests := []struct {
		name       string
		objs       []client.Object
		parentName string
		wantErr    []string
		notReady   bool
	}{
		{
			name:       "missing parent CR",
			parentName: "missing-parent-cr",
			wantErr:    []string{`parent component "missing-parent-cr" not found`},
			notReady:   true,
		},
		{
			name:       "self reference",
			parentName: testComponentName,
			wantErr:    []string{`parentRef "test-component" must not reference the component itself`},
		},
		{
			name:       "parent being deleted",
			objs:       []client.Object{deleting},
			parentName: "deleting-parent",
			wantErr:    []string{`parent component "deleting-parent" is being deleted`},
		},
		{
			name: "parent in another realm",
			objs: append(append(realmStack(testRealmRef().Name, testRealmName),
				realmStack("other-realm-cr", "other-realm")[1]), otherRealm),
			parentName: "other-realm-parent",
			wantErr:    []string{`parent component "other-realm-parent" belongs to a different realm`},
		},
		{
			name:       "parent realm cannot be resolved",
			objs:       append(realmStack(testRealmRef().Name, testRealmName), unresolvable),
			parentName: "unresolvable-parent",
			wantErr:    []string{`unable to compare realms of parent component "unresolvable-parent"`},
		},
		{
			name:       "parent not ready",
			objs:       []client.Object{parentComponentCR("pending-parent", "Pending Parent", "")},
			parentName: "pending-parent",
			wantErr:    []string{`parent component "pending-parent" has no ID in status`},
			notReady:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// no expectations: Keycloak must not be called before the parent CR is validated
			kClient := &keycloakapi.KeycloakClient{RealmComponents: mocks.NewMockRealmComponentsClient(t)}
			fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(tt.objs...).Build()

			_, err := ResolveParent(context.Background(), fakeClient, kClient, componentChild(tt.parentName), testRealmName)
			for _, want := range tt.wantErr {
				require.ErrorContains(t, err, want)
			}

			assert.Equal(t, tt.notReady, errors.Is(err, ErrParentNotReady))
		})
	}
}

func TestResolveParent_ComponentParentKeycloakErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantErr  string
		notReady bool
	}{
		{
			name:     "stale parent ID",
			err:      &keycloakapi.ApiError{Code: http.StatusNotFound, Message: "Not Found"},
			wantErr:  `parent not ready: parent component "corporate-ldap" (ID parent-comp-id-789) not found in Keycloak`,
			notReady: true,
		},
		{
			name:    "lookup error",
			err:     errors.New("boom"),
			wantErr: `unable to get parent component "corporate-ldap" from Keycloak: boom`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockComponents := mocks.NewMockRealmComponentsClient(t)
			kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}
			fakeClient := fake.NewClientBuilder().
				WithScheme(newScheme(t)).
				WithObjects(parentComponentCR(testParentCRName, "Corporate LDAP", testParentID)).
				Build()

			mockComponents.EXPECT().
				GetComponent(context.Background(), testRealmName, testParentID).
				Return(nil, nil, tt.err)

			_, err := ResolveParent(context.Background(), fakeClient, kClient, componentChild(testParentCRName), testRealmName)
			require.EqualError(t, err, tt.wantErr)
			assert.Equal(t, tt.notReady, errors.Is(err, ErrParentNotReady))
		})
	}
}

func TestResolveParent_UnsupportedKind(t *testing.T) {
	component := baseComponent()
	component.Spec.ParentRef = &keycloakApi.ParentComponent{Kind: "UnsupportedKind", Name: "something"}

	_, err := ResolveParent(context.Background(), fake.NewClientBuilder().WithScheme(newScheme(t)).Build(),
		&keycloakapi.KeycloakClient{}, component, testRealmName)
	assert.ErrorContains(t, err, "parent kind UnsupportedKind is not supported")
}

func TestResolveParent_ParentCycle(t *testing.T) {
	// A -> B -> A: both parents are ready and exist, only the chain walk can catch it.
	a := parentComponentCR(testComponentName, "A", testComponentID)
	a.Spec.ParentRef = &keycloakApi.ParentComponent{Kind: keycloakApi.KeycloakRealmComponentKind, Name: "b"}
	b := parentComponentCR("b", "B", testParentID)
	b.Spec.ParentRef = &keycloakApi.ParentComponent{Kind: keycloakApi.KeycloakRealmComponentKind, Name: testComponentName}

	objs := append(realmStack(testRealmRef().Name, testRealmName), a, b)
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(objs...).Build()
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mocks.NewMockRealmComponentsClient(t)}

	_, err := ResolveParent(context.Background(), fakeClient, kClient, componentChild("b"), testRealmName)
	require.ErrorContains(t, err, `parentRef "b" forms a cycle through "test-component"`)
}

func TestResolveParent_GrandparentChainAccepted(t *testing.T) {
	// child -> parent -> grandparent -> realm is a valid chain; a missing ancestor ends the walk.
	grandparent := parentComponentCR("grandparent", "Grandparent", "grand-id")
	parent := parentComponentCR(testParentCRName, "Parent", testParentID)
	parent.Spec.ParentRef = &keycloakApi.ParentComponent{Kind: keycloakApi.KeycloakRealmComponentKind, Name: "grandparent"}
	orphanParent := parentComponentCR("orphan-parent", "Orphan Parent", testParentID)
	orphanParent.Spec.ParentRef = &keycloakApi.ParentComponent{Kind: keycloakApi.KeycloakRealmComponentKind, Name: "missing"}

	objs := append(realmStack(testRealmRef().Name, testRealmName), grandparent, parent, orphanParent)
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(objs...).Build()

	for _, parentName := range []string{testParentCRName, "orphan-parent"} {
		t.Run(parentName, func(t *testing.T) {
			mockComponents := mocks.NewMockRealmComponentsClient(t)
			mockComponents.EXPECT().
				GetComponent(context.Background(), testRealmName, testParentID).
				Return(&keycloakapi.ComponentRepresentation{Id: ptr.To(testParentID)}, nil, nil)

			kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

			got, err := ResolveParent(context.Background(), fakeClient, kClient, componentChild(parentName), testRealmName)
			require.NoError(t, err)
			assert.Equal(t, testParentID, got.ID)
		})
	}
}

func TestResolveParent_ParentChainTooDeep(t *testing.T) {
	objs := realmStack(testRealmRef().Name, testRealmName)

	for i := range maxParentDepth + 1 {
		level := parentComponentCR(fmt.Sprintf("level-%d", i), fmt.Sprintf("Level %d", i), testParentID)
		level.Spec.ParentRef = &keycloakApi.ParentComponent{
			Kind: keycloakApi.KeycloakRealmComponentKind,
			Name: fmt.Sprintf("level-%d", i+1),
		}
		objs = append(objs, level)
	}

	fakeClient := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(objs...).Build()
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mocks.NewMockRealmComponentsClient(t)}

	_, err := ResolveParent(context.Background(), fakeClient, kClient, componentChild("level-0"), testRealmName)
	require.ErrorContains(t, err, fmt.Sprintf("chain exceeds %d levels", maxParentDepth))
}
