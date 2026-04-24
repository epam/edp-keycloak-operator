package chain

import (
	"context"
	"fmt"
	"maps"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

const policyLogKey = "policy"

type ProcessPolicy struct {
	kClient   *keycloakapi.KeycloakClient
	k8sClient client.Client
}

func NewProcessPolicy(kClient *keycloakapi.KeycloakClient, k8sClient client.Client) *ProcessPolicy {
	return &ProcessPolicy{kClient: kClient, k8sClient: k8sClient}
}

// Serve method for processing keycloak client policies.
func (h *ProcessPolicy) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string, clientCtx *ClientContext) error {
	log := ctrl.LoggerFrom(ctx)

	if keycloakClient.Spec.Authorization == nil {
		log.Info("Authorization settings are not specified")
		return nil
	}

	clientUUID := clientCtx.ClientUUID

	policiesList, _, err := h.kClient.Authorization.GetPolicies(ctx, realmName, clientUUID)
	if err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization policies: %s", err.Error()))

		return fmt.Errorf("failed to get policies: %w", err)
	}

	existingPolicies := maputil.SliceToMapSelf(policiesList, func(p keycloakapi.AbstractPolicyRepresentation) (string, bool) {
		return *p.Name, p.Name != nil
	})

	// policiesToDelete tracks orphaned policies for cleanup; kept separate so
	// existingPolicies can still be used as an ID-lookup table for aggregate policies.
	policiesToDelete := make(map[string]keycloakapi.AbstractPolicyRepresentation, len(existingPolicies))
	maps.Copy(policiesToDelete, existingPolicies)

	for i := 0; i < len(keycloakClient.Spec.Authorization.Policies); i++ {
		log.Info("Processing policy", policyLogKey, keycloakClient.Spec.Authorization.Policies[i].Name)

		policyBody, err := h.toPolicyBody(ctx, &keycloakClient.Spec.Authorization.Policies[i], realmName, existingPolicies)
		if err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization policies: %s", err.Error()))

			return fmt.Errorf("failed to convert policy: %w", err)
		}

		policyName := keycloakClient.Spec.Authorization.Policies[i].Name
		policyType := keycloakClient.Spec.Authorization.Policies[i].Type

		existingPolicy, ok := existingPolicies[policyName]
		if ok {
			if existingPolicy.Id == nil {
				return fmt.Errorf("existing policy %s does not have ID", policyName)
			}

			if _, err = h.kClient.Authorization.UpdatePolicy(ctx, realmName, clientUUID, policyType, *existingPolicy.Id, policyBody); err != nil {
				h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization policies: %s", err.Error()))

				return fmt.Errorf("failed to update policy: %w", err)
			}

			log.Info("Policy updated", policyLogKey, policyName)

			delete(policiesToDelete, policyName)

			continue
		}

		createdPolicy, _, err := h.kClient.Authorization.CreatePolicy(ctx, realmName, clientUUID, policyType, policyBody)
		if err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization policies: %s", err.Error()))

			return fmt.Errorf("failed to create policy: %w", err)
		}

		log.Info("Policy created", policyLogKey, policyName)

		if createdPolicy != nil && createdPolicy.Name != nil {
			existingPolicies[*createdPolicy.Name] = keycloakapi.AbstractPolicyRepresentation{
				Id:   createdPolicy.Id,
				Name: createdPolicy.Name,
			}
		}
	}

	if keycloakClient.Spec.ReconciliationStrategy != keycloakApi.ReconciliationStrategyAddOnly {
		if err = h.deletePolicies(ctx, policiesToDelete, realmName, clientUUID); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization policies: %s", err.Error()))

			return err
		}
	}

	h.setSuccessCondition(ctx, keycloakClient, "Authorization policies synchronized")

	return nil
}

func (h *ProcessPolicy) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionAuthorizationPoliciesSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (h *ProcessPolicy) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionAuthorizationPoliciesSynced,
		metav1.ConditionTrue,
		ReasonAuthorizationPoliciesSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (h *ProcessPolicy) deletePolicies(ctx context.Context, existingPolicies map[string]keycloakapi.AbstractPolicyRepresentation, realmName string, clientUUID string) error {
	log := ctrl.LoggerFrom(ctx)

	for name := range existingPolicies {
		if name == "Default Policy" {
			continue
		}

		p := existingPolicies[name]
		if p.Id == nil {
			continue
		}

		if _, err := h.kClient.Authorization.DeletePolicy(ctx, realmName, clientUUID, *p.Id); err != nil {
			if !keycloakapi.IsNotFound(err) {
				return fmt.Errorf("failed to delete policy: %w", err)
			}
		}

		log.Info("Policy deleted", policyLogKey, name)
	}

	return nil
}

