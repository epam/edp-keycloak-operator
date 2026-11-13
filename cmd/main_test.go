package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

// makeDestinationGuard is the only place the operator's security policy is read from the
// environment.
func TestMakeDestinationGuard(t *testing.T) {
	tests := []struct {
		name        string
		hosts       string
		enforce     string
		wantHosts   []string
		wantEnforce bool
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name:      "unset is an inactive guard",
			wantHosts: []string{},
			wantErr:   require.NoError,
		},
		{
			name:      "empty values are an inactive guard",
			hosts:     "",
			enforce:   "",
			wantHosts: []string{},
			wantErr:   require.NoError,
		},
		{
			name:      "comma separated list with padding",
			hosts:     " keycloak.example.com , smtp.example.com ",
			wantHosts: []string{"keycloak.example.com", "smtp.example.com"},
			wantErr:   require.NoError,
		},
		{
			name:        "enforcement with a list",
			hosts:       "keycloak.example.com",
			enforce:     "true",
			wantHosts:   []string{"keycloak.example.com"},
			wantEnforce: true,
			wantErr:     require.NoError,
		},
		{
			name:        "ParseBool spellings are honoured",
			hosts:       "keycloak.example.com",
			enforce:     "True",
			wantHosts:   []string{"keycloak.example.com"},
			wantEnforce: true,
			wantErr:     require.NoError,
		},
		{
			name:    "a malformed boolean is fatal rather than silently permissive",
			hosts:   "keycloak.example.com",
			enforce: "yes-please",
			wantErr: require.Error,
		},
		{
			name:    "enforcement without a list is fatal",
			enforce: "true",
			wantErr: require.Error,
		},
		{
			name:    "an entry carrying a scheme is fatal",
			hosts:   "https://keycloak.example.com",
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(allowedDestinationHostsEnv, tt.hosts)
			t.Setenv(enforceDestinationAllowlistEnv, tt.enforce)

			guard, err := makeDestinationGuard()
			tt.wantErr(t, err)

			if err != nil {
				return
			}

			assert.ElementsMatch(t, tt.wantHosts, guard.Hosts())
			assert.Equal(t, tt.wantEnforce, guard.Enforcing())
		})
	}
}

// No other kind bypasses the cache.
func TestManagerClientOptions(t *testing.T) {
	opts := managerClientOptions()

	require.NotNil(t, opts.Cache)
	assert.ElementsMatch(t, []client.Object{&corev1.Secret{}, &corev1.ConfigMap{}}, opts.Cache.DisableFor)
}

var errCacheCalled = errors.New("cache called")

type recordedRequest struct {
	method string
	path   string
	query  string
}

type requestRecorder struct {
	mu   sync.Mutex
	reqs []recordedRequest
}

func (r *requestRecorder) add(req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reqs = append(r.reqs, recordedRequest{method: req.Method, path: req.URL.Path, query: req.URL.RawQuery})
}

func (r *requestRecorder) take() []recordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()

	reqs := r.reqs
	r.reqs = nil

	return reqs
}

// fakeCache records the kinds read through it; the embedded nil Cache fails any other call.
type fakeCache struct {
	cache.Cache

	scheme *runtime.Scheme

	mu   sync.Mutex
	gvks []schema.GroupVersionKind
}

func (f *fakeCache) Get(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	return f.record(obj)
}

func (f *fakeCache) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	return f.record(list)
}

func (f *fakeCache) record(obj runtime.Object) error {
	gvk, err := apiutil.GVKForObject(obj, f.scheme)
	if err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.gvks = append(f.gvks, gvk)

	return errCacheCalled
}

func (f *fakeCache) take() []schema.GroupVersionKind {
	f.mu.Lock()
	defer f.mu.Unlock()

	gvks := f.gvks
	f.gvks = nil

	return gvks
}

func newRoutingTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	return scheme
}

