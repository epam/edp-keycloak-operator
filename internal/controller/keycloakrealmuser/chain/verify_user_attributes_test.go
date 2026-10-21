package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

const (
	verifyRealmName = "test-realm"
	verifyUserID    = "user-1"
	attrBaz         = "baz"
)

func TestNewVerifyUserAttributes(t *testing.T) {
	h := NewVerifyUserAttributes(nil)
	require.NotNil(t, h)
}

func newVerifyUser(
	attrs map[string][]string,
	strategy string,
	realmRef common.RealmRef,
) *keycloakApi.KeycloakRealmUser {
	return &keycloakApi.KeycloakRealmUser{
		ObjectMeta: metav1.ObjectMeta{Name: testUserCRName, Namespace: defaultNamespace},
		Spec: keycloakApi.KeycloakRealmUserSpec{
			RealmRef:               realmRef,
			Username:               testUsername,
			AttributesV2:           attrs,
			ReconciliationStrategy: strategy,
		},
	}
}

func TestVerifyUserAttributes_Serve(t *testing.T) {
	demoRealm := common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: "demo"}

	storedUser := func(attrs map[string][]string) *keycloakapi.UserRepresentation {
		u := &keycloakapi.UserRepresentation{Id: ptr.To(verifyUserID)}
		if attrs != nil {
			u.Attributes = &attrs
		}

		return u
	}

	expectGetUser := func(stored *keycloakapi.UserRepresentation) func(*v2mocks.MockUsersClient) {
		return func(m *v2mocks.MockUsersClient) {
			m.EXPECT().GetUser(context.Background(), verifyRealmName, verifyUserID).
				Return(stored, nil, nil)
		}
	}

	tests := []struct {
		name        string
		user        *keycloakApi.KeycloakRealmUser
		mockSetup   func(*v2mocks.MockUsersClient)
		wantErr     require.ErrorAssertionFunc
		errContains []string
		errExact    string
	}{
		{
			name:      "no attributes - no GetUser",
			user:      newVerifyUser(nil, "", demoRealm),
			mockSetup: func(*v2mocks.MockUsersClient) {},
			wantErr:   require.NoError,
		},
		{
			name:      "empty attributes - no GetUser",
			user:      newVerifyUser(map[string][]string{}, "", demoRealm),
			mockSetup: func(*v2mocks.MockUsersClient) {},
			wantErr:   require.NoError,
		},
		{
			name: "only root keys - no GetUser",
			user: newVerifyUser(map[string][]string{
				"username":                 {"a"},
				"email":                    {"a@example.com"},
				"firstName":                {"A"},
				"lastName":                 {"B"},
				"user.attributes.email":    {"a@example.com"},
				"user.attributes.lastName": {"B"},
			}, "", demoRealm),
			mockSetup: func(*v2mocks.MockUsersClient) {},
			wantErr:   require.NoError,
		},
		{
			name: "only blank values - no GetUser",
			user: newVerifyUser(map[string][]string{
				"empty":      {""},
				"whitespace": {" ", "\t"},
				"nothing":    {},
				"nil":        nil,
			}, "", demoRealm),
			mockSetup: func(*v2mocks.MockUsersClient) {},
			wantErr:   require.NoError,
		},
		{
			name: "all stored",
			user: newVerifyUser(map[string][]string{
				attrBaz: {"jazz", "rock"},
				"one":   {"x"},
			}, "", demoRealm),
			mockSetup: expectGetUser(storedUser(map[string][]string{
				attrBaz: {"rock", "jazz"},
				"one":   {"x"},
			})),
			wantErr: require.NoError,
		},
		{
			name: "root, blank and prefixed attributes next to a stored one",
			user: newVerifyUser(map[string][]string{
				"email":                 {"a@example.com"},
				"user.attributes.team":  {"blue"},
				"mixed":                 {"", "v", "  "},
				"onlyblank":             {" "},
				"user.attributes.email": {"a@example.com"},
			}, "", demoRealm),
			mockSetup: expectGetUser(storedUser(map[string][]string{
				"team":  {"blue"},
				"mixed": {"v"},
			})),
			wantErr: require.NoError,
		},
		{
			name: "values with surrounding spaces and mixed-case key stored as written",
			user: newVerifyUser(map[string][]string{
				"Team": {" x "},
			}, "", demoRealm),
			mockSetup: expectGetUser(storedUser(map[string][]string{
				"Team": {" x "},
			})),
			wantErr: require.NoError,
		},
		{
			name: "mixed-case key stored under another case is not stored",
			user: newVerifyUser(map[string][]string{
				"Team": {"x"},
			}, "", demoRealm),
			mockSetup: expectGetUser(storedUser(map[string][]string{
				"team": {"x"},
			})),
			wantErr:     require.Error,
			errContains: []string{`["Team"]`},
		},
		{
			name: "addOnly allows extra stored values and keys",
			user: newVerifyUser(map[string][]string{
				attrBaz: {"jazz"},
			}, keycloakApi.ReconciliationStrategyAddOnly, demoRealm),
			mockSetup: expectGetUser(storedUser(map[string][]string{
				attrBaz: {"jazz", "rock"},
				"other": {"x"},
			})),
			wantErr: require.NoError,
		},
		{
			name: "addOnly value not stored",
			user: newVerifyUser(map[string][]string{
				attrBaz: {"jazz", "pop"},
			}, keycloakApi.ReconciliationStrategyAddOnly, demoRealm),
			mockSetup: expectGetUser(storedUser(map[string][]string{
				attrBaz: {"jazz", "rock"},
			})),
			wantErr:     require.Error,
			errContains: []string{`["baz"]`},
		},
		{
			name: "attributes dropped - error names keys, field and realm",
			user: newVerifyUser(map[string][]string{
				attrBaz: {"jazz", "rock"},
			}, "", demoRealm),
			mockSetup: expectGetUser(storedUser(nil)),
			wantErr:   require.Error,
			errContains: []string{
				`"baz"`,
				"unmanagedAttributePolicy",
				"spec.userProfileConfig.attributes",
				"KeycloakRealm demo",
			},
		},
		{
			name: "ClusterKeycloakRealm ref is named",
			user: newVerifyUser(map[string][]string{
				attrBaz: {"jazz"},
			}, "", common.RealmRef{Kind: "ClusterKeycloakRealm", Name: "shared"}),
			mockSetup:   expectGetUser(storedUser(nil)),
			wantErr:     require.Error,
			errContains: []string{"ClusterKeycloakRealm shared"},
		},
		{
			name: "empty realmRef kind is named as KeycloakRealm",
			user: newVerifyUser(map[string][]string{
				attrBaz: {"jazz"},
			}, "", common.RealmRef{Name: "demo"}),
			mockSetup:   expectGetUser(storedUser(nil)),
			wantErr:     require.Error,
			errContains: []string{"KeycloakRealm demo"},
		},
		{
			name: "undeclared key dropped next to a stored one lists only the dropped key",
			user: newVerifyUser(map[string][]string{
				"team":  {"blue"},
				attrBaz: {"jazz"},
			}, "", demoRealm),
			mockSetup: expectGetUser(storedUser(map[string][]string{
				"team": {"blue"},
			})),
			wantErr:     require.Error,
			errContains: []string{`["baz"]`},
		},
		{
			name: "value change ignored",
			user: newVerifyUser(map[string][]string{
				attrBaz: {"changed"},
			}, "", demoRealm),
			mockSetup: expectGetUser(storedUser(map[string][]string{
				attrBaz: {"jazz", "rock"},
			})),
			wantErr:     require.Error,
			errContains: []string{`["baz"]`},
		},
		{
			name: "value removal ignored in full strategy",
			user: newVerifyUser(map[string][]string{
				attrBaz: {"jazz"},
			}, "", demoRealm),
			mockSetup: expectGetUser(storedUser(map[string][]string{
				attrBaz: {"jazz", "rock"},
			})),
			wantErr:     require.Error,
			errContains: []string{`["baz"]`},
		},
		{
			name: "duplicate spec values against a single stored value",
			user: newVerifyUser(map[string][]string{
				attrBaz: {"a", "a"},
			}, "", demoRealm),
			mockSetup: expectGetUser(storedUser(map[string][]string{
				attrBaz: {"a"},
			})),
			wantErr: require.NoError,
		},
		{
			name: "duplicate stored values against a single spec value",
			user: newVerifyUser(map[string][]string{
				attrBaz: {"a"},
			}, "", demoRealm),
			mockSetup: expectGetUser(storedUser(map[string][]string{
				attrBaz: {"a", "a"},
			})),
			wantErr: require.NoError,
		},
		{
			name: "prefixed key is looked up without the prefix and reported as written",
			user: newVerifyUser(map[string][]string{
				"user.attributes.team": {"blue"},
			}, "", demoRealm),
			mockSetup:   expectGetUser(storedUser(nil)),
			wantErr:     require.Error,
			errContains: []string{`["user.attributes.team"]`},
		},
		{
			name: "plain and prefixed spec keys with different values in full strategy",
			user: newVerifyUser(map[string][]string{
				"team":                 {"blue"},
				"user.attributes.team": {"red"},
			}, "", demoRealm),
			mockSetup: expectGetUser(storedUser(map[string][]string{
				"team": {"blue"},
			})),
			wantErr:     require.Error,
			errContains: []string{`["user.attributes.team"]`},
		},
		{
			name: "missing keys are sorted with exact text",
			user: newVerifyUser(map[string][]string{
				"c": {"1"},
				"b": {"1"},
				"a": {"2"},
			}, "", demoRealm),
			mockSetup: expectGetUser(storedUser(nil)),
			wantErr:   require.Error,
			errExact: `keycloak did not store user attributes ["a" "b" "c"] as specified: ` +
				`set spec.userProfileConfig.unmanagedAttributePolicy to ADMIN_EDIT or ENABLED in KeycloakRealm demo, ` +
				`or declare the attributes in its spec.userProfileConfig.attributes with admin edit permission`,
		},
		{
			name: "locale not stored - error explains internationalization",
			user: newVerifyUser(map[string][]string{
				"user.attributes.locale": {"de"},
				attrBaz:                  {"jazz"},
			}, "", demoRealm),
			mockSetup: expectGetUser(storedUser(nil)),
			wantErr:   require.Error,
			errExact: `keycloak did not store user attributes ["baz" "user.attributes.locale"] as specified: ` +
				`set spec.userProfileConfig.unmanagedAttributePolicy to ADMIN_EDIT or ENABLED in KeycloakRealm demo, ` +
				`or declare the attributes in its spec.userProfileConfig.attributes with admin edit permission; ` +
				`locale is stored only when realm internationalization is enabled`,
		},
		{
			name: "non-breaking space value is stored and verified",
			user: newVerifyUser(map[string][]string{
				"nbsp": {"\u00A0"},
			}, "", demoRealm),
			mockSetup:   expectGetUser(storedUser(nil)),
			wantErr:     require.Error,
			errContains: []string{`["nbsp"]`},
		},
		{
			name: "control-separator value is blank and not verified",
			user: newVerifyUser(map[string][]string{
				"sep": {"\u001F"},
			}, "", demoRealm),
			mockSetup: func(*v2mocks.MockUsersClient) {},
			wantErr:   require.NoError,
		},
		{
			name: "GetUser error is wrapped",
			user: newVerifyUser(map[string][]string{
				attrBaz: {"jazz"},
			}, "", demoRealm),
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().GetUser(context.Background(), verifyRealmName, verifyUserID).
					Return(nil, nil, errors.New("boom"))
			},
			wantErr:     require.Error,
			errContains: []string{"unable to get user to verify attributes", "boom"},
		},
		{
			name: "GetUser returns nil user",
			user: newVerifyUser(map[string][]string{
				attrBaz: {"jazz"},
			}, "", demoRealm),
			mockSetup:   expectGetUser(nil),
			wantErr:     require.Error,
			errContains: []string{"user " + verifyUserID + " not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usersMock := v2mocks.NewMockUsersClient(t)
			tt.mockSetup(usersMock)

			h := NewVerifyUserAttributes(&keycloakapi.KeycloakClient{Users: usersMock})

			err := h.Serve(context.Background(), tt.user, verifyRealmName, &UserContext{UserID: verifyUserID})
			tt.wantErr(t, err)

			for _, s := range tt.errContains {
				assert.ErrorContains(t, err, s)
			}

			if tt.errExact != "" {
				assert.EqualError(t, err, tt.errExact)
			}
		})
	}
}