// toPolicyBody converts a keycloakApi.Policy to the appropriate policy body struct for the Keycloak API.
// nolint:cyclop // it's a conversion method, so it's ok
func (h *ProcessPolicy) toPolicyBody(
	ctx context.Context,
	policy *keycloakApi.Policy,
	realm string,
	existingPolicies map[string]keycloakapi.AbstractPolicyRepresentation,
) (any, error) {
	base := keycloakapi.PolicyBodyBase{
		Name:             policy.Name,
		Type:             policy.Type,
		Description:      policy.Description,
		DecisionStrategy: keycloakapi.DecisionStrategy(policy.DecisionStrategy),
		Logic:            keycloakapi.Logic(policy.Logic),
	}

	switch policy.Type {
	case keycloakApi.PolicyTypeAggregate:
		return h.toAggregatePolicyBody(policy, existingPolicies, base)
	case keycloakApi.PolicyTypeClient:
		return h.toClientPolicyBody(ctx, policy, realm, base)
	case keycloakApi.PolicyTypeGroup:
		return h.toGroupPolicyBody(ctx, policy, realm, base)
	case keycloakApi.PolicyTypeRole:
		return h.toRolePolicyBody(ctx, policy, realm, base)
	case keycloakApi.PolicyTypeTime:
		return h.toTimePolicyBody(policy, base)
	case keycloakApi.PolicyTypeUser:
		return h.toUserPolicyBody(ctx, policy, realm, base)
	default:
		return nil, fmt.Errorf("unsupported policy type %s", policy.Type)
	}
}

func (h *ProcessPolicy) toAggregatePolicyBody(
	policy *keycloakApi.Policy,
	existingPolicies map[string]keycloakapi.AbstractPolicyRepresentation,
	base keycloakapi.PolicyBodyBase,
) (any, error) {
	if policy.AggregatedPolicy == nil {
		return nil, fmt.Errorf("aggregatedPolicy spec is not specified")
	}

	policies := make([]string, 0, len(policy.AggregatedPolicy.Policies))

	for _, p := range policy.AggregatedPolicy.Policies {
		existingPolicy, ok := existingPolicies[p]
		if !ok {
			return nil, fmt.Errorf("policy %s does not exist", p)
		}

		if existingPolicy.Id == nil {
			return nil, fmt.Errorf("policy %s does not have ID", p)
		}

		policies = append(policies, *existingPolicy.Id)
	}

	return &keycloakapi.AggregatePolicyBody{PolicyBodyBase: base, Policies: policies}, nil
}

func (h *ProcessPolicy) toClientPolicyBody(ctx context.Context, policy *keycloakApi.Policy, realm string, base keycloakapi.PolicyBodyBase) (any, error) {
	if policy.ClientPolicy == nil {
		return nil, fmt.Errorf("clientPolicy spec is not specified")
	}

	clientsList, _, err := h.kClient.Clients.GetClients(ctx, realm, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get clients: %w", err)
	}

	existingClients := maputil.SliceToMapSelf(clientsList, func(c keycloakapi.ClientRepresentation) (string, bool) {
		return *c.ClientId, c.ClientId != nil
	})

	clients := make([]string, 0, len(policy.ClientPolicy.Clients))

	for _, c := range policy.ClientPolicy.Clients {
		existingClient, ok := existingClients[c]
		if !ok {
			return nil, fmt.Errorf("client %s does not exist", c)
		}

		if existingClient.Id == nil {
			return nil, fmt.Errorf("client %s does not have ID", c)
		}

		clients = append(clients, *existingClient.Id)
	}

	return &keycloakapi.ClientPolicyBody{PolicyBodyBase: base, Clients: clients}, nil
}

