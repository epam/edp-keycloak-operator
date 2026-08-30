package secretref

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
)

func TestGetValueFromSourceRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sourceRef *common.SourceRef
		k8sClient func(t *testing.T) client.Client
		want      string
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "config map key ref",
			sourceRef: &common.SourceRef{
				ConfigMapKeyRef: &common.ConfigMapKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: "configmap-with-secret",
					},
					Key: "key",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithObjects(
					&v1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "configmap-with-secret",
							Namespace: "default",
						},
						Data: map[string]string{
							"key": "value",
						},
					}).Build()
			},
			want:    "value",
			wantErr: require.NoError,
		},
		{
			name: "secret key ref",
			sourceRef: &common.SourceRef{
				SecretKeyRef: &common.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: "secret-with-value",
					},
					Key: "key",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithObjects(
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "secret-with-value",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"key": []byte("value"),
						},
					}).Build()
			},
			want:    "value",
			wantErr: require.NoError,
		},
		{
			name: "empty source ref",
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			want:    "",
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetValueFromSourceRef(
				context.Background(),
				tt.sourceRef,
				"default",
				tt.k8sClient(t),
			)

			assert.Equal(t, tt.want, got)
			tt.wantErr(t, err)
		})
	}
}

func TestGetValueFromSecretKeySelector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		selector  *common.SecretKeySelector
		k8sClient func(t *testing.T) client.Client
		want      string
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "successful resolution",
			selector: &common.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "my-secret",
				},
				Key: "password",
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithObjects(
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-secret",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"password": []byte("s3cret"),
						},
					}).Build()
			},
			want:    "s3cret",
			wantErr: require.NoError,
		},
		{
			name:     "nil selector",
			selector: nil,
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			want:    "",
			wantErr: require.NoError,
		},
		{
			name: "secret not found",
			selector: &common.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "missing-secret",
				},
				Key: "password",
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			want:    "",
			wantErr: require.Error,
		},
		{
			name: "key not found in secret",
			selector: &common.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "my-secret",
				},
				Key: "missing-key",
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithObjects(
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-secret",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"password": []byte("s3cret"),
						},
					}).Build()
			},
			want:    "",
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetValueFromSecretKeySelector(
				context.Background(),
				tt.selector,
				"default",
				tt.k8sClient(t),
			)

			assert.Equal(t, tt.want, got)
			tt.wantErr(t, err)
		})
	}
}

func TestGetValueFromSourceRefOrVal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sourceRef *common.SourceRefOrVal
		k8sClient func(t *testing.T) client.Client
		want      string
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "secret ref",
			sourceRef: &common.SourceRefOrVal{
				SourceRef: common.SourceRef{
					SecretKeyRef: &common.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "secret-with-value",
						},
						Key: "key",
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithObjects(
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "secret-with-value",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"key": []byte("value"),
						},
					},
				).Build()
			},
			want:    "value",
			wantErr: require.NoError,
		},
		{
			name: "direct value",
			sourceRef: &common.SourceRefOrVal{
				Value: "value",
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			want:    "value",
			wantErr: require.NoError,
		},
		{
			name: "empty source ref",
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			want:    "",
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetValueFromSourceRefOrVal(
				context.Background(),
				tt.sourceRef,
				"default",
				tt.k8sClient(t),
			)

			assert.Equal(t, tt.want, got)
			tt.wantErr(t, err)
		})
	}
}

func TestGetValueAndVersionFromSourceRef_VersionTokens(t *testing.T) {
	t.Parallel()

	k8sClient := fake.NewClientBuilder().WithObjects(
		&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "cm",
				Namespace:       "default",
				UID:             "cm-uid",
				ResourceVersion: "7",
			},
			Data: map[string]string{"key": "cm-value"},
		},
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "sec",
				Namespace:       "default",
				UID:             "sec-uid",
				ResourceVersion: "9",
			},
			Data: map[string][]byte{"key": []byte("sec-value")},
		},
	).Build()

	value, version, err := GetValueAndVersionFromSourceRef(context.Background(), &common.SourceRef{
		ConfigMapKeyRef: &common.ConfigMapKeySelector{
			LocalObjectReference: v1.LocalObjectReference{Name: "cm"},
			Key:                  "key",
		},
	}, "default", k8sClient)
	require.NoError(t, err)
	assert.Equal(t, "cm-value", value)
	assert.Equal(t, "configmap:cm:key@cm-uid@7", version)

	value, version, err = GetValueAndVersionFromSourceRef(context.Background(), &common.SourceRef{
		SecretKeyRef: &common.SecretKeySelector{
			LocalObjectReference: v1.LocalObjectReference{Name: "sec"},
			Key:                  "key",
		},
	}, "default", k8sClient)
	require.NoError(t, err)
	assert.Equal(t, "sec-value", value)
	assert.Equal(t, "secret:sec:key@sec-uid@9", version)
	assert.NotContains(t, version, "sec-value")
}
