package chain

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// ResolveGroup resolves spec.group to the Keycloak group ID.
type ResolveGroup struct {
	kClient *keycloakapi.KeycloakClient
}

func NewResolveGroup(kClient *keycloakapi.KeycloakClient) *ResolveGroup {
	return &ResolveGroup{kClient: kClient}
}

func (h *ResolveGroup) Serve(
	ctx context.Context,
	membership *keycloakApi.KeycloakRealmGroupMembership,
	realmName string,
	membershipCtx *MembershipContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Resolving group", "group", membership.Spec.Group)

	groupID, err := findGroupID(ctx, h.kClient, realmName, membership.Spec.Group)
	if err != nil {
		return err
	}

	membershipCtx.GroupID = groupID

	return nil
}
