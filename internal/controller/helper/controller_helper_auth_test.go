package helper

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakClient "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/fakehttp"
)

func TestMakeKeycloakAuthDataFromKeycloak(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name      string
		keycloak  *keycloakApi.Keycloak
		k8sClient func(t *testing.T) client.Client
		want      *KeycloakAuthData
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "successfully create keycloak auth data with caCert secret",
			keycloak: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url:    "https://test.com",
					Secret: "admin-secret",
					CACert: &common.SourceRef{
						SecretKeyRef: &common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-secret",
							},
							Key: "ca.crt",
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithRuntimeObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"ca.crt": []byte("test-ca-cert"),
						},
					}).Build()
			},
			want: &KeycloakAuthData{
				Url:             "https://test.com",
				SecretName:      "admin-secret",
				SecretNamespace: "default",
				KeycloakCRName:  "test",
				CACert:          "test-ca-cert",
			},
			wantErr: require.NoError,
		},
		{
			name: "successfully create keycloak auth data with caCert configmap",
			keycloak: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url:    "https://test.com",
					Secret: "admin-secret",
					CACert: &common.SourceRef{
						ConfigMapKeyRef: &common.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-configmap",
							},
							Key: "ca.crt",
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithRuntimeObjects(
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-configmap",
							Namespace: "default",
						},
						Data: map[string]string{
							"ca.crt": "test-ca-cert",
						},
					}).Build()
			},
			want: &KeycloakAuthData{
				Url:             "https://test.com",
				SecretName:      "admin-secret",
				SecretNamespace: "default",
				KeycloakCRName:  "test",
				CACert:          "test-ca-cert",
			},
			wantErr: require.NoError,
		},
		{
			name: "caCert secret not found",
			keycloak: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					CACert: &common.SourceRef{
						SecretKeyRef: &common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-secret",
							},
							Key: "ca.crt",
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get secret")
			},
		},
		{
			name: "caCert configmap not found",
			keycloak: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					CACert: &common.SourceRef{
						ConfigMapKeyRef: &common.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-configmap",
							},
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get configmap")
			},
		},
		{
			name: "successfully propagate auth spec",
			keycloak: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url: "https://test.com",
					Auth: &common.AuthSpec{
						PasswordGrant: &common.PasswordGrantConfig{
							Username: common.SourceRefOrVal{
								Value: "admin",
							},
							PasswordRef: common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-secret",
								},
								Key: "password",
							},
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			want: &KeycloakAuthData{
				Url:             "https://test.com",
				SecretNamespace: "default",
				KeycloakCRName:  "test",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							Value: "admin",
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-secret",
							},
							Key: "password",
						},
					},
				},
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeKeycloakAuthDataFromKeycloak(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.keycloak, tt.k8sClient(t))

			tt.wantErr(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMakeKeycloakAuthDataFromClusterKeycloak(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name      string
		keycloak  *keycloakAlpha.ClusterKeycloak
		k8sClient func(t *testing.T) client.Client
		want      *KeycloakAuthData
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "successfully create keycloak auth data with caCert secret",
			keycloak: &keycloakAlpha.ClusterKeycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: keycloakAlpha.ClusterKeycloakSpec{
					Url:    "https://test.com",
					Secret: "admin-secret",
					CACert: &common.SourceRef{
						SecretKeyRef: &common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-secret",
							},
							Key: "ca.crt",
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithRuntimeObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: "ns-with-secrets",
						},
						Data: map[string][]byte{
							"ca.crt": []byte("test-ca-cert"),
						},
					},
				).Build()
			},
			want: &KeycloakAuthData{
				Url:             "https://test.com",
				SecretName:      "admin-secret",
				SecretNamespace: "ns-with-secrets",
				KeycloakCRName:  "test",
				CACert:          "test-ca-cert",
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeKeycloakAuthDataFromClusterKeycloak(
				ctrl.LoggerInto(context.Background(),
					logr.Discard()),
				tt.keycloak,
				"ns-with-secrets",
				tt.k8sClient(t),
			)

			tt.wantErr(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestHelper_CreateKeycloakClientFromRealm(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddKeycloakAuthResponders().
		BuildAndStart()
	defer mockServer.Close()

	tests := []struct {
		name      string
		realm     *keycloakApi.KeycloakRealm
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, client *keycloakClient.KeycloakClient)
	}{
		{
			name: "successfully create v2 client from realm",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "successfully create v2 client with CA cert",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
						CACert: &common.SourceRef{
							SecretKeyRef: &common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "ca-cert-secret",
								},
								Key: "ca.crt",
							},
						},
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ca-cert-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"ca.crt": []byte("-----BEGIN CERTIFICATE-----\ntest-ca-cert\n-----END CERTIFICATE-----"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "realm not found",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get keycloak")
			},
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "keycloak not connected",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    "https://keycloak.example.com",
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: false,
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrKeycloakIsNotAvailable)
			},
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "credentials secret not found",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    "https://keycloak.example.com",
						Secret: "non-existent-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get credentials")
			},
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "cluster keycloak ref",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakAlpha.ClusterKeycloakKind,
						Name: "test-cluster-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			client, err := helper.CreateKeycloakClientFromRealm(ctx, tt.realm)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, client)
			}
		})
	}
}

