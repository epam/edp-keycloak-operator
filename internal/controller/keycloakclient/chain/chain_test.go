package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type testHandler struct {
	err    error
	called bool
}

func (h *testHandler) Serve(_ context.Context, _ *keycloakApi.KeycloakClient, _ string, _ *ClientContext) error {
	h.called = true
	return h.err
}

func TestChain_Serve(t *testing.T) {
	tests := []struct {
		name     string
		handlers []*testHandler
		wantErr  require.ErrorAssertionFunc
		check    func(t *testing.T, handlers []*testHandler)
	}{
		{
			name:     "empty chain succeeds",
			handlers: nil,
			wantErr:  require.NoError,
		},
		{
			name: "all handlers succeed",
			handlers: []*testHandler{
				{},
				{},
				{},
			},
			wantErr: require.NoError,
			check: func(t *testing.T, handlers []*testHandler) {
				for i, h := range handlers {
					require.True(t, h.called, "handler %d should have been called", i)
				}
			},
		},
		{
			name: "stops on first error",
			handlers: []*testHandler{
				{},
				{err: errors.New("boom")},
				{},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to serve handler")
				require.Contains(t, err.Error(), "boom")
			},
			check: func(t *testing.T, handlers []*testHandler) {
				require.True(t, handlers[0].called, "handler 0 should have been called")
				require.True(t, handlers[1].called, "handler 1 should have been called")
				require.False(t, handlers[2].called, "handler 2 should not have been called")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := &Chain{}
			for _, h := range tt.handlers {
				ch.Use(h)
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			err := ch.Serve(ctx, &keycloakApi.KeycloakClient{}, "realm")
			tt.wantErr(t, err)

			if tt.check != nil {
				tt.check(t, tt.handlers)
			}
		})
	}
}

func TestMakeChain(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	k8sClient := fake.NewClientBuilder().WithScheme(s).Build()

	c := MakeChain(&keycloakapi.APIClient{}, k8sClient)

	require.Len(t, c.handlers, 11)
}
