package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestCleanupResource_Serve(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	tests := []struct {
		name      string
		user      *keycloakApi.KeycloakRealmUser
		k8sClient func(t *testing.T) client.Client
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "KeepResource is true - should skip deletion",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:     "testuser",
					KeepResource: true,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "KeepResource is false - should delete resource",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:     "testuser",
					KeepResource: false,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				user := &keycloakApi.KeycloakRealmUser{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-user",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmUserSpec{
						Username:     "testuser",
						KeepResource: false,
					},
				}
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(user).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "KeepResource is false - resource not found should not error",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:     "testuser",
					KeepResource: false,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "KeepResource is false - delete fails with error",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:     "testuser",
					KeepResource: false,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return &fakeClientWithDeleteError{
					Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
				}
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewCleanupResource(tt.k8sClient(t))
			err := handler.Serve(
				context.Background(),
				tt.user,
				mocks.NewMockClient(t),
				&gocloak.RealmRepresentation{
					Realm: gocloak.StringP("test-realm"),
				},
				&UserContext{},
			)

			tt.wantErr(t, err)
		})
	}
}

// fakeClientWithDeleteError is a fake client that returns an error on Delete.
type fakeClientWithDeleteError struct {
	client.Client
}

func (f *fakeClientWithDeleteError) Delete(_ context.Context, _ client.Object, _ ...client.DeleteOption) error {
	return errors.New("delete error")
}
