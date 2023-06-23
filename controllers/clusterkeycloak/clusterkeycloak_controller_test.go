package clusterkeycloak

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestNewReconcile(t *testing.T) {
	kc := NewReconcile(nil, nil, mock.NewLogr(), &helper.Mock{})
	if kc.scheme != nil {
		t.Fatal("something went wrong")
	}
}

func TestReconcileClusterKeycloak_ReconcilePass(t *testing.T) {
	cr := &keycloakApi.ClusterKeycloak{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterKeycloak",
			APIVersion: "apps/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewKeycloak",
			Namespace: "namespace",
		},
		Spec: keycloakApi.ClusterKeycloakSpec{
			Url:    "https://some",
			Secret: "keycloak-secret",
		},
	}

	objs := []runtime.Object{
		cr,
	}
	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, cr)

	cl := fake.NewClientBuilder().WithRuntimeObjects(objs...).Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewKeycloak",
			Namespace: "namespace",
		},
	}

	logger := mock.NewLogr()

	r := ClusterKeycloakReconciler{
		client: cl,
		scheme: s,
		log:    logger,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)

	persisted := &keycloakApi.ClusterKeycloak{}
	err = cl.Get(context.TODO(), req.NamespacedName, persisted)
	assert.Nil(t, err)
	assert.False(t, persisted.Status.Connected)
}

func TestReconcileClusterKeycloak_ReconcilePassWithNoFound(t *testing.T) {

	objs := []runtime.Object{}
	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion)

	cl := fake.NewClientBuilder().WithRuntimeObjects(objs...).Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewKeycloak",
			Namespace: "namespace",
		},
	}

	logger := mock.NewLogr()

	r := ClusterKeycloakReconciler{
		client: cl,
		scheme: s,
		log:    logger,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}
