package chain

import (
	"context"
	"fmt"
	"slices"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const updatePasswordAction = "UPDATE_PASSWORD"

type CreateOrUpdateUser struct {
	k8sClient client.Client
	kClientV2 *keycloakapi.APIClient
}

func NewCreateOrUpdateUser(k8sClient client.Client, kClientV2 *keycloakapi.APIClient) *CreateOrUpdateUser {
	return &CreateOrUpdateUser{k8sClient: k8sClient, kClientV2: kClientV2}
}

func (h *CreateOrUpdateUser) Serve(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	realmName string,
	userCtx *UserContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating or updating user in Keycloak")

	userSpec := user.Spec.DeepCopy()
	addOnly := user.IsReconciliationStrategyAddOnly()

	existing, _, err := h.kClientV2.Users.FindUserByUsername(ctx, realmName, userSpec.Username)
	if err != nil && !keycloakapi.IsNotFound(err) {
		return fmt.Errorf("unable to find user by username: %w", err)
	}

	if keycloakapi.IsNotFound(err) {
		// User does not exist — create
		newUser := keycloakapi.UserRepresentation{
			Username:        &userSpec.Username,
			Enabled:         &userSpec.Enabled,
			EmailVerified:   &userSpec.EmailVerified,
			FirstName:       &userSpec.FirstName,
			LastName:        &userSpec.LastName,
			RequiredActions: &userSpec.RequiredUserActions,
			Email:           &userSpec.Email,
		}

		if len(userSpec.AttributesV2) > 0 {
			newUser.Attributes = makeUserAttributes(nil, userSpec.AttributesV2, addOnly)
		}

		resp, err := h.kClientV2.Users.CreateUser(ctx, realmName, newUser)
		if err != nil {
			return fmt.Errorf("unable to create user: %w", err)
		}

		userID := keycloakapi.GetResourceIDFromResponse(resp)
		if userID == "" {
			return fmt.Errorf("unable to get user ID from response")
		}

		userCtx.UserID = userID

		log.Info("User created successfully", "userID", userID)

		return nil
	}

	// User exists — update
	requiredActions := preserveUpdatePasswordAction(existing.RequiredActions, userSpec.RequiredUserActions)
	existing.Username = &userSpec.Username
	existing.Enabled = &userSpec.Enabled
	existing.EmailVerified = &userSpec.EmailVerified
	existing.FirstName = &userSpec.FirstName
	existing.LastName = &userSpec.LastName
	existing.RequiredActions = &requiredActions
	existing.Email = &userSpec.Email

	if len(userSpec.AttributesV2) > 0 {
		existing.Attributes = makeUserAttributes(existing.Attributes, userSpec.AttributesV2, addOnly)
	}

	if _, err := h.kClientV2.Users.UpdateUser(ctx, realmName, *existing.Id, *existing); err != nil {
		return fmt.Errorf("unable to update user: %w", err)
	}

	userCtx.UserID = *existing.Id

	log.Info("User updated successfully", "userID", *existing.Id)

	return nil
}

// makeUserAttributes merges desired attributes into existing ones respecting addOnly semantics.
func makeUserAttributes(existing *map[string][]string, desired map[string][]string, addOnly bool) *map[string][]string {
	attrs := make(map[string][]string)

	if existing != nil {
		for k, v := range *existing {
			attrs[k] = v
		}
	}

	for k, v := range desired {
		if addOnly {
			current := attrs[k]

			for _, newVal := range v {
				if !slices.Contains(current, newVal) {
					current = append(current, newVal)
				}
			}

			attrs[k] = current
		} else {
			attrs[k] = v
		}
	}

	if !addOnly {
		for k := range attrs {
			if _, exists := desired[k]; !exists {
				delete(attrs, k)
			}
		}
	}

	return &attrs
}

// preserveUpdatePasswordAction merges required actions, preserving UPDATE_PASSWORD
// if it was already present in Keycloak.
func preserveUpdatePasswordAction(current *[]string, desired []string) []string {
	result := slices.Clone(desired)

	if current != nil && slices.Contains(*current, updatePasswordAction) && !slices.Contains(result, updatePasswordAction) {
		result = append(result, updatePasswordAction)
	}

	return result
}