func (h *ProcessPolicy) toGroupPolicyBody(ctx context.Context, policy *keycloakApi.Policy, realm string, base keycloakapi.PolicyBodyBase) (any, error) {
	if policy.GroupPolicy == nil {
		return nil, fmt.Errorf("group spec is not specified")
	}

	groupsList, _, err := h.kClient.Groups.GetGroups(ctx, realm, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	existingGroups := maputil.SliceToMapSelf(groupsList, func(g keycloakapi.GroupRepresentation) (string, bool) {
		return *g.Name, g.Name != nil
	})

	groups := make([]keycloakapi.GroupDefinition, 0, len(policy.GroupPolicy.Groups))

	for _, g := range policy.GroupPolicy.Groups {
		existingGroup, ok := existingGroups[g.Name]
		if !ok {
			return nil, fmt.Errorf("group %s does not exist", g.Name)
		}

		if existingGroup.Id == nil {
			return nil, fmt.Errorf("group %s does not have ID", g.Name)
		}

		groups = append(groups, keycloakapi.GroupDefinition{
			ID:             *existingGroup.Id,
			ExtendChildren: g.ExtendChildren,
		})
	}

	return &keycloakapi.GroupPolicyBody{
		PolicyBodyBase: base,
		Groups:         groups,
		GroupsClaim:    policy.GroupPolicy.GroupsClaim,
	}, nil
}

func (h *ProcessPolicy) toRolePolicyBody(ctx context.Context, policy *keycloakApi.Policy, realm string, base keycloakapi.PolicyBodyBase) (any, error) {
	if policy.RolePolicy == nil {
		return nil, fmt.Errorf("role spec is not specified")
	}

	rolesList, _, err := h.kClient.Roles.GetRealmRoles(ctx, realm, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get realm roles: %w", err)
	}

	existingRoles := maputil.SliceToMapSelf(rolesList, func(r keycloakapi.RoleRepresentation) (string, bool) {
		return *r.Name, r.Name != nil
	})

	roles := make([]keycloakapi.RoleDefinition, 0, len(policy.RolePolicy.Roles))

	for _, r := range policy.RolePolicy.Roles {
		existingRole, ok := existingRoles[r.Name]
		if !ok {
			return nil, fmt.Errorf("role %s does not exist", r.Name)
		}

		if existingRole.Id == nil {
			return nil, fmt.Errorf("role %s does not have ID", r.Name)
		}

		roles = append(roles, keycloakapi.RoleDefinition{
			ID:       *existingRole.Id,
			Required: r.Required,
		})
	}

	return &keycloakapi.RolePolicyBody{PolicyBodyBase: base, Roles: roles}, nil
}

func (h *ProcessPolicy) toTimePolicyBody(policy *keycloakApi.Policy, base keycloakapi.PolicyBodyBase) (any, error) {
	if policy.TimePolicy == nil {
		return nil, fmt.Errorf("time spec is not specified")
	}

	return &keycloakapi.TimePolicyBody{
		PolicyBodyBase: base,
		NotBefore:      policy.TimePolicy.NotBefore,
		NotOnOrAfter:   policy.TimePolicy.NotOnOrAfter,
		DayMonth:       policy.TimePolicy.DayMonth,
		DayMonthEnd:    policy.TimePolicy.DayMonthEnd,
		Month:          policy.TimePolicy.Month,
		MonthEnd:       policy.TimePolicy.MonthEnd,
		Hour:           policy.TimePolicy.Hour,
		HourEnd:        policy.TimePolicy.HourEnd,
		Minute:         policy.TimePolicy.Minute,
		MinuteEnd:      policy.TimePolicy.MinuteEnd,
	}, nil
}

func (h *ProcessPolicy) toUserPolicyBody(ctx context.Context, policy *keycloakApi.Policy, realm string, base keycloakapi.PolicyBodyBase) (any, error) {
	if policy.UserPolicy == nil {
		return nil, fmt.Errorf("user spec is not specified")
	}

	users := make([]string, 0, len(policy.UserPolicy.Users))

	for _, u := range policy.UserPolicy.Users {
		existingUser, _, err := h.kClient.Users.FindUserByUsername(ctx, realm, u)
		if err != nil {
			return nil, fmt.Errorf("failed to get user %s: %w", u, err)
		}

		if existingUser == nil || existingUser.Id == nil {
			return nil, fmt.Errorf("user %s does not have ID", u)
		}

		users = append(users, *existingUser.Id)
	}

	return &keycloakapi.UserPolicyBody{PolicyBodyBase: base, Users: users}, nil
}
