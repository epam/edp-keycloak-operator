package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// MembershipContext holds data that is passed between chain handlers.
type MembershipContext struct {
	// GroupID is the Keycloak group ID, set by ResolveGroup handler.
	GroupID string
	// Resolved holds the spec members found in Keycloak, set by ResolveMembers handler.
	Resolved []ResolvedMember
	// Pending holds the spec members not found in Keycloak, set by ResolveMembers handler.
	Pending []keycloakApi.GroupMember
}

// ResolvedMember is a spec member with its Keycloak user ID.
type ResolvedMember struct {
	Member keycloakApi.GroupMember
	UserID string
}

type GroupMembershipHandler interface {
	Serve(
		ctx context.Context,
		membership *keycloakApi.KeycloakRealmGroupMembership,
		realmName string,
		membershipCtx *MembershipContext,
	) error
}

type Chain struct {
	handlers []GroupMembershipHandler
}

func (ch *Chain) Use(handlers ...GroupMembershipHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(
	ctx context.Context,
	membership *keycloakApi.KeycloakRealmGroupMembership,
	realmName string,
	membershipCtx *MembershipContext,
) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting KeycloakRealmGroupMembership chain")

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, membership, realmName, membershipCtx)
		if err != nil {
			log.Info("KeycloakRealmGroupMembership chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of KeycloakRealmGroupMembership has been finished")

	return nil
}

func MakeChain(kClient *keycloakapi.KeycloakClient) *Chain {
	ch := &Chain{}

	ch.Use(
		NewResolveGroup(kClient),
		NewResolveMembers(kClient),
		NewSyncMembers(kClient),
	)

	return ch
}
