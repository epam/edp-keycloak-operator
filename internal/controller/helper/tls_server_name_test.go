package helper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/destination"
	"github.com/epam/edp-keycloak-operator/pkg/fakehttp"
)

const (
	tlsTestOperatorNamespace = "operator-ns"
	tlsTestCertName          = "keycloak.example.com"
)

// tlsTestServer serves the Keycloak auth endpoints on 127.0.0.1 with a certificate for certName only.
func tlsTestServer(t *testing.T, certName string) (string, string) {
	t.Helper()

	server, caCertPEM, err := fakehttp.NewServerBuilder().AddKeycloakAuthResponders().BuildAndStartTLS(certName)
	require.NoError(t, err)
	t.Cleanup(server.Close)

	return server.GetURL(), caCertPEM
}

func tlsTestCACertRef() *common.SourceRef {
	return &common.SourceRef{
		SecretKeyRef: &common.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: testKeycloakSecretName},
			Key:                  "ca.crt",
		},
	}
}

func tlsTestSecret(namespace, caCertPEM string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testKeycloakSecretName, Namespace: namespace},
		Data: map[string][]byte{
			"username":      []byte("admin"),
			testPasswordKey: []byte("admin123"),
			"ca.crt":        []byte(caCertPEM),
		},
	}
}

func tlsTestHelper(t *testing.T, guard *destination.Guard, objects ...client.Object) *Helper {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, keycloakAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	h, err := MakeHelper(fake.NewClientBuilder().WithScheme(s).WithObjects(objects...).Build(), s,
		tlsTestOperatorNamespace, guard)
	require.NoError(t, err)

	return h
}

func tlsTestKeycloak(url, caCertPEM, tlsServerName string) []client.Object {
	return []client.Object{
		&keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{Name: testKeycloakName, Namespace: defaultNamespace},
			Spec: keycloakApi.KeycloakSpec{
				Url:           url,
				Secret:        testKeycloakSecretName,
				CACert:        tlsTestCACertRef(),
				TLSServerName: common.TLSServerName(tlsServerName),
			},
			Status: keycloakApi.KeycloakStatus{Connected: true},
		},
		tlsTestSecret(defaultNamespace, caCertPEM),
	}
}

func tlsTestClusterKeycloak(url, caCertPEM, tlsServerName string) []client.Object {
	return []client.Object{
		&keycloakAlpha.ClusterKeycloak{
			ObjectMeta: metav1.ObjectMeta{Name: testKeycloakName},
			Spec: keycloakAlpha.ClusterKeycloakSpec{
				Url:           url,
				Secret:        testKeycloakSecretName,
				CACert:        tlsTestCACertRef(),
				TLSServerName: common.TLSServerName(tlsServerName),
			},
			Status: keycloakAlpha.ClusterKeycloakStatus{Connected: true},
		},
		tlsTestSecret(tlsTestOperatorNamespace, caCertPEM),
	}
}

func tlsTestRealm(keycloakKind string) *keycloakApi.KeycloakRealm {
	return &keycloakApi.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{Name: testRealmName, Namespace: defaultNamespace},
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName:   testResourceName,
			KeycloakRef: common.KeycloakRef{Kind: keycloakKind, Name: testKeycloakName},
		},
	}
}

func tlsTestClusterRealm() *keycloakAlpha.ClusterKeycloakRealm {
	return &keycloakAlpha.ClusterKeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{Name: testRealmName},
		Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
			RealmName:          testResourceName,
			ClusterKeycloakRef: testKeycloakName,
		},
	}
}

func tlsTestGroup(realmKind string) *keycloakApi.KeycloakRealmGroup {
	return &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testGroupName, Namespace: defaultNamespace},
		Spec: keycloakApi.KeycloakRealmGroupSpec{
			RealmRef: common.RealmRef{Kind: realmKind, Name: testRealmName},
		},
	}
}

