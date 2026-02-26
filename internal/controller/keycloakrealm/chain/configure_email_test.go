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
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestConfigureEmail_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name        string
		realm       *keycloakApi.KeycloakRealm
		realmClient func(t *testing.T) keycloakv2.RealmClient
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
			realmClient: func(t *testing.T) keycloakv2.RealmClient {
				m := v2mocks.NewMockRealmClient(t)

				m.EXPECT().GetRealm(mock.Anything, "realm").
					Return(&keycloakv2.RealmRepresentation{
						Realm: ptr.To("realm"),
					}, nil, nil)

				m.EXPECT().UpdateRealm(mock.Anything, "realm", mock.MatchedBy(func(rep keycloakv2.RealmRepresentation) bool {
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
			realmClient: func(t *testing.T) keycloakv2.RealmClient {
				m := v2mocks.NewMockRealmClient(t)

				m.EXPECT().GetRealm(mock.Anything, "realm").
					Return(&keycloakv2.RealmRepresentation{
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
			realmClient: func(t *testing.T) keycloakv2.RealmClient {
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
			}
			mockRealm := tt.realmClient(t)
			kClientV2 := &keycloakv2.KeycloakClient{Realms: mockRealm}

			tt.wantErr(t,
				s.ServeRequest(
					ctrl.LoggerInto(
						context.Background(),
						logr.Discard(),
					),
					tt.realm,
					kClientV2,
				),
			)
		})
	}
}
