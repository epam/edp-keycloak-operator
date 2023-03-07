package adapter

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type KeycloakAuthFlow struct {
	ID                       string                    `json:"id,omitempty"`
	Alias                    string                    `json:"alias"`
	Description              string                    `json:"description"`
	ProviderID               string                    `json:"providerId"`
	TopLevel                 bool                      `json:"topLevel"`
	BuiltIn                  bool                      `json:"builtIn"`
	ParentName               string                    `json:"-"`
	ChildType                string                    `json:"-"`
	AuthenticationExecutions []AuthenticationExecution `json:"-"`
}

type KeycloakChildAuthFlow struct {
	Alias       string `json:"alias"`
	Description string `json:"description"`
	Provider    string `json:"provider"`
	Type        string `json:"type"`
}

type AuthenticationExecution struct {
	Authenticator       string               `json:"authenticator"`
	Requirement         string               `json:"requirement"`
	Priority            int                  `json:"priority"`
	ParentFlow          string               `json:"parentFlow"`
	AuthenticatorConfig *AuthenticatorConfig `json:"-"`
	AutheticatorFlow    bool                 `json:"autheticatorFlow"`
	ID                  string               `json:"-"`
	Alias               string               `json:"-"`
}

type FlowExecution struct {
	AuthenticationFlow   bool     `json:"authenticationFlow"`
	Configurable         bool     `json:"configurable"`
	Description          string   `json:"description"`
	DisplayName          string   `json:"displayName"`
	FlowID               string   `json:"flowId"`
	ID                   string   `json:"id"`
	Index                int      `json:"index"`
	Level                int      `json:"level"`
	Requirement          string   `json:"requirement"`
	RequirementChoices   []string `json:"requirementChoices"`
	AuthenticationConfig string   `json:"authenticationConfig"`
}

type AuthenticatorConfig struct {
	Alias  string            `json:"alias"`
	Config map[string]string `json:"config"`
}

type orderByPriority []AuthenticationExecution

func (a orderByPriority) Len() int           { return len(a) }
func (a orderByPriority) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a orderByPriority) Less(i, j int) bool { return a[i].Priority < a[j].Priority }

func (a GoCloakAdapter) DeleteAuthFlow(realmName string, flow *KeycloakAuthFlow) error {
	if flow.ParentName != "" {
		execID, err := a.getFlowExecutionID(realmName, flow)
		if err != nil {
			return errors.Wrap(err, "unable to get flow exec id")
		}

		if err := a.deleteFlowExecution(realmName, execID); err != nil {
			return errors.Wrap(err, "unable to delete execution")
		}

		return nil
	}

	flowID, err := a.getAuthFlowID(realmName, flow)
	if err != nil {
		return errors.Wrap(err, "unable to get auth flow")
	}

	if _, _, err := a.unsetBrowserFlow(realmName, flow.Alias); err != nil {
		return errors.Wrapf(err, "unable to unset browser flow for realm: %s, alias: %s", realmName, flow.Alias)
	}

	if err := a.deleteAuthFlow(realmName, flowID); err != nil {
		return errors.Wrap(err, "unable to delete auth flow")
	}

	return nil
}

func (a GoCloakAdapter) SyncAuthFlow(realmName string, flow *KeycloakAuthFlow) error {
	id, err := a.syncBaseAuthFlow(realmName, flow)
	if err != nil {
		return errors.Wrap(err, "unable to sync base auth flow")
	}

	sort.Sort(orderByPriority(flow.AuthenticationExecutions))

	for _, e := range flow.AuthenticationExecutions {
		if e.AutheticatorFlow {
			continue
		}

		e.ParentFlow = id
		if err := a.addAuthFlowExecution(realmName, &e); err != nil {
			return errors.Wrap(err, "unable to add auth execution")
		}
	}

	if err := a.adjustChildFlowsPriority(realmName, flow); err != nil {
		return errors.Wrap(err, "unable to adjust child flow priority")
	}

	return nil
}

func (a GoCloakAdapter) adjustChildFlowsPriority(realmName string, flow *KeycloakAuthFlow) error {
	childFlows := a.makeChildFlows(flow)
	if len(childFlows) == 0 {
		return nil
	}

	flowExecs, err := a.getFlowExecutions(realmName, flow.Alias)
	if err != nil {
		return errors.Wrap(err, "unable to get flow executions")
	}

	for i := range flowExecs {
		if err := a.adjustFlowExecutionPriority(realmName, flow.Alias, &flowExecs[i], len(flowExecs), childFlows); err != nil {
			return err
		}
	}

	return nil
}

