package chain

import (
	"context"
	"errors"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// PutRealmRole creates the realm roles listed in spec.realmRoles.
//
// Deprecated: removed together with KeycloakClientSpec.RealmRoles.
type PutRealmRole struct {
	kClient   *keycloakapi.KeycloakClient
	k8sClient client.Client
}

// NewPutRealmRole returns a PutRealmRole handler.
//
// Deprecated: removed together with KeycloakClientSpec.RealmRoles.
func NewPutRealmRole(kClient *keycloakapi.KeycloakClient, k8sClient client.Client) *PutRealmRole {
	return &PutRealmRole{kClient: kClient, k8sClient: k8sClient}
}

func (h *PutRealmRole) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string, _ *ClientContext) error {
	if err := h.putRealmRoles(ctx, keycloakClient, realmName); err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync realm roles: %s", err.Error()))

		return fmt.Errorf("unable to put realm roles: %w", err)
	}

	h.setSuccessCondition(ctx, keycloakClient, "Realm roles synchronized")

	return nil
}

func (h *PutRealmRole) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionRealmRolesSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (h *PutRealmRole) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionRealmRolesSynced,
		metav1.ConditionTrue,
		ReasonRealmRolesSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

// putRealmRoles creates the realm roles listed in spec.realmRoles that do not exist yet.
// A role whose composite is another missing entry is created after that entry.
//
// Deprecated: removed together with KeycloakClientSpec.RealmRoles.
func (h *PutRealmRole) putRealmRoles(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put realm roles")

	if keycloakClient.Spec.RealmRoles == nil || len(*keycloakClient.Spec.RealmRoles) == 0 {
		reqLog.Info("Keycloak client does not have realm roles")
		return nil
	}

	pending, err := h.missingRealmRoles(ctx, realmName, *keycloakClient.Spec.RealmRoles)
	if err != nil {
		return err
	}

	ordered, err := orderByComposite(pending)
	if err != nil {
		return err
	}

	if err := h.requireComposites(ctx, realmName, ordered); err != nil {
		return err
	}

	for _, role := range ordered {
		if err := h.createRealmRole(ctx, realmName, role); err != nil {
			return err
		}
	}

	reqLog.Info("End put realm roles")

	return nil
}

// orderByComposite returns roles in creation order: a role whose composite is another entry of roles
// comes after that entry; independent entries keep their order. Entries in a composite cycle, and
// entries that depend on one, return an error.
//
// Deprecated: removed together with KeycloakClientSpec.RealmRoles.
func orderByComposite(roles []keycloakApi.RealmRole) ([]keycloakApi.RealmRole, error) {
	index := make(map[string]int, len(roles))
	for i, role := range roles {
		index[role.Name] = i
	}

	dependents := make(map[int][]int, len(roles))
	queue := make([]int, 0, len(roles))

	for i, role := range roles {
		parent, inList := index[role.Composite]
		if role.Composite == "" || !inList {
			queue = append(queue, i)
			continue
		}

		dependents[parent] = append(dependents[parent], i)
	}

	ordered := make([]keycloakApi.RealmRole, 0, len(roles))
	placed := make([]bool, len(roles))

	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]

		ordered = append(ordered, roles[i])
		placed[i] = true

		queue = append(queue, dependents[i]...)
	}

	if len(ordered) == len(roles) {
		return ordered, nil
	}

	unresolved := make([]string, 0, len(roles)-len(ordered))

	for i, role := range roles {
		if !placed[i] {
			unresolved = append(unresolved, role.Name)
		}
	}

	return nil, fmt.Errorf("realm roles %q cannot be ordered: composite cycle", unresolved)
}

