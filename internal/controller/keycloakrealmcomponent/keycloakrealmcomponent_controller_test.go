package keycloakrealmcomponent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const unitNamespace = "unit-ns"

func unitScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))

	return s
}

func unitComponent(name string, parentRef *keycloakApi.ParentComponent, owners ...metav1.OwnerReference) *keycloakApi.KeycloakRealmComponent {
	return &keycloakApi.KeycloakRealmComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       unitNamespace,
			UID:             types.UID(name + "-uid"),
			OwnerReferences: owners,
		},
		Spec: keycloakApi.KeycloakComponentSpec{
			Name:      name,
			ParentRef: parentRef,
		},
	}
}

func componentOwnerRef(name string) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         keycloakApi.GroupVersion.String(),
		Kind:               keycloakApi.KeycloakRealmComponentKind,
		Name:               name,
		UID:                types.UID(name + "-uid"),
		BlockOwnerDeletion: ptr.To(true),
		Controller:         ptr.To(false),
	}
}

func realmOwnerRef() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: keycloakApi.GroupVersion.String(),
		Kind:       keycloakApi.KeycloakRealmKind,
		Name:       "realm",
		UID:        "realm-uid",
		Controller: ptr.To(true),
	}
}

func TestSyncParentOwnerReference(t *testing.T) {
	parentA := unitComponent("parent-a", nil)
	parentB := unitComponent("parent-b", nil)

	tests := []struct {
		name        string
		owners      []metav1.OwnerReference
		parent      *keycloakApi.KeycloakRealmComponent
		want        []metav1.OwnerReference
		wantPatches int
	}{
		{
			name:        "add parent reference",
			owners:      []metav1.OwnerReference{realmOwnerRef()},
			parent:      parentA,
			want:        []metav1.OwnerReference{realmOwnerRef(), componentOwnerRef("parent-a")},
			wantPatches: 1,
		},
		{
			name:        "replace reference on re-parent",
			owners:      []metav1.OwnerReference{componentOwnerRef("parent-a"), realmOwnerRef()},
			parent:      parentB,
			want:        []metav1.OwnerReference{realmOwnerRef(), componentOwnerRef("parent-b")},
			wantPatches: 1,
		},
		{
			name:        "remove reference when the parent is the realm",
			owners:      []metav1.OwnerReference{componentOwnerRef("parent-a"), realmOwnerRef()},
			parent:      nil,
			want:        []metav1.OwnerReference{realmOwnerRef()},
			wantPatches: 1,
		},
		{
			name:        "drop duplicate parent references",
			owners:      []metav1.OwnerReference{componentOwnerRef("parent-a"), componentOwnerRef("parent-a")},
			parent:      parentA,
			want:        []metav1.OwnerReference{componentOwnerRef("parent-a")},
			wantPatches: 1,
		},
		{
			name:        "unchanged reference keeps its position without a patch",
			owners:      []metav1.OwnerReference{componentOwnerRef("parent-a"), realmOwnerRef()},
			parent:      parentA,
			want:        []metav1.OwnerReference{componentOwnerRef("parent-a"), realmOwnerRef()},
			wantPatches: 0,
		},
		{
			name:        "no references and no parent",
			parent:      nil,
			want:        nil,
			wantPatches: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := unitScheme(t)
			patches := 0

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(unitComponent("child", nil, tt.owners...)).
				WithInterceptorFuncs(interceptor.Funcs{
					Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
						patches++

						return c.Patch(ctx, obj, patch, opts...)
					},
				}).
				Build()

			r := NewRealmComponentReconciler(k8sClient, scheme, nil, nil)

			child := &keycloakApi.KeycloakRealmComponent{}
			require.NoError(t, k8sClient.Get(context.Background(), types.NamespacedName{Namespace: unitNamespace, Name: "child"}, child))

			require.NoError(t, r.syncParentOwnerReference(context.Background(), child, tt.parent))
			assert.Equal(t, tt.wantPatches, patches)

			stored := &keycloakApi.KeycloakRealmComponent{}
			require.NoError(t, k8sClient.Get(context.Background(), types.NamespacedName{Namespace: unitNamespace, Name: "child"}, stored))
			assert.Equal(t, tt.want, stored.GetOwnerReferences())
			assert.Equal(t, tt.want, child.GetOwnerReferences())
		})
	}
}

func TestIndexParentComponent(t *testing.T) {
	assert.Nil(t, indexParentComponent(unitComponent("top", nil)))
	assert.Nil(t, indexParentComponent(unitComponent("realm-parent", &keycloakApi.ParentComponent{
		Kind: keycloakApi.KeycloakRealmKind, Name: "realm",
	})))
	assert.Equal(t, []string{"parent"}, indexParentComponent(unitComponent("child", &keycloakApi.ParentComponent{
		Kind: keycloakApi.KeycloakRealmComponentKind, Name: "parent",
	})))
	assert.Nil(t, indexParentComponent(&keycloakApi.KeycloakRealm{}))
}

