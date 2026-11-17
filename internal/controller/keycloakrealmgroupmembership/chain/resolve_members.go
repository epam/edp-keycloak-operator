package chain

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// ResolveMembers looks up the users of spec.members. It never creates users:
// a member whose user does not exist yet is recorded as pending.
type ResolveMembers struct {
	kClient *keycloakapi.KeycloakClient
}

func NewResolveMembers(kClient *keycloakapi.KeycloakClient) *ResolveMembers {
	return &ResolveMembers{kClient: kClient}
}

func (h *ResolveMembers) Serve(
	ctx context.Context,
	membership *keycloakApi.KeycloakRealmGroupMembership,
	realmName string,
	membershipCtx *MembershipContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Resolving group members")

	membershipCtx.Resolved = nil
	membershipCtx.Pending = nil

	seen := make(map[keycloakApi.GroupMember]struct{}, len(membership.Spec.Members))

	for _, member := range membership.Spec.Members {
		if _, ok := seen[member]; ok {
			continue
		}

		seen[member] = struct{}{}

		userID, err := findUser(ctx, h.kClient, realmName, member)
		if err != nil {
			if keycloakapi.IsNotFound(err) {
				log.Info("User not found, member is pending", "member", memberLabel(member))

				membershipCtx.Pending = append(membershipCtx.Pending, member)

				continue
			}

			return err
		}

		membershipCtx.Resolved = append(membershipCtx.Resolved, ResolvedMember{Member: member, UserID: userID})
	}

	return nil
}
