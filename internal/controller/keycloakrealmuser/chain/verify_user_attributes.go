package chain

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const (
	userAttributesPrefix = "user.attributes."
	localeAttribute      = "locale"
)

// Keycloak 26.x root user fields (DefaultUserProfile).
var rootUserAttributes = []string{"username", "email", "firstName", "lastName"}

// VerifyUserAttributes reads the user back and fails when Keycloak did not store the spec attributes.
type VerifyUserAttributes struct {
	kClient *keycloakapi.KeycloakClient
}

func NewVerifyUserAttributes(kClient *keycloakapi.KeycloakClient) *VerifyUserAttributes {
	return &VerifyUserAttributes{kClient: kClient}
}

func (h *VerifyUserAttributes) Serve(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	realmName string,
	userCtx *UserContext,
) error {
	expected := expectedAttributes(user.Spec.AttributesV2)
	if len(expected) == 0 {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Verifying user attributes")

	stored, _, err := h.kClient.Users.GetUser(ctx, realmName, userCtx.UserID)
	if err != nil {
		return fmt.Errorf("unable to get user to verify attributes: %w", err)
	}

	if stored == nil {
		return fmt.Errorf("user %s not found", userCtx.UserID)
	}

	missing := notStoredAttributes(
		expected,
		ptr.Deref(stored.Attributes, nil),
		user.IsReconciliationStrategyAddOnly(),
	)
	if len(missing) > 0 {
		return unstoredAttributesError(user.Spec.RealmRef, missing)
	}

	log.Info("User attributes verified")

	return nil
}

// expectedAttribute is a spec entry as Keycloak keeps it: root fields and blank values removed.
type expectedAttribute struct {
	specName string
	key      string
	values   sets.Set[string]
}

func expectedAttributes(desired map[string][]string) []expectedAttribute {
	expected := make([]expectedAttribute, 0, len(desired))

	for name, values := range desired {
		key := strings.TrimPrefix(name, userAttributesPrefix)
		if slices.Contains(rootUserAttributes, key) {
			continue
		}

		want := sets.New[string]()

		for _, v := range values {
			if !isJavaBlank(v) {
				want.Insert(v)
			}
		}

		if want.Len() > 0 {
			expected = append(expected, expectedAttribute{specName: name, key: key, values: want})
		}
	}

	return expected
}

// isJavaBlank matches String.isBlank: every rune satisfies Character.isWhitespace.
func isJavaBlank(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return !isJavaWhitespace(r) }) < 0
}

func isJavaWhitespace(r rune) bool {
	switch r {
	case '\u00A0', '\u2007', '\u202F':
		return false
	case '\t', '\n', '\u000B', '\f', '\r', '\u001C', '\u001D', '\u001E', '\u001F':
		return true
	}

	return unicode.In(r, unicode.Zs, unicode.Zl, unicode.Zp)
}

// notStoredAttributes returns the sorted spec attribute names whose values Keycloak did not store.
func notStoredAttributes(expected []expectedAttribute, stored map[string][]string, addOnly bool) []string {
	var missing []string

	for _, e := range expected {
		got := sets.New(stored[e.key]...)

		if addOnly && !got.IsSuperset(e.values) || !addOnly && !got.Equal(e.values) {
			missing = append(missing, e.specName)
		}
	}

	slices.Sort(missing)

	return missing
}

func unstoredAttributesError(realmRef common.RealmRef, keys []string) error {
	kind := helper.RealmRefKind(realmRef)

	msg := fmt.Sprintf(
		"keycloak did not store user attributes %q as specified: "+
			"set spec.userProfileConfig.unmanagedAttributePolicy to ADMIN_EDIT or ENABLED in %s %s, "+
			"or declare the attributes in its spec.userProfileConfig.attributes with admin edit permission",
		keys, kind, realmRef.Name,
	)

	if slices.ContainsFunc(keys, func(k string) bool {
		return strings.TrimPrefix(k, userAttributesPrefix) == localeAttribute
	}) {
		msg += "; locale is stored only when realm internationalization is enabled"
	}

	return errors.New(msg)
}