func TestHelper_CreateKeycloakClientFromClusterRealm(t *testing.T) {
	t.Parallel()

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddKeycloakAuthResponders().
		BuildAndStart()

	t.Cleanup(func() {
		mockServer.Close()
	})

	tests := []struct {
		name      string
		realm     *keycloakAlpha.ClusterKeycloakRealm
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, client *keycloakClient.KeycloakClient)
	}{
		{
			name: "successfully create v2 client from cluster realm",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "test-cluster-keycloak",
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "cluster keycloak not found",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "non-existent-cluster-keycloak",
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get cluster keycloak")
			},
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "cluster keycloak not connected",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "test-cluster-keycloak",
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    "https://keycloak.example.com",
						Secret: "keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: false,
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrKeycloakIsNotAvailable)
			},
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "credentials secret not found",
			realm: &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-realm",
				},
				Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
					RealmName:          "test",
					ClusterKeycloakRef: "test-cluster-keycloak",
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    "https://keycloak.example.com",
						Secret: "non-existent-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get credentials")
			},
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.Nil(t, client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			client, err := helper.CreateKeycloakClientFromClusterRealm(ctx, tt.realm)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, client)
			}
		})
	}
}

func TestHelper_CreateKeycloakClientFromKeycloak(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddKeycloakAuthResponders().
		BuildAndStart()

	t.Cleanup(func() {
		mockServer.Close()
	})

	tests := []struct {
		name      string
		kc        *keycloakApi.Keycloak
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, cl *keycloakClient.KeycloakClient)
	}{
		{
			name: "successfully create v2 client from keycloak",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url:    mockServer.GetURL(),
					Secret: "keycloak-secret",
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, cl *keycloakClient.KeycloakClient) {
				require.NotNil(t, cl)
			},
		},
		{
			name: "successfully create v2 client with auth passwordGrant",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url: mockServer.GetURL(),
					Auth: &common.AuthSpec{
						PasswordGrant: &common.PasswordGrantConfig{
							Username: common.SourceRefOrVal{
								Value: "admin",
							},
							PasswordRef: common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "keycloak-secret",
								},
								Key: "password",
							},
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, cl *keycloakClient.KeycloakClient) {
				require.NotNil(t, cl)
			},
		},
		{
			name: "successfully create v2 client with auth clientCredentials",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url: mockServer.GetURL(),
					Auth: &common.AuthSpec{
						ClientCredentials: &common.ClientCredentialsConfig{
							ClientID: common.SourceRefOrVal{
								Value: "my-admin-client",
							},
							ClientSecretRef: common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "client-secret",
								},
								Key: "secret",
							},
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "client-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"secret": []byte("client-secret-value"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, cl *keycloakClient.KeycloakClient) {
				require.NotNil(t, cl)
			},
		},
		{
			name: "auth passwordGrant with missing password secret",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url: mockServer.GetURL(),
					Auth: &common.AuthSpec{
						PasswordGrant: &common.PasswordGrantConfig{
							Username: common.SourceRefOrVal{
								Value: "admin",
							},
							PasswordRef: common.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "missing-secret",
								},
								Key: "password",
							},
						},
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to resolve password")
			},
			checkFunc: func(t *testing.T, cl *keycloakClient.KeycloakClient) {
				require.Nil(t, cl)
			},
		},
		{
			name: "credentials secret not found",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url:    mockServer.GetURL(),
					Secret: "non-existent-secret",
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get credentials")
			},
			checkFunc: func(t *testing.T, cl *keycloakClient.KeycloakClient) {
				require.Nil(t, cl)
			},
		},
		{
			name: "ca cert secret not found returns error",
			kc: &keycloakApi.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakSpec{
					Url:    mockServer.GetURL(),
					Secret: "keycloak-secret",
					CACert: &common.SourceRef{
						SecretKeyRef: &common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "non-existent-ca-secret",
							},
							Key: "ca.crt",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get ca cert")
			},
			checkFunc: func(t *testing.T, cl *keycloakClient.KeycloakClient) {
				require.Nil(t, cl)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			h := &Helper{
				client: k8sClient,
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			cl, err := h.CreateKeycloakClientFromKeycloak(ctx, tt.kc)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, cl)
			}
		})
	}
}