func (a GoCloakAdapter) SetRealmBrowserFlow(realmName string, flowAlias string) error {
	realm, err := a.client.GetRealm(context.Background(), a.token.AccessToken, realmName)
	if err != nil {
		return errors.Wrap(err, "unable to get realm")
	}

	realm.BrowserFlow = &flowAlias
	if err := a.client.UpdateRealm(context.Background(), a.token.AccessToken, *realm); err != nil {
		return errors.Wrap(err, "unable to update realm")
	}

	return nil
}

func (a GoCloakAdapter) syncBaseAuthFlow(realmName string, flow *KeycloakAuthFlow) (string, error) {
	authFlowID, err := a.getAuthFlowID(realmName, flow)
	if err != nil {
		if !IsErrNotFound(err) {
			return "", errors.Wrap(err, "unable to get auth flow")
		}

		id, err := a.createAuthFlow(realmName, flow)
		if err != nil {
			return "", errors.Wrap(err, "unable to create auth flow")
		}

		authFlowID = id
	} else {
		if err := a.clearFlowExecutions(realmName, flow.Alias); err != nil {
			return "", errors.Wrap(err, "unable to clear flow executions")
		}
	}

	if err := a.validateChildFlowsCreated(realmName, flow); err != nil {
		return "", errors.Wrap(err, "child flows validation failed")
	}

	return authFlowID, nil
}

func (a GoCloakAdapter) validateChildFlowsCreated(realmName string, flow *KeycloakAuthFlow) error {
	childFlows := 0

	for _, authExec := range flow.AuthenticationExecutions {
		if authExec.AutheticatorFlow {
			childFlows++
		}
	}

	if childFlows == 0 {
		return nil
	}

	childExecs, err := a.getFlowExecutions(realmName, flow.Alias)
	if err != nil {
		return errors.Wrap(err, "unable to get flow executions")
	}

	for i := range childExecs {
		if childExecs[i].AuthenticationFlow && childExecs[i].Level == 0 {
			childFlows--
		}
	}

	if childFlows == 0 {
		return nil
	}

	return errors.New("not all child flows created")
}

func (a GoCloakAdapter) clearFlowExecutions(realmName, flowAlias string) error {
	execs, err := a.getFlowExecutions(realmName, flowAlias)
	if err != nil {
		return errors.Wrap(err, "unable to get flow executions")
	}

	for i := range execs {
		if execs[i].AuthenticationFlow || execs[i].Level > 0 {
			continue
		}

		if err := a.deleteFlowExecution(realmName, execs[i].ID); err != nil {
			return errors.Wrap(err, "unable to delete flow execution")
		}

		// after deleting flow execution, we need to delete its config as well
		// as it is not deleted automatically
		if execs[i].AuthenticationConfig != "" {
			if err := a.deleteAuthFlowConfig(realmName, execs[i].AuthenticationConfig); err != nil {
				return fmt.Errorf("unable to delete flow execution config: %w", err)
			}
		}
	}

	return nil
}

func (a GoCloakAdapter) deleteFlowExecution(realmName, id string) error {
	rsp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			keycloakApiParamId:    id,
		}).
		Delete(a.buildPath(authFlowExecutionDelete))

	if err = a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to delete flow execution")
	}

	return nil
}

func (a GoCloakAdapter) getFlowExecutionID(realmName string, flow *KeycloakAuthFlow) (string, error) {
	execs, err := a.getFlowExecutions(realmName, flow.ParentName)
	if err != nil {
		return "", errors.Wrap(err, "unable to get auth flow executions")
	}

	for i := range execs {
		if execs[i].DisplayName == flow.Alias {
			return execs[i].ID, nil
		}
	}

	return "", NotFoundError("auth flow not found")
}

func (a GoCloakAdapter) getAuthFlowID(realmName string, flow *KeycloakAuthFlow) (string, error) {
	if flow.ParentName != "" {
		execs, err := a.getFlowExecutions(realmName, flow.ParentName)
		if err != nil {
			return "", errors.Wrap(err, "unable to get auth flow executions")
		}

		for i := range execs {
			if execs[i].DisplayName == flow.Alias {
				return execs[i].FlowID, nil
			}
		}

		return "", NotFoundError("auth flow not found")
	}

	flows, err := a.getRealmAuthFlows(realmName)
	if err != nil {
		return "", errors.Wrap(err, "unable to get realm auth flows")
	}

	for i := range flows {
		if flows[i].Alias == flow.Alias {
			return flows[i].ID, nil
		}
	}

	return "", NotFoundError("auth flow not found")
}

