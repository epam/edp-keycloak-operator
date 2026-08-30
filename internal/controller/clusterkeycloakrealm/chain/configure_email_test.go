package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

func TestConfigureEmail_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name        string
		realm       *keycloakApi.ClusterKeycloakRealm
		realmClient func(t *testing.T) keycloakapi.RealmClient
		k8sClient   func(t *testing.T) client.Client
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name: "realm email configured successfully",
			realm: &keycloakApi.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name: "realm",
				},
				Spec: keycloakApi.ClusterKeycloakRealmSpec{
					RealmName: "realm",
					Smtp: &common.SMTP{
						Template: common.EmailTemplate{
							From: "from@mailcom",
						},
						Connection: common.EmailConnection{
							Host: "smtp-host",
							Authentication: &common.EmailAuthentication{
								Username: common.SourceRefOrVal{
									Value: "username",
								},
								Password: common.SourceRef{
									SecretKeyRef: &common.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "secret",
										},
										Key: "secret",
									},
								},
							},
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "secret",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"secret": []byte("password"),
						},
					},
				).Build()
			},
			realmClient: func(t *testing.T) keycloakapi.RealmClient {
				m := v2mocks.NewMockRealmClient(t)

				m.EXPECT().GetRealm(mock.Anything, "realm").
					Return(&keycloakapi.RealmRepresentation{
						Realm: ptr.To("realm"),
					}, nil, nil)

				m.EXPECT().UpdateRealm(mock.Anything, "realm", mock.MatchedBy(func(rep keycloakapi.RealmRepresentation) bool {
					return rep.SmtpServer != nil &&
						(*rep.SmtpServer)["from"] == "from@mailcom" &&
						(*rep.SmtpServer)["user"] == "username" &&
						(*rep.SmtpServer)["password"] == "password"
				})).Return(nil, nil)

				return m
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewConfigureEmail(tt.k8sClient(t), "default")
			mockRealm := tt.realmClient(t)
			kClient := &keycloakapi.KeycloakClient{Realms: mockRealm}

			tt.wantErr(t,
				s.ServeRequest(
					ctrl.LoggerInto(
						context.Background(),
						logr.Discard(),
					),
					tt.realm,
					kClient,
				),
			)
		})
	}
}

// TestConfigureEmail_ServeRequest_Idempotency exercises the ConfigSecretsHash/ObservedGeneration
// plumbing this wrapper passes to the shared keycloakrealmchain.ConfigureRealmEmail.
func TestConfigureEmail_ServeRequest_Idempotency(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	const namespace = "default"

	smtpSpec := &common.SMTP{
		Template: common.EmailTemplate{From: "from@mail.com"},
		Connection: common.EmailConnection{
			Host: "smtp-host",
			Port: 25,
			Authentication: &common.EmailAuthentication{
				Username: common.SourceRefOrVal{Value: "username"},
				Password: common.SourceRef{
					SecretKeyRef: &common.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "secret"},
						Key:                  "secret",
					},
				},
			},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: namespace},
			Data:       map[string][]byte{"secret": []byte("password")},
		},
	).Build()

	inSyncSmtpServer := map[string]string{
		"from":               "from@mail.com",
		"fromDisplayName":    "",
		"replyTo":            "",
		"replyToDisplayName": "",
		"envelopeFrom":       "",
		"host":               "smtp-host",
		"port":               "25",
		"ssl":                "false",
		"starttls":           "false",
		"auth":               "true",
		"user":               "username",
		"password":           keycloakapi.MaskedSecretValue,
	}

	// Probe the seeded secret so the expected hash uses the same version token the handler
	// computes.
	probeSecret := corev1.Secret{}
	require.NoError(t, k8sClient.Get(
		context.Background(),
		client.ObjectKey{Namespace: namespace, Name: "secret"},
		&probeSecret,
	))

	resolvedHash := secretref.ValuesHashSingle(map[string]string{
		"password": secretref.SecretKeyVersion(&probeSecret, "secret"),
	})

	tests := []struct {
		name        string
		realm       *keycloakApi.ClusterKeycloakRealm
		realmClient func(t *testing.T) keycloakapi.RealmClient
	}{
		{
			// smtpServer already matches spec, hash matches, generation unchanged, so
			// UpdateRealm is intentionally not stubbed here.
			name: "in sync and hash matches — no write",
			realm: &keycloakApi.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{Name: "realm", Generation: 1},
				Spec:       keycloakApi.ClusterKeycloakRealmSpec{RealmName: "realm", Smtp: smtpSpec},
				Status: keycloakApi.ClusterKeycloakRealmStatus{
					ObservedGeneration: 1,
					ConfigSecretsHash:  resolvedHash,
				},
			},
			realmClient: func(t *testing.T) keycloakapi.RealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "realm").
					Return(&keycloakapi.RealmRepresentation{SmtpServer: &inSyncSmtpServer}, nil, nil)

				return m
			},
		},
		{
			name: "empty stored hash (upgrade path) — write forced",
			realm: &keycloakApi.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{Name: "realm", Generation: 1},
				Spec:       keycloakApi.ClusterKeycloakRealmSpec{RealmName: "realm", Smtp: smtpSpec},
				Status:     keycloakApi.ClusterKeycloakRealmStatus{ObservedGeneration: 1},
			},
			realmClient: func(t *testing.T) keycloakapi.RealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "realm").
					Return(&keycloakapi.RealmRepresentation{SmtpServer: &inSyncSmtpServer}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "realm", mock.Anything).Return(nil, nil)

				return m
			},
		},
		{
			name: "generation changed — write forced even though in sync",
			realm: &keycloakApi.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{Name: "realm", Generation: 2},
				Spec:       keycloakApi.ClusterKeycloakRealmSpec{RealmName: "realm", Smtp: smtpSpec},
				Status: keycloakApi.ClusterKeycloakRealmStatus{
					ObservedGeneration: 1,
					ConfigSecretsHash:  resolvedHash,
				},
			},
			realmClient: func(t *testing.T) keycloakapi.RealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "realm").
					Return(&keycloakapi.RealmRepresentation{SmtpServer: &inSyncSmtpServer}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "realm", mock.Anything).Return(nil, nil)

				return m
			},
		},
		{
			name: "password rotated — hash mismatch forces write despite matching non-password fields",
			realm: &keycloakApi.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{Name: "realm", Generation: 1},
				Spec:       keycloakApi.ClusterKeycloakRealmSpec{RealmName: "realm", Smtp: smtpSpec},
				Status: keycloakApi.ClusterKeycloakRealmStatus{
					ObservedGeneration: 1,
					ConfigSecretsHash:  secretref.ValuesHashSingle(map[string]string{"password": "secret:secret:secret@stale-uid@1"}),
				},
			},
			realmClient: func(t *testing.T) keycloakapi.RealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "realm").
					Return(&keycloakapi.RealmRepresentation{SmtpServer: &inSyncSmtpServer}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "realm", mock.Anything).Return(nil, nil)

				return m
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := NewConfigureEmail(k8sClient, namespace)
			kClient := &keycloakapi.KeycloakClient{Realms: tt.realmClient(t)}

			err := s.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.realm, kClient)
			require.NoError(t, err)
			require.Equal(t, resolvedHash, tt.realm.Status.ConfigSecretsHash)
		})
	}
}