func newRoutingTestServer(t *testing.T, rec *requestRecorder) *httptest.Server {
	t.Helper()

	writeJSON := func(w http.ResponseWriter, code int, body any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		assert.NoError(t, json.NewEncoder(w).Encode(body))
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces/ns/secrets/s1":
			writeJSON(w, http.StatusOK, &corev1.Secret{
				TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "s1", Namespace: "ns"},
				Data:       map[string][]byte{"password": []byte("from-server")},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces/ns/configmaps/c1":
			writeJSON(w, http.StatusOK, &corev1.ConfigMap{
				TypeMeta:   metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "ns"},
				Data:       map[string]string{"ca.crt": "from-server"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces/ns/secrets/missing":
			writeJSON(w, http.StatusNotFound, &metav1.Status{
				TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
				Status:   metav1.StatusFailure,
				Message:  `secrets "missing" not found`,
				Reason:   metav1.StatusReasonNotFound,
				Details:  &metav1.StatusDetails{Name: "missing", Kind: "secrets"},
				Code:     http.StatusNotFound,
			})
		default:
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	return srv
}

// Secret and ConfigMap reads go to the API server; CR reads go to the cache.
func TestManagerClientReadRouting(t *testing.T) {
	scheme := newRoutingTestScheme(t)
	rec := &requestRecorder{}
	srv := newRoutingTestServer(t, rec)
	fc := &fakeCache{scheme: scheme}

	cl, err := cluster.New(
		&rest.Config{Host: srv.URL, ContentConfig: rest.ContentConfig{ContentType: runtime.ContentTypeJSON}},
		func(o *cluster.Options) {
			o.Scheme = scheme
			o.Client = managerClientOptions()
			o.MapperProvider = func(*rest.Config, *http.Client) (meta.RESTMapper, error) {
				mapper := meta.NewDefaultRESTMapper(nil)
				mapper.Add(corev1.SchemeGroupVersion.WithKind("Secret"), meta.RESTScopeNamespace)
				mapper.Add(corev1.SchemeGroupVersion.WithKind("ConfigMap"), meta.RESTScopeNamespace)
				mapper.Add(keycloakApi.GroupVersion.WithKind("KeycloakRealm"), meta.RESTScopeNamespace)

				return mapper, nil
			}
			o.NewCache = cache.NewCacheFunc(func(*rest.Config, cache.Options) (cache.Cache, error) {
				return fc, nil
			})
		},
	)
	require.NoError(t, err)

	c := cl.GetClient()
	ctx := context.Background()

	t.Run("secret", func(t *testing.T) {
		rec.take()
		fc.take()

		got := &corev1.Secret{}
		require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: "ns", Name: "s1"}, got))
		assert.Equal(t, "from-server", string(got.Data["password"]))
		assert.Empty(t, fc.take())

		reqs := rec.take()
		assert.Equal(t, []recordedRequest{{method: http.MethodGet, path: "/api/v1/namespaces/ns/secrets/s1"}}, reqs)
	})

	t.Run("configmap", func(t *testing.T) {
		rec.take()
		fc.take()

		got := &corev1.ConfigMap{}
		require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: "ns", Name: "c1"}, got))
		assert.Equal(t, "from-server", got.Data["ca.crt"])
		assert.Empty(t, fc.take())

		reqs := rec.take()
		assert.Equal(t, []recordedRequest{{method: http.MethodGet, path: "/api/v1/namespaces/ns/configmaps/c1"}}, reqs)
	})

	t.Run("missing secret", func(t *testing.T) {
		rec.take()
		fc.take()

		err := c.Get(ctx, client.ObjectKey{Namespace: "ns", Name: "missing"}, &corev1.Secret{})
		require.Error(t, err)
		assert.True(t, apierrors.IsNotFound(err), "got %v", err)
		assert.Empty(t, fc.take())

		reqs := rec.take()
		assert.Equal(t, []recordedRequest{{method: http.MethodGet, path: "/api/v1/namespaces/ns/secrets/missing"}}, reqs)
	})

	t.Run("custom resource stays cached", func(t *testing.T) {
		rec.take()
		fc.take()

		err := c.Get(ctx, client.ObjectKey{Namespace: "ns", Name: "r1"}, &keycloakApi.KeycloakRealm{})
		require.ErrorIs(t, err, errCacheCalled)
		assert.Equal(t, []schema.GroupVersionKind{keycloakApi.GroupVersion.WithKind("KeycloakRealm")}, fc.take())
		assert.Empty(t, rec.take())
	})
}