func TestHelper_CreateKeycloakClientFromClusterKeycloak(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddKeycloakAuthResponders().
		BuildAndStart()

	t.Cleanup(func() {
		mockServer.Close()
	})

	tests := []struct {
		name      string
		kc        *keycloakAlpha.ClusterKeycloak
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, cl *keycloakClient.KeycloakClient)
	}{
		{
			name: "successfully create v2 client from cluster keycloak",
			kc: &keycloakAlpha.ClusterKeycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-keycloak",
				},
				Spec: keycloakAlpha.ClusterKeycloakSpec{
					Url:    mockServer.GetURL(),
					Secret: "keycloak-secret",
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, cl *keycloakClient.KeycloakClient) {
				require.NotNil(t, cl)
			},
		},
		{
			name: "credentials secret not found",
			kc: &keycloakAlpha.ClusterKeycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-keycloak",
				},
				Spec: keycloakAlpha.ClusterKeycloakSpec{
					Url:    mockServer.GetURL(),
					Secret: "non-existent-secret",
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get credentials")
			},
			checkFunc: func(t *testing.T, cl *keycloakClient.KeycloakClient) {
				require.Nil(t, cl)
			},
		},
		{
			name: "ca cert secret not found returns error",
			kc: &keycloakAlpha.ClusterKeycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-keycloak",
				},
				Spec: keycloakAlpha.ClusterKeycloakSpec{
					Url:    mockServer.GetURL(),
					Secret: "keycloak-secret",
					CACert: &common.SourceRef{
						SecretKeyRef: &common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "non-existent-ca-secret",
							},
							Key: "ca.crt",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get ca cert")
			},
			checkFunc: func(t *testing.T, cl *keycloakClient.KeycloakClient) {
				require.Nil(t, cl)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			h := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			cl, err := h.CreateKeycloakClientFromClusterKeycloak(ctx, tt.kc)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, cl)
			}
		})
	}
}

func TestHelper_buildV2AuthOptions(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name         string
		authData     *KeycloakAuthData
		objects      []client.Object
		wantClientID string
		wantErr      require.ErrorAssertionFunc
	}{
		{
			name: "success with password grant - returns admin-cli client ID",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							Value: "admin",
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "password-secret",
							},
							Key: "password",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "password-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"password": []byte("secret-password"),
					},
				},
			},
			wantClientID: keycloakClient.DefaultAdminClientID,
			wantErr:      require.NoError,
		},
		{
			name: "success with client credentials - returns custom client ID",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					ClientCredentials: &common.ClientCredentialsConfig{
						ClientID: common.SourceRefOrVal{
							Value: "my-service-client",
						},
						ClientSecretRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "client-secret",
							},
							Key: "secret",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "client-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"secret": []byte("client-secret-value"),
					},
				},
			},
			wantClientID: "my-service-client",
			wantErr:      require.NoError,
		},
		{
			name: "error resolving username in password grant",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							SourceRef: common.SourceRef{
								SecretKeyRef: &common.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "non-existent-secret",
									},
									Key: "username",
								},
							},
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "password-secret",
							},
							Key: "password",
						},
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to resolve username")
			},
		},
		{
			name: "error when neither password grant nor client credentials set",
			authData: &KeycloakAuthData{
				SecretNamespace: "default",
				AuthSpec:        &common.AuthSpec{},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "one of passwordGrant or clientCredentials must be set")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client: k8sClient,
			}

			clientID, _, err := helper.buildV2AuthOptions(context.Background(), tt.authData)

			tt.wantErr(t, err)

			if err == nil {
				require.Equal(t, tt.wantClientID, clientID)
			}
		})
	}
}