func (a GoCloakAdapter) getRealmAuthFlows(realmName string) ([]KeycloakAuthFlow, error) {
	var flows []KeycloakAuthFlow

	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
		}).
		SetResult(&flows).
		Get(a.buildPath(authFlows))

	if err = a.checkError(err, resp); err != nil {
		return nil, errors.Wrap(err, "unable to list auth flow by realm")
	}

	return flows, nil
}

func (a GoCloakAdapter) createAuthFlow(realmName string, flow *KeycloakAuthFlow) (id string, err error) {
	if flow.ParentName != "" {
		return a.createChildAuthFlow(realmName, flow)
	}

	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
		}).
		SetBody(flow).
		Post(a.buildPath(authFlows))

	if err = a.checkError(err, resp); err != nil {
		return "", errors.Wrap(err, "unable to create auth flow in realm")
	}

	id, err = getIDFromResponseLocation(resp.RawResponse)
	if err != nil {
		return "", errors.Wrap(err, "unable to get flow id")
	}

	return
}

func (a GoCloakAdapter) createChildAuthFlow(realmName string, flow *KeycloakAuthFlow) (string, error) {
	rsp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
		}).
		SetBody(KeycloakChildAuthFlow{
			Description: flow.Description,
			Alias:       flow.Alias,
			Provider:    flow.ProviderID,
			Type:        flow.ChildType,
		}).
		Post(a.buildPath(path.Join(authFlows, flow.ParentName, "executions/flow")))

	if err = a.checkError(err, rsp); err != nil {
		return "", errors.Wrap(err, "unable to create child auth flow in realm")
	}

	id, err := getIDFromResponseLocation(rsp.RawResponse)
	if err != nil {
		return "", errors.Wrap(err, "unable to get flow id")
	}

	return id, nil
}

func (a GoCloakAdapter) updateFlowExecution(realmName, parentFlowAlias string, flowExec *FlowExecution) error {
	rsp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			keycloakApiParamAlias: parentFlowAlias,
		}).
		SetBody(flowExec).
		Put(a.buildPath(authFlowExecutionGetUpdate))

	if err = a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to update flow execution")
	}

	return nil
}

func (a GoCloakAdapter) adjustExecutionPriority(realmName, executionID string, delta int) error {
	route := raiseExecutionPriority
	if delta < 0 {
		route = lowerExecutionPriority
	}

	for i := 0; i < int(math.Abs(float64(delta))); i++ {
		rsp, err := a.startRestyRequest().
			SetPathParams(map[string]string{
				keycloakApiParamRealm: realmName,
				keycloakApiParamId:    executionID,
			}).
			SetBody(map[string]string{
				keycloakApiParamRealm: realmName,
				"execution":           executionID,
			}).
			Post(a.buildPath(route))

		if err = a.checkError(err, rsp); err != nil {
			return errors.Wrap(err, "unable to adjust execution priority")
		}
	}

	return nil
}

func (a GoCloakAdapter) getFlowExecutions(realmName, flowAlias string) ([]FlowExecution, error) {
	var execs []FlowExecution
	rsp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			keycloakApiParamAlias: flowAlias,
		}).
		SetResult(&execs).
		Get(a.buildPath(authFlowExecutionGetUpdate))

	if err = a.checkError(err, rsp); err != nil {
		return nil, errors.Wrap(err, "unable get flow executions")
	}

	return execs, nil
}

func (a GoCloakAdapter) deleteAuthFlow(realmName, id string) error {
	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			keycloakApiParamId:    id,
		}).
		Delete(a.buildPath(authFlow))

	if err = a.checkError(err, resp); err != nil {
		return errors.Wrap(err, "unable to delete auth flow")
	}

	return nil
}

func (a GoCloakAdapter) addAuthFlowExecution(realmName string, flowExec *AuthenticationExecution) error {
	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
		}).
		SetBody(flowExec).
		Post(a.buildPath(authFlowExecutionCreate))

	if err = a.checkError(err, resp); err != nil {
		return errors.Wrap(err, "unable to add auth flow execution")
	}

	flowExec.ID, err = getIDFromResponseLocation(resp.RawResponse)
	if err != nil {
		return errors.Wrap(err, "unable to get auth exec id")
	}

	if flowExec.AuthenticatorConfig != nil {
		if err := a.createAuthFlowExecutionConfig(realmName, flowExec); err != nil {
			return errors.Wrap(err, "unable to create auth flow execution config")
		}
	}

	return nil
}