// Every path from a realm to the api client carries spec.tlsServerName; without it the handshake fails.
func TestCreateKeycloakClient_TLSServerNameReachesEveryPath(t *testing.T) {
	t.Parallel()

	url, caCertPEM := tlsTestServer(t, tlsTestCertName)

	tests := []struct {
		name    string
		objects func(tlsServerName string) []client.Object
		create  func(ctx context.Context, h *Helper) error
	}{
		{
			name: "KeycloakRealm with Keycloak",
			objects: func(tlsServerName string) []client.Object {
				return append(tlsTestKeycloak(url, caCertPEM, tlsServerName), tlsTestRealm(keycloakApi.KeycloakKind))
			},
			create: func(ctx context.Context, h *Helper) error {
				_, err := h.CreateKeycloakClientFromRealm(ctx, tlsTestRealm(keycloakApi.KeycloakKind))
				return err
			},
		},
		{
			name: "KeycloakRealm with ClusterKeycloak",
			objects: func(tlsServerName string) []client.Object {
				return append(tlsTestClusterKeycloak(url, caCertPEM, tlsServerName),
					tlsTestRealm(keycloakAlpha.ClusterKeycloakKind))
			},
			create: func(ctx context.Context, h *Helper) error {
				_, err := h.CreateKeycloakClientFromRealm(ctx, tlsTestRealm(keycloakAlpha.ClusterKeycloakKind))
				return err
			},
		},
		{
			name: "ClusterKeycloakRealm",
			objects: func(tlsServerName string) []client.Object {
				return append(tlsTestClusterKeycloak(url, caCertPEM, tlsServerName), tlsTestClusterRealm())
			},
			create: func(ctx context.Context, h *Helper) error {
				_, err := h.CreateKeycloakClientFromClusterRealm(ctx, tlsTestClusterRealm())
				return err
			},
		},
		{
			name: "realm ref to KeycloakRealm",
			objects: func(tlsServerName string) []client.Object {
				return append(tlsTestKeycloak(url, caCertPEM, tlsServerName), tlsTestRealm(keycloakApi.KeycloakKind))
			},
			create: func(ctx context.Context, h *Helper) error {
				_, err := h.CreateKeycloakClientFromRealmRef(ctx, tlsTestGroup(keycloakApi.KeycloakRealmKind))
				return err
			},
		},
		{
			name: "realm ref to ClusterKeycloakRealm",
			objects: func(tlsServerName string) []client.Object {
				return append(tlsTestClusterKeycloak(url, caCertPEM, tlsServerName), tlsTestClusterRealm())
			},
			create: func(ctx context.Context, h *Helper) error {
				_, err := h.CreateKeycloakClientFromRealmRef(ctx, tlsTestGroup(keycloakAlpha.ClusterKeycloakRealmKind))
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			withName := tlsTestHelper(t, destination.AllowAll(), tt.objects(tlsTestCertName)...)
			require.NoError(t, tt.create(ctx, withName))

			withoutName := tlsTestHelper(t, destination.AllowAll(), tt.objects("")...)
			require.ErrorContains(t, tt.create(ctx, withoutName), "x509")
		})
	}
}

func TestCreateKeycloakClientFromKeycloak_TLSServerNameWithInsecureSkipVerify(t *testing.T) {
	t.Parallel()

	url, caCertPEM := tlsTestServer(t, "other.example.com")
	objects := tlsTestKeycloak(url, caCertPEM, tlsTestCertName)
	kc := objects[0].(*keycloakApi.Keycloak)
	kc.Spec.InsecureSkipVerify = true

	_, err := tlsTestHelper(t, destination.AllowAll(), objects...).CreateKeycloakClientFromKeycloak(context.Background(), kc)

	require.NoError(t, err)
}

func TestCreateKeycloakClientFromKeycloak_WarnModePermitsUnlistedTLSServerName(t *testing.T) {
	t.Parallel()

	url, caCertPEM := tlsTestServer(t, tlsTestCertName)
	objects := tlsTestKeycloak(url, caCertPEM, tlsTestCertName)

	guard, err := destination.New([]string{"127.0.0.1"}, false)
	require.NoError(t, err)

	_, err = tlsTestHelper(t, guard, objects...).
		CreateKeycloakClientFromKeycloak(context.Background(), objects[0].(*keycloakApi.Keycloak))

	require.NoError(t, err)
}

func TestCreateKeycloakClientFromKeycloak_EnforcingPermitsListedTLSServerName(t *testing.T) {
	t.Parallel()

	url, caCertPEM := tlsTestServer(t, tlsTestCertName)
	objects := tlsTestKeycloak(url, caCertPEM, tlsTestCertName)

	guard, err := destination.New([]string{"127.0.0.1", tlsTestCertName}, true)
	require.NoError(t, err)

	_, err = tlsTestHelper(t, guard, objects...).
		CreateKeycloakClientFromKeycloak(context.Background(), objects[0].(*keycloakApi.Keycloak))

	require.NoError(t, err)
}