func TestHelper_createKeycloakClientFromAuthData_withServiceAccount(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddKeycloakAuthResponders().
		BuildAndStart()
	defer mockServer.Close()

	tests := []struct {
		name      string
		authData  *KeycloakAuthData
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, client *keycloakClient.KeycloakClient)
	}{
		{
			name: "successfully create v2 client with service account admin type",
			authData: &KeycloakAuthData{
				Url:             mockServer.GetURL(),
				SecretName:      "keycloak-secret",
				SecretNamespace: "default",
				AdminType:       keycloakApi.KeycloakAdminTypeServiceAccount,
				KeycloakCRName:  "test-keycloak",
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("service-account-client"),
						"password": []byte("client-secret"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "successfully create v2 client with password grant auth spec",
			authData: &KeycloakAuthData{
				Url:             mockServer.GetURL(),
				SecretNamespace: "default",
				KeycloakCRName:  "test-keycloak",
				AuthSpec: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							Value: "admin",
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "password-secret",
							},
							Key: "password",
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "password-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "error when auth spec is invalid",
			authData: &KeycloakAuthData{
				Url:             mockServer.GetURL(),
				SecretNamespace: "default",
				KeycloakCRName:  "test-keycloak",
				AuthSpec:        &common.AuthSpec{},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "one of passwordGrant or clientCredentials must be set")
			},
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.Nil(t, client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			client, err := helper.createKeycloakClientFromAuthData(context.Background(), tt.authData)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, client)
			}
		})
	}
}

func TestHelper_CreateKeycloakClientFromRealmRef(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	mockServer := fakehttp.NewServerBuilder().
		AddKeycloakAuthResponders().
		BuildAndStart()
	defer mockServer.Close()

	tests := []struct {
		name      string
		object    ObjectWithRealmRef
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, client *keycloakClient.KeycloakClient)
	}{
		{
			name: "successfully create v2 client from KeycloakRealm ref",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "test-realm",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test",
						KeycloakRef: common.KeycloakRef{
							Kind: keycloakApi.KeycloakKind,
							Name: "test-keycloak",
						},
					},
				},
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "successfully create v2 client from ClusterKeycloakRealm ref",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakAlpha.ClusterKeycloakRealmKind,
						Name: "test-cluster-realm",
					},
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-realm",
					},
					Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
						RealmName:          "test",
						ClusterKeycloakRef: "test-cluster-keycloak",
					},
				},
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    mockServer.GetURL(),
						Secret: "keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"username": []byte("admin"),
						"password": []byte("admin123"),
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.NotNil(t, client)
			},
		},
		{
			name: "error when KeycloakRealm not found",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "non-existent-realm",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get realm")
			},
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.Nil(t, client)
			},
		},
		{
			name: "error with unknown realm kind",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: "UnknownKind",
						Name: "test-realm",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unknown realm kind")
			},
			checkFunc: func(t *testing.T, client *keycloakClient.KeycloakClient) {
				require.Nil(t, client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			client, err := helper.CreateKeycloakClientFromRealmRef(ctx, tt.object)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, client)
			}
		})
	}
}