func (a GoCloakAdapter) createAuthFlowExecutionConfig(realmName string, flowExec *AuthenticationExecution) error {
	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			keycloakApiParamId:    flowExec.ID,
		}).
		SetBody(flowExec.AuthenticatorConfig).
		Post(a.buildPath(authFlowExecutionConfig))

	if err = a.checkError(err, resp); err != nil {
		return errors.Wrap(err, "unable to add auth flow execution")
	}

	return nil
}

func getIDFromResponseLocation(response *http.Response) (string, error) {
	location := response.Header.Get("Location")
	if location == "" {
		return "", errors.New("location header is not set or empty")
	}

	locationParts := strings.Split(response.Header.Get("Location"), "/")
	if len(locationParts) == 0 {
		return "", errors.New("location header does not have ID")
	}

	return locationParts[len(locationParts)-1], nil
}

func (a GoCloakAdapter) unsetBrowserFlow(realmName, flowAlias string) (realm *gocloak.RealmRepresentation, isBrowserFlowUnset bool, err error) {
	realm, err = a.client.GetRealm(context.Background(), a.token.AccessToken, realmName)
	if err != nil {
		return nil, false, errors.Wrapf(err, "unable to get realm: %s", realmName)
	}

	if realm.BrowserFlow == nil || *realm.BrowserFlow != flowAlias {
		return realm, false, nil
	}

	authFlows, err := a.getRealmAuthFlows(realmName)
	if err != nil {
		return nil, false, errors.Wrapf(err, "unable to get auth flows for realm: %s", realmName)
	}

	var replaceFlow *KeycloakAuthFlow

	for i := range authFlows {
		if authFlows[i].Alias != flowAlias {
			replaceFlow = &authFlows[i]
			break
		}
	}

	if replaceFlow == nil {
		return nil, false,
			errors.Errorf("unable to delete auth flow: %s, no replacement for browser flow found", flowAlias)
	}

	realm.BrowserFlow = &replaceFlow.Alias
	if err := a.client.UpdateRealm(context.Background(), a.token.AccessToken, *realm); err != nil {
		return nil, false, errors.Wrapf(err, "unable to update realm: %s", realmName)
	}

	return realm, true, nil
}

func (a GoCloakAdapter) makeChildFlows(flow *KeycloakAuthFlow) map[string]AuthenticationExecution {
	childFlows := make(map[string]AuthenticationExecution)

	for i, authExec := range flow.AuthenticationExecutions {
		if authExec.AutheticatorFlow {
			childFlows[authExec.Alias] = flow.AuthenticationExecutions[i]
		}
	}

	return childFlows
}

func (a GoCloakAdapter) adjustFlowExecutionPriority(
	realmName,
	flowAlias string,
	flowExec *FlowExecution,
	flowExecsCount int,
	childFlows map[string]AuthenticationExecution,
) error {
	if flowExec.AuthenticationFlow && flowExec.Level != 0 {
		return nil
	}

	childFlow, ok := childFlows[flowExec.DisplayName]
	if !ok {
		return errors.Errorf("unable to find child flow with name: %s", flowExec.DisplayName)
	}

	if childFlow.Requirement != flowExec.Requirement {
		flowExec.Requirement = childFlow.Requirement
		if err := a.updateFlowExecution(realmName, flowAlias, flowExec); err != nil {
			return errors.Wrap(err, "unable to update flow execution")
		}
	}

	if childFlow.Priority == flowExec.Index {
		return nil
	}

	if childFlow.Priority < 0 || childFlow.Priority > flowExecsCount {
		return errors.Errorf("wrong flow priority, flow name: %s, priority: %d", childFlow.Alias, childFlow.Priority)
	}

	if err := a.adjustExecutionPriority(realmName, flowExec.ID, flowExec.Index-childFlow.Priority); err != nil {
		return errors.Wrap(err, "unable to adjust flow priority")
	}

	return nil
}

func (a GoCloakAdapter) deleteAuthFlowConfig(realmName, configID string) error {
	rsp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			keycloakApiParamId:    configID,
		}).
		Delete(a.buildPath(authFlowConfig))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to delete auth flow config: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) updateRedirectConfig(realm *dto.Realm, configID string) error {
	rsp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm.Name,
			keycloakApiParamId:    configID,
		}).
		SetBody(map[string]interface{}{
			keycloakApiParamAlias: "edp-sso",
			"config": map[string]string{
				"defaultProvider": realm.SsoRealmName,
			},
		}).
		Put(a.buildPath(authFlowConfig))
	if err != nil {
		return fmt.Errorf("unable to update auth flow config: %w", err)
	}

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to update auth flow config: %w", err)
	}

	return nil
}