// requireComposites fails if a composite of roles is neither another entry of roles nor an existing realm role.
//
// Deprecated: removed together with KeycloakClientSpec.RealmRoles.
func (h *PutRealmRole) requireComposites(ctx context.Context, realmName string, roles []keycloakApi.RealmRole) error {
	listed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		listed[role.Name] = struct{}{}
	}

	exists := make(map[string]bool)

	var missing []string

	for _, role := range roles {
		if role.Composite == "" {
			continue
		}

		if _, ok := listed[role.Composite]; ok {
			continue
		}

		found, checked := exists[role.Composite]
		if !checked {
			_, _, err := h.kClient.Roles.GetRealmRole(ctx, realmName, role.Composite)
			if err != nil && !keycloakapi.IsNotFound(err) {
				return fmt.Errorf("error getting composite realm role %s: %w", role.Composite, err)
			}

			found = err == nil
			exists[role.Composite] = found
		}

		if !found {
			missing = append(missing, fmt.Sprintf("%q of %q", role.Composite, role.Name))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("composite realm roles %s not found in realm %q; create them with a KeycloakRealmRole",
			strings.Join(missing, ", "), realmName)
	}

	return nil
}

// missingRealmRoles returns the entries of roles that do not exist in the realm, first entry per name.
//
// Deprecated: removed together with KeycloakClientSpec.RealmRoles.
func (h *PutRealmRole) missingRealmRoles(ctx context.Context, realmName string, roles []keycloakApi.RealmRole) ([]keycloakApi.RealmRole, error) {
	reqLog := ctrl.LoggerFrom(ctx)
	missing := make([]keycloakApi.RealmRole, 0, len(roles))
	seen := make(map[string]struct{}, len(roles))

	for _, role := range roles {
		if _, ok := seen[role.Name]; ok {
			continue
		}

		seen[role.Name] = struct{}{}

		_, _, err := h.kClient.Roles.GetRealmRole(ctx, realmName, role.Name)
		if err == nil {
			reqLog.Info("Realm role already exists", "role", role.Name)
			continue
		}

		if !keycloakapi.IsNotFound(err) {
			return nil, fmt.Errorf("error checking realm role %s: %w", role.Name, err)
		}

		missing = append(missing, role)
	}

	return missing, nil
}

// createRealmRole creates role and adds it to role.Composite.
// Deletes the created role if adding it to role.Composite fails.
//
// Deprecated: removed together with KeycloakClientSpec.RealmRoles.
func (h *PutRealmRole) createRealmRole(ctx context.Context, realmName string, role keycloakApi.RealmRole) error {
	if _, err := h.kClient.Roles.CreateRealmRole(ctx, realmName, keycloakapi.RoleRepresentation{Name: ptr.To(role.Name)}); err != nil {
		return fmt.Errorf("error creating realm role %s: %w", role.Name, err)
	}

	if role.Composite == "" {
		return nil
	}

	if err := h.addToComposite(ctx, realmName, role); err != nil {
		if _, delErr := h.kClient.Roles.DeleteRealmRole(ctx, realmName, role.Name); delErr != nil && !keycloakapi.IsNotFound(delErr) {
			return errors.Join(err, fmt.Errorf("unable to delete realm role %s after failed composite add: %w", role.Name, delErr))
		}

		return err
	}

	return nil
}

// addToComposite adds the created role to role.Composite.
//
// Deprecated: removed together with KeycloakClientSpec.RealmRoles.
func (h *PutRealmRole) addToComposite(ctx context.Context, realmName string, role keycloakApi.RealmRole) error {
	created, _, err := h.kClient.Roles.GetRealmRole(ctx, realmName, role.Name)
	if err != nil {
		return fmt.Errorf("error getting created realm role %s: %w", role.Name, err)
	}

	child, err := keycloakapi.RequireRealmRoleWithID(created, realmName, role.Name)
	if err != nil {
		return fmt.Errorf("created realm role %q: %w", role.Name, err)
	}

	if _, err := h.kClient.Roles.AddRealmRoleComposites(ctx, realmName, role.Composite, []keycloakapi.RoleRepresentation{child}); err != nil {
		return fmt.Errorf("error adding realm role %s to composite realm role %s: %w", role.Name, role.Composite, err)
	}

	return nil
}
