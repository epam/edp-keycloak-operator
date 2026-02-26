package secretref

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSecretRef_MapComponentConfigSecretsRefs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		config     map[string][]string
		client     func(t *testing.T) client.Client
		wantErr    require.ErrorAssertionFunc
		wantConfig map[string][]string
	}{
		{
			name: "config with secret ref",
			config: map[string][]string{
				"bindDn":         {"uid=serviceaccount,cn=users,dc=example,dc=com"},
				"bindCredential": {"$bind-credential:data"},
			},
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "bind-credential",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"data": []byte("secretValue"),
						},
					},
				).Build()
			},
			wantErr: require.NoError,
			wantConfig: map[string][]string{
				"bindDn":         {"uid=serviceaccount,cn=users,dc=example,dc=com"},
				"bindCredential": {"secretValue"},
			},
		},
		{
			name: "skip keycloak ref",
			config: map[string][]string{
				"bindDn":         {"uid=serviceaccount,cn=users,dc=example,dc=com"},
				"bindCredential": {"${bind-credential.Data}"},
			},
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).WithObjects().Build()
			},
			wantErr: require.NoError,
			wantConfig: map[string][]string{
				"bindDn":         {"uid=serviceaccount,cn=users,dc=example,dc=com"},
				"bindCredential": {"${bind-credential.Data}"},
			},
		},
		{
			name: "secret key not found",
			config: map[string][]string{
				"bindDn":         {"uid=serviceaccount,cn=users,dc=example,dc=com"},
				"bindCredential": {"$bind-credential:data"},
			},
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "bind-credential",
							Namespace: "default",
						},
						Data: map[string][]byte{},
					},
				).Build()
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "does not contain key")
			},
			wantConfig: map[string][]string{
				"bindDn":         {"uid=serviceaccount,cn=users,dc=example,dc=com"},
				"bindCredential": {"$bind-credential:data"},
			},
		},
		{
			name: "secret not found",
			config: map[string][]string{
				"bindDn":         {"uid=serviceaccount,cn=users,dc=example,dc=com"},
				"bindCredential": {"$bind-credential:data"},
			},
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).Build()
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get secret")
			},
			wantConfig: map[string][]string{
				"bindDn":         {"uid=serviceaccount,cn=users,dc=example,dc=com"},
				"bindCredential": {"$bind-credential:data"},
			},
		},
		{
			name: "invalid secret ref format",
			config: map[string][]string{
				"bindDn":         {"uid=serviceaccount,cn=users,dc=example,dc=com"},
				"bindCredential": {"$invalid-secret-ref"},
			},
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).Build()
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid config secret  reference")
			},
			wantConfig: map[string][]string{
				"bindDn":         {"uid=serviceaccount,cn=users,dc=example,dc=com"},
				"bindCredential": {"$invalid-secret-ref"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSecretRef(tt.client(t))

			tt.wantErr(t, s.MapComponentConfigSecretsRefs(context.Background(), tt.config, "default"))
			require.Equal(t, tt.wantConfig, tt.config)
		})
	}
}

func TestSecretRef_MapConfigSecretsRefs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		config     map[string]string
		client     func(t *testing.T) client.Client
		wantErr    require.ErrorAssertionFunc
		wantConfig map[string]string
	}{
		{
			name: "config with secret ref",
			config: map[string]string{
				"clientId":     "provider-client",
				"clientSecret": "$client-secret:data",
			},
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "client-secret",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"data": []byte("secretValue"),
						},
					},
				).Build()
			},
			wantErr: require.NoError,
			wantConfig: map[string]string{
				"clientId":     "provider-client",
				"clientSecret": "secretValue",
			},
		},
		{
			name: "skip keycloak ref",
			config: map[string]string{
				"clientId":     "provider-client",
				"clientSecret": "${client-secret.Data}",
			},
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).WithObjects().Build()
			},
			wantErr: require.NoError,
			wantConfig: map[string]string{
				"clientId":     "provider-client",
				"clientSecret": "${client-secret.Data}",
			},
		},
		{
			name: "secret key not found",
			config: map[string]string{
				"clientId":     "provider-client",
				"clientSecret": "$client-secret:data",
			},
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "client-secret",
							Namespace: "default",
						},
						Data: map[string][]byte{},
					},
				).Build()
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "does not contain key")
			},
			wantConfig: map[string]string{
				"clientId":     "provider-client",
				"clientSecret": "$client-secret:data",
			},
		},
		{
			name: "secret not found",
			config: map[string]string{
				"clientId":     "provider-client",
				"clientSecret": "$client-secret:data",
			},
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).Build()
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get secret")
			},
			wantConfig: map[string]string{
				"clientId":     "provider-client",
				"clientSecret": "$client-secret:data",
			},
		},
		{
			name: "invalid secret ref format",
			config: map[string]string{
				"clientId":     "provider-client",
				"clientSecret": "$invalid-secret-ref",
			},
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).Build()
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid config secret  reference")
			},
			wantConfig: map[string]string{
				"clientId":     "provider-client",
				"clientSecret": "$invalid-secret-ref",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSecretRef(tt.client(t))

			tt.wantErr(t, s.MapConfigSecretsRefs(context.Background(), tt.config, "default"))
			require.Equal(t, tt.wantConfig, tt.config)
		})
	}
}

func TestGenerateSecretRef(t *testing.T) {
	t.Parallel()

	type args struct {
		secretName  string
		secretFiled string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should generate secret ref",
			args: args{
				secretName:  "secret",
				secretFiled: "field",
			},
			want: "$secret:field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, GenerateSecretRef(tt.args.secretName, tt.args.secretFiled))
		})
	}
}
