package chain

import (
	"context"
	"errors"
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
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/destination"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

func TestConfigureEmail_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name        string
		realm       *keycloakApi.KeycloakRealm
		realmClient func(t *testing.T) keycloakapi.RealmClient
		k8sClient   func(t *testing.T) client.Client
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name: "realm email configured successfully",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "realm",
					Smtp: &common.SMTP{
						Template: common.EmailTemplate{
							From:               "from@mailcom",
							FromDisplayName:    "from test",
							ReplyTo:            "to@mail.com",
							ReplyToDisplayName: "to test",
							EnvelopeFrom:       "envelope@mail.com",
						},
						Connection: common.EmailConnection{
							Host:           "smtp-host",
							Port:           25,
							EnableSSL:      true,
							EnableStartTLS: true,
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
		{
			name: "secret not found",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
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
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
			},
			realmClient: func(t *testing.T) keycloakapi.RealmClient {
				m := v2mocks.NewMockRealmClient(t)

				m.EXPECT().GetRealm(mock.Anything, "realm").
					Return(&keycloakapi.RealmRepresentation{
						Realm: ptr.To("realm"),
					}, nil, nil)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to get password")
			},
		},
		{
			name: "failed to get realm",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "realm",
					Smtp: &common.SMTP{
						Template: common.EmailTemplate{
							From: "from@mailcom",
						},
						Connection: common.EmailConnection{
							Host: "smtp-host",
						},
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
			},
			realmClient: func(t *testing.T) keycloakapi.RealmClient {
				m := v2mocks.NewMockRealmClient(t)

				m.EXPECT().GetRealm(mock.Anything, "realm").
					Return(nil, nil, errors.New("realm not found"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "realm not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := ConfigureEmail{
				client: tt.k8sClient(t),
				guard:  destination.AllowAll(),
			}
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

func TestConfigureRealmEmail(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	const namespace = "default"

	emailSpec := &common.SMTP{
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

	newK8sClient := func(t *testing.T, password string) client.Client {
		t.Helper()

		return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: namespace},
				Data:       map[string][]byte{"secret": []byte(password)},
			},
		).Build()
	}

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

	// Every case seeds an identical secret, so one probe client yields the version token the
	// handler will compute for all of them.
	probeSecret := corev1.Secret{}
	require.NoError(t, newK8sClient(t, "password").Get(
		context.Background(),
		client.ObjectKey{Namespace: namespace, Name: "secret"},
		&probeSecret,
	))

	resolvedHash := secretref.ValuesHashSingle(map[string]string{
		"password": secretref.SecretKeyVersion(&probeSecret, "secret"),
	})

	tests := []struct {
		name        string
		storedHash  string
		forceWrite  bool
		k8sClient   client.Client
		realmClient func(t *testing.T) keycloakapi.RealmClient
		wantHash    string
		wantErr     require.ErrorAssertionFunc
	}{
		{
			// smtpServer already matches spec and the hash matches, so UpdateRealm is
			// intentionally not stubbed here.
			name:       "in sync and hash matches — no write",
			storedHash: resolvedHash,
			forceWrite: false,
			k8sClient:  newK8sClient(t, "password"),
			realmClient: func(t *testing.T) keycloakapi.RealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "realm").
					Return(&keycloakapi.RealmRepresentation{SmtpServer: &inSyncSmtpServer}, nil, nil)

				return m
			},
			wantHash: resolvedHash,
			wantErr:  require.NoError,
		},
		{
			name:       "empty stored hash (upgrade path) — write forced",
			storedHash: "",
			forceWrite: false,
			k8sClient:  newK8sClient(t, "password"),
			realmClient: func(t *testing.T) keycloakapi.RealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "realm").
					Return(&keycloakapi.RealmRepresentation{SmtpServer: &inSyncSmtpServer}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "realm", mock.Anything).Return(nil, nil)

				return m
			},
			wantHash: resolvedHash,
			wantErr:  require.NoError,
		},
		{
			name:       "forceWrite true — write applied even though in sync",
			storedHash: resolvedHash,
			forceWrite: true,
			k8sClient:  newK8sClient(t, "password"),
			realmClient: func(t *testing.T) keycloakapi.RealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "realm").
					Return(&keycloakapi.RealmRepresentation{SmtpServer: &inSyncSmtpServer}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "realm", mock.Anything).Return(nil, nil)

				return m
			},
			wantHash: resolvedHash,
			wantErr:  require.NoError,
		},
		{
			name:       "password rotated — hash mismatch forces write despite matching non-password fields",
			storedHash: secretref.ValuesHashSingle(map[string]string{"password": "secret:secret:secret@stale-uid@1"}),
			forceWrite: false,
			k8sClient:  newK8sClient(t, "password"),
			realmClient: func(t *testing.T) keycloakapi.RealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "realm").
					Return(&keycloakapi.RealmRepresentation{SmtpServer: &inSyncSmtpServer}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "realm", mock.Anything).Return(nil, nil)

				return m
			},
			wantHash: resolvedHash,
			wantErr:  require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotHash, err := ConfigureRealmEmail(
				context.Background(),
				"realm",
				emailSpec,
				namespace,
				tt.realmClient(t),
				tt.k8sClient,
				tt.storedHash,
				tt.forceWrite,
				destination.AllowAll(),
			)

			tt.wantErr(t, err)
			require.Equal(t, tt.wantHash, gotHash)
		})
	}
}