func TestHelper_getKeycloakAuthDataFromRealmRef(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name      string
		object    ObjectWithRealmRef
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, authData *KeycloakAuthData)
	}{
		{
			name: "successfully get auth data from KeycloakRealm ref",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "test-realm",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test",
						KeycloakRef: common.KeycloakRef{
							Kind: keycloakApi.KeycloakKind,
							Name: "test-keycloak",
						},
					},
				},
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    "https://keycloak.example.com",
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: true,
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Equal(t, "https://keycloak.example.com", authData.Url)
				require.Equal(t, "keycloak-secret", authData.SecretName)
				require.Equal(t, "default", authData.SecretNamespace)
			},
		},
		{
			name: "successfully get auth data from ClusterKeycloakRealm ref",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakAlpha.ClusterKeycloakRealmKind,
						Name: "test-cluster-realm",
					},
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-realm",
					},
					Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
						RealmName:          "test",
						ClusterKeycloakRef: "test-cluster-keycloak",
					},
				},
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    "https://cluster-keycloak.example.com",
						Secret: "cluster-keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Equal(t, "https://cluster-keycloak.example.com", authData.Url)
				require.Equal(t, "cluster-keycloak-secret", authData.SecretName)
			},
		},
		{
			name: "error when realm not found and object being deleted returns ErrKeycloakRealmNotFound",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-group",
					Namespace:         "default",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{"test-finalizer"},
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "non-existent-realm",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.ErrorIs(t, err, ErrKeycloakRealmNotFound)
			},
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Nil(t, authData)
			},
		},
		{
			name: "error with unknown realm kind",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: "UnknownKind",
						Name: "test-realm",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unknown realm kind")
			},
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Nil(t, authData)
			},
		},
		{
			name: "error when keycloak not connected",
			object: &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: "test-realm",
					},
				},
			},
			objects: []client.Object{
				&keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test",
						KeycloakRef: common.KeycloakRef{
							Kind: keycloakApi.KeycloakKind,
							Name: "test-keycloak",
						},
					},
				},
				&keycloakApi.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-keycloak",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakSpec{
						Url:    "https://keycloak.example.com",
						Secret: "keycloak-secret",
					},
					Status: keycloakApi.KeycloakStatus{
						Connected: false,
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.ErrorIs(t, err, ErrKeycloakIsNotAvailable)
			},
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Nil(t, authData)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			authData, err := helper.getKeycloakAuthDataFromRealmRef(ctx, tt.object)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, authData)
			}
		})
	}
}

func TestHelper_getKeycloakAuthDataFromRealm_withClusterKeycloakKind(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name      string
		realm     *keycloakApi.KeycloakRealm
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		checkFunc func(t *testing.T, authData *KeycloakAuthData)
	}{
		{
			name: "successfully get auth data with ClusterKeycloak kind",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakAlpha.ClusterKeycloakKind,
						Name: "test-cluster-keycloak",
					},
				},
			},
			objects: []client.Object{
				&keycloakAlpha.ClusterKeycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-keycloak",
					},
					Spec: keycloakAlpha.ClusterKeycloakSpec{
						Url:    "https://cluster-keycloak.example.com",
						Secret: "cluster-keycloak-secret",
					},
					Status: keycloakAlpha.ClusterKeycloakStatus{
						Connected: true,
					},
				},
			},
			wantErr: require.NoError,
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Equal(t, "https://cluster-keycloak.example.com", authData.Url)
				require.Equal(t, "cluster-keycloak-secret", authData.SecretName)
			},
		},
		{
			name: "error when ClusterKeycloak not found",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakAlpha.ClusterKeycloakKind,
						Name: "non-existent-cluster-keycloak",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get cluster keycloak")
			},
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Nil(t, authData)
			},
		},
		{
			name: "error with unknown keycloak kind",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test",
					KeycloakRef: common.KeycloakRef{
						Kind: "UnknownKind",
						Name: "test-keycloak",
					},
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unknown keycloak kind")
			},
			checkFunc: func(t *testing.T, authData *KeycloakAuthData) {
				require.Nil(t, authData)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			helper := &Helper{
				client:            k8sClient,
				operatorNamespace: "default",
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			authData, err := helper.getKeycloakAuthDataFromRealm(ctx, tt.realm)

			tt.wantErr(t, err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, authData)
			}
		})
	}
}