func TestChildComponentRequests(t *testing.T) {
	componentParent := func(name string) *keycloakApi.ParentComponent {
		return &keycloakApi.ParentComponent{Kind: keycloakApi.KeycloakRealmComponentKind, Name: name}
	}

	otherNamespaceChild := unitComponent("child-other-ns", componentParent("parent"))
	otherNamespaceChild.Namespace = "other-ns"

	scheme := unitScheme(t)
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithIndex(&keycloakApi.KeycloakRealmComponent{}, parentComponentIndexField, indexParentComponent).
		WithObjects(
			unitComponent("parent", nil),
			unitComponent("child-1", componentParent("parent")),
			unitComponent("child-2", componentParent("parent")),
			unitComponent("child-of-other", componentParent("other-parent")),
			unitComponent("realm-child", &keycloakApi.ParentComponent{Kind: keycloakApi.KeycloakRealmKind, Name: "parent"}),
			otherNamespaceChild,
		).
		Build()

	r := NewRealmComponentReconciler(k8sClient, scheme, nil, nil)

	got := r.childComponentRequests(context.Background(), unitComponent("parent", nil))
	assert.ElementsMatch(t, []reconcile.Request{
		{NamespacedName: types.NamespacedName{Namespace: unitNamespace, Name: "child-1"}},
		{NamespacedName: types.NamespacedName{Namespace: unitNamespace, Name: "child-2"}},
	}, got)

	assert.Empty(t, r.childComponentRequests(context.Background(), unitComponent("childless", nil)))
}

func TestParentIDChangedPredicate(t *testing.T) {
	p := parentIDChangedPredicate()

	withID := func(id string) *keycloakApi.KeycloakRealmComponent {
		c := unitComponent("parent", nil)
		c.Status.ID = id

		return c
	}

	assert.True(t, p.Update(event.UpdateEvent{ObjectOld: withID(""), ObjectNew: withID("id-1")}), "parent became ready")
	assert.True(t, p.Update(event.UpdateEvent{ObjectOld: withID("id-1"), ObjectNew: withID("id-2")}), "parent recreated")
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: withID("id-1"), ObjectNew: withID("id-1")}), "ID unchanged")
	assert.False(t, p.Create(event.CreateEvent{Object: withID("id-1")}))
	assert.False(t, p.Delete(event.DeleteEvent{Object: withID("id-1")}))
	assert.False(t, p.Generic(event.GenericEvent{Object: withID("id-1")}))
}

func TestHandleReconciliation_ParentErrors(t *testing.T) {
	componentParent := func(name string) *keycloakApi.ParentComponent {
		return &keycloakApi.ParentComponent{Kind: keycloakApi.KeycloakRealmComponentKind, Name: name}
	}

	tests := []struct {
		name        string
		parentRef   *keycloakApi.ParentComponent
		wantStatus  string
		wantRequeue bool
	}{
		{
			name:        "missing parent requeues without error",
			parentRef:   componentParent("missing-parent"),
			wantStatus:  `parent component "missing-parent" not found`,
			wantRequeue: true,
		},
		{
			name:       "self reference fails the reconcile",
			parentRef:  componentParent("child"),
			wantStatus: `parentRef "child" must not reference the component itself`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := unitScheme(t)
			child := unitComponent("child", tt.parentRef)
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(child).
				WithStatusSubresource(child).
				Build()
			r := NewRealmComponentReconciler(k8sClient, scheme, nil, nil)

			key := types.NamespacedName{Namespace: unitNamespace, Name: "child"}
			instance := &keycloakApi.KeycloakRealmComponent{}
			require.NoError(t, k8sClient.Get(context.Background(), key, instance))

			// No Keycloak calls happen before the parent CR is validated.
			res, err := r.handleReconciliation(context.Background(), instance, &keycloakapi.KeycloakClient{}, "realm")

			if tt.wantRequeue {
				require.NoError(t, err)
				assert.Equal(t, parentNotReadyRequeueTime, res.RequeueAfter)
			} else {
				require.ErrorContains(t, err, tt.wantStatus)
				assert.Zero(t, res.RequeueAfter)
			}

			stored := &keycloakApi.KeycloakRealmComponent{}
			require.NoError(t, k8sClient.Get(context.Background(), key, stored))
			assert.Contains(t, stored.Status.Value, tt.wantStatus)
			assert.Contains(t, stored.Finalizers, common.FinalizerName)
		})
	}
}