func TestIsJavaBlank(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"empty", "", true},
		{"space tab newline", " \t\n\r\f\u000B", true},
		{"U+001C-U+001F", "\u001C\u001D\u001E\u001F", true},
		{"line and paragraph separators", "\u2028\u2029", true},
		{"ideographic and em space", "\u3000\u2003", true},
		{"U+00A0 is not whitespace", "\u00A0", false},
		{"U+2007 is not whitespace", "\u2007", false},
		{"U+202F is not whitespace", "\u202F", false},
		{"U+0085 is not whitespace", "\u0085", false},
		{"text between spaces", " x ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isJavaBlank(tt.value))
		})
	}
}

// UserID is empty here: the chain starts without CreateOrUpdateUser, so the mock matches the empty id.
func TestVerifyUserAttributes_KeepResourceFalse(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	demoRealm := common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: "demo"}

	tests := []struct {
		name        string
		stored      map[string][]string
		wantErr     require.ErrorAssertionFunc
		wantDeleted bool
	}{
		{
			name:        "attributes not stored - CR is kept",
			stored:      nil,
			wantErr:     require.Error,
			wantDeleted: false,
		},
		{
			name:        "attributes stored - CR is deleted",
			stored:      map[string][]string{attrBaz: {"jazz"}},
			wantErr:     require.NoError,
			wantDeleted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := newVerifyUser(map[string][]string{attrBaz: {"jazz"}}, "", demoRealm)
			user.Spec.KeepResource = false

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(user.DeepCopy()).Build()

			usersMock := v2mocks.NewMockUsersClient(t)
			usersMock.EXPECT().GetUser(context.Background(), verifyRealmName, "").
				Return(&keycloakapi.UserRepresentation{Attributes: &tt.stored}, nil, nil)

			ch := &Chain{}
			ch.Use(
				NewVerifyUserAttributes(&keycloakapi.KeycloakClient{Users: usersMock}),
				NewCleanupResource(k8sClient),
			)

			tt.wantErr(t, ch.Serve(context.Background(), user, verifyRealmName))

			err := k8sClient.Get(
				context.Background(),
				types.NamespacedName{Name: testUserCRName, Namespace: defaultNamespace},
				&keycloakApi.KeycloakRealmUser{},
			)
			if tt.wantDeleted {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
