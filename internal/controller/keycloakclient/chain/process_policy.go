package chain

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

const policyLogKey = "policy"

type ProcessPolicy struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewProcessPolicy(keycloakApiClient keycloak.Client, k8sClient client.Client) *ProcessPolicy {
	return &ProcessPolicy{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient}
}

// Serve method for processing keycloak client policies.
func (h *ProcessPolicy) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	if keycloakClient.Spec.Authorization == nil {
		log.Info("Authorization settings are not specified")
		return nil
	}

	clientID, err := h.keycloakApiClient.GetClientID(keycloakClient.Spec.ClientId, realmName)
	if err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization policies: %s", err.Error()))

		return fmt.Errorf("failed to get client id: %w", err)
	}

	existingPolicies, err := h.keycloakApiClient.GetPolicies(ctx, realmName, clientID)
	if err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization policies: %s", err.Error()))

		return fmt.Errorf("failed to get policies: %w", err)
	}

	for i := 0; i < len(keycloakClient.Spec.Authorization.Policies); i++ {
		log.Info("Processing policy", policyLogKey, keycloakClient.Spec.Authorization.Policies[i].Name)

		var policyRepresentation *gocloak.PolicyRepresentation

		if policyRepresentation, err = h.toPolicyRepresentation(ctx, &keycloakClient.Spec.Authorization.Policies[i], clientID, realmName); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization policies: %s", err.Error()))

			return fmt.Errorf("failed to convert policy: %w", err)
		}

		existingPolicy, ok := existingPolicies[keycloakClient.Spec.Authorization.Policies[i].Name]
		if ok {
			policyRepresentation.ID = existingPolicy.ID
			if err = h.keycloakApiClient.UpdatePolicy(ctx, realmName, clientID, *policyRepresentation); err != nil {
				h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization policies: %s", err.Error()))

				return fmt.Errorf("failed to update policy: %w", err)
			}

			log.Info("Policy updated", policyLogKey, keycloakClient.Spec.Authorization.Policies[i].Name)

			delete(existingPolicies, keycloakClient.Spec.Authorization.Policies[i].Name)

			continue
		}

		if _, err = h.keycloakApiClient.CreatePolicy(ctx, realmName, clientID, *policyRepresentation); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization policies: %s", err.Error()))

			return fmt.Errorf("failed to create policy: %w", err)
		}

		log.Info("Policy created", policyLogKey, keycloakClient.Spec.Authorization.Policies[i].Name)
	}

	if keycloakClient.Spec.ReconciliationStrategy != keycloakApi.ReconciliationStrategyAddOnly {
		if err = h.deletePolicies(ctx, existingPolicies, realmName, clientID); err != nil {
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

func (h *ProcessPolicy) deletePolicies(ctx context.Context, existingPolicies map[string]*gocloak.PolicyRepresentation, realmName string, clientID string) error {
	log := ctrl.LoggerFrom(ctx)

	for name := range existingPolicies {
		if name == "Default Policy" {
			continue
		}

		if err := h.keycloakApiClient.DeletePolicy(ctx, realmName, clientID, *existingPolicies[name].ID); err != nil {
			if !adapter.IsErrNotFound(err) {
				return fmt.Errorf("failed to delete policy: %w", err)
			}
		}

		log.Info("Policy deleted", policyLogKey, name)
	}

	return nil
}

// toPolicyRepresentation converts keycloakApi.Policy to gocloak.PolicyRepresentation.
// nolint:cyclop // it's a conversion method, so it's ok
func (h *ProcessPolicy) toPolicyRepresentation(ctx context.Context, policy *keycloakApi.Policy, clientID, realm string) (*gocloak.PolicyRepresentation, error) {
	keycloakPolicy := getBasePolicyRepresentation(policy)

	switch policy.Type {
	case keycloakApi.PolicyTypeAggregate:
		if err := h.toAggregatePolicyRepresentation(ctx, policy, clientID, realm, keycloakPolicy); err != nil {
			return nil, err
		}
	case "client":
		if err := h.toClientPolicyRepresentation(ctx, policy, realm, keycloakPolicy); err != nil {
			return nil, err
		}
	case keycloakApi.PolicyTypeGroup:
		if err := h.toGroupPolicyRepresentation(ctx, policy, realm, keycloakPolicy); err != nil {
			return nil, err
		}
	case keycloakApi.PolicyTypeRole:
		if err := h.toRolePolicyRepresentation(ctx, policy, realm, keycloakPolicy); err != nil {
			return nil, err
		}
	case keycloakApi.PolicyTypeTime:
		if err := h.toTimePolicyRepresentation(ctx, policy, keycloakPolicy); err != nil {
			return nil, err
		}
	case keycloakApi.PolicyTypeUser:
		if err := h.toUserPolicyRepresentation(ctx, policy, realm, keycloakPolicy); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported policy type %s", policy.Type)
	}

	return keycloakPolicy, nil
}

func (h *ProcessPolicy) toAggregatePolicyRepresentation(
	ctx context.Context,
	policy *keycloakApi.Policy,
	clientID string,
	realm string,
	policyRepresentation *gocloak.PolicyRepresentation,
) error {
	if policy.AggregatedPolicy == nil {
		return fmt.Errorf("aggregatedPolicy spec is not specified")
	}

	existingPolicies, err := h.keycloakApiClient.GetPolicies(ctx, realm, clientID)
	if err != nil {
		return fmt.Errorf("failed to get policies: %w", err)
	}

	aggregatedPolicies := make([]string, 0, len(policy.AggregatedPolicy.Policies))

	for _, p := range policy.AggregatedPolicy.Policies {
		existingPolicy, ok := existingPolicies[p]
		if !ok {
			return fmt.Errorf("policy %s does not exist", p)
		}

		if existingPolicy.ID == nil {
			return fmt.Errorf("policy %s does not have ID", p)
		}

		aggregatedPolicies = append(aggregatedPolicies, *existingPolicy.ID)
	}

	policyRepresentation.AggregatedPolicyRepresentation.Policies = &aggregatedPolicies

	return nil
}

func (h *ProcessPolicy) toClientPolicyRepresentation(ctx context.Context, policy *keycloakApi.Policy, realm string, keycloakPolicy *gocloak.PolicyRepresentation) error {
	if policy.ClientPolicy == nil {
		return fmt.Errorf("clientPolicy spec is not specified")
	}

	existingClients, err := h.keycloakApiClient.GetClients(ctx, realm)
	if err != nil {
		return fmt.Errorf("failed to get clients: %w", err)
	}

	clientPolicy := make([]string, 0, len(policy.ClientPolicy.Clients))

	for _, c := range policy.ClientPolicy.Clients {
		existingClient, ok := existingClients[c]
		if !ok {
			return fmt.Errorf("client %s does not exist", c)
		}

		if existingClient.ID == nil {
			return fmt.Errorf("client %s does not have ID", c)
		}

		clientPolicy = append(clientPolicy, *existingClient.ID)
	}

	keycloakPolicy.Clients = &clientPolicy

	return nil
}

func (h *ProcessPolicy) toGroupPolicyRepresentation(ctx context.Context, policy *keycloakApi.Policy, realm string, keycloakPolicy *gocloak.PolicyRepresentation) error {
	if policy.GroupPolicy == nil {
		return fmt.Errorf("group spec is not specified")
	}

	existingGroups, err := h.keycloakApiClient.GetGroups(ctx, realm)
	if err != nil {
		return fmt.Errorf("failed to get groups: %w", err)
	}

	groupPolicy := make([]gocloak.GroupDefinition, 0, len(policy.GroupPolicy.Groups))

	for _, g := range policy.GroupPolicy.Groups {
		existingGroup, ok := existingGroups[g.Name]
		if !ok {
			return fmt.Errorf("group %s does not exist", g.Name)
		}

		if existingGroup.ID == nil {
			return fmt.Errorf("group %s does not have ID", g.Name)
		}

		extendChildren := g.ExtendChildren
		groupPolicy = append(groupPolicy, gocloak.GroupDefinition{
			ID:             existingGroup.ID,
			ExtendChildren: &extendChildren,
		})
	}

	groupsClaim := policy.GroupPolicy.GroupsClaim
	keycloakPolicy.GroupPolicyRepresentation = gocloak.GroupPolicyRepresentation{
		Groups:      &groupPolicy,
		GroupsClaim: &groupsClaim,
	}

	return nil
}

func (h *ProcessPolicy) toRolePolicyRepresentation(ctx context.Context, policy *keycloakApi.Policy, realm string, keycloakPolicy *gocloak.PolicyRepresentation) error {
	if policy.RolePolicy == nil {
		return fmt.Errorf("role spec is not specified")
	}

	existingRoles, err := h.keycloakApiClient.GetRealmRoles(ctx, realm)
	if err != nil {
		return fmt.Errorf("failed to get realm roles: %w", err)
	}

	rolePolicy := make([]gocloak.RoleDefinition, 0, len(policy.RolePolicy.Roles))

	for _, r := range policy.RolePolicy.Roles {
		existingRole, ok := existingRoles[r.Name]
		if !ok {
			return fmt.Errorf("role %s does not exist", r.Name)
		}

		if existingRole.ID == nil {
			return fmt.Errorf("role %s does not have ID", r.Name)
		}

		required := r.Required
		rolePolicy = append(rolePolicy, gocloak.RoleDefinition{
			ID:       existingRole.ID,
			Required: &required,
		})
	}

	keycloakPolicy.Roles = &rolePolicy

	return nil
}

func (h *ProcessPolicy) toTimePolicyRepresentation(_ context.Context, policy *keycloakApi.Policy, keycloakPolicy *gocloak.PolicyRepresentation) error {
	if policy.TimePolicy == nil {
		return fmt.Errorf("time spec is not specified")
	}

	notBefore := policy.TimePolicy.NotBefore
	notOnOrAfter := policy.TimePolicy.NotOnOrAfter
	dayMonth := policy.TimePolicy.DayMonth
	dayMonthEnd := policy.TimePolicy.DayMonthEnd
	month := policy.TimePolicy.Month
	monthEnd := policy.TimePolicy.MonthEnd
	hour := policy.TimePolicy.Hour
	hourEnd := policy.TimePolicy.HourEnd
	minute := policy.TimePolicy.Minute
	minuteEnd := policy.TimePolicy.MinuteEnd

	keycloakPolicy.TimePolicyRepresentation = gocloak.TimePolicyRepresentation{
		NotBefore:    &notBefore,
		NotOnOrAfter: &notOnOrAfter,
		DayMonth:     &dayMonth,
		DayMonthEnd:  &dayMonthEnd,
		Month:        &month,
		MonthEnd:     &monthEnd,
		Hour:         &hour,
		HourEnd:      &hourEnd,
		Minute:       &minute,
		MinuteEnd:    &minuteEnd,
	}

	return nil
}

func (h *ProcessPolicy) toUserPolicyRepresentation(ctx context.Context, policy *keycloakApi.Policy, realm string, keycloakPolicy *gocloak.PolicyRepresentation) error {
	if policy.UserPolicy == nil {
		return fmt.Errorf("user spec is not specified")
	}

	existingUsers, err := h.keycloakApiClient.GetUsersByNames(ctx, realm, policy.UserPolicy.Users)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	userPolicy := make([]string, 0, len(policy.UserPolicy.Users))

	for _, u := range policy.UserPolicy.Users {
		existingUser, ok := existingUsers[u]
		if !ok {
			return fmt.Errorf("user %s does not exist", u)
		}

		if existingUser.ID == nil {
			return fmt.Errorf("user %s does not have ID", u)
		}

		userPolicy = append(userPolicy, *existingUser.ID)
	}

	keycloakPolicy.Users = &userPolicy

	return nil
}

func getBasePolicyRepresentation(policy *keycloakApi.Policy) *gocloak.PolicyRepresentation {
	keycloakPolicy := &gocloak.PolicyRepresentation{}

	name := policy.Name
	keycloakPolicy.Name = &name

	pType := policy.Type
	keycloakPolicy.Type = &pType

	desc := policy.Description
	decisionStrategy := gocloak.DecisionStrategy(policy.DecisionStrategy)

	keycloakPolicy.DecisionStrategy = &decisionStrategy
	keycloakPolicy.Description = &desc

	logic := gocloak.Logic(policy.Logic)
	keycloakPolicy.Logic = &logic

	return keycloakPolicy
}
