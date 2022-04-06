package adapter

import (
	"context"
	"math"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/Nerzal/gocloak/v10"
	"github.com/pkg/errors"
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
	AuthenticationFlow bool     `json:"authenticationFlow"`
	Configurable       bool     `json:"configurable"`
	Description        string   `json:"description"`
	DisplayName        string   `json:"displayName"`
	FlowID             string   `json:"flowId"`
	ID                 string   `json:"id"`
	Index              int      `json:"index"`
	Level              int      `json:"level"`
	Requirement        string   `json:"requirement"`
	RequirementChoices []string `json:"requirementChoices"`
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
			return errors.Wrap(err, "unabel to get flow exec id")
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
	childFlows := make(map[string]AuthenticationExecution)
	for i, authExec := range flow.AuthenticationExecutions {
		if authExec.AutheticatorFlow {
			childFlows[authExec.Alias] = flow.AuthenticationExecutions[i]
		}
	}

	if len(childFlows) == 0 {
		return nil
	}

	flowExecs, err := a.getFlowExecutions(realmName, flow.Alias)
	if err != nil {
		return errors.Wrap(err, "unable to get flow executions")
	}

	for _, fe := range flowExecs {
		if fe.AuthenticationFlow && fe.Level == 0 {
			childFlow, ok := childFlows[fe.DisplayName]
			if !ok {
				return errors.Errorf("unable to find child flow with name: %s", fe.DisplayName)
			}

			if childFlow.Requirement != fe.Requirement {
				fe.Requirement = childFlow.Requirement
				if err := a.updateFlowExecution(realmName, flow.Alias, &fe); err != nil {
					return errors.Wrap(err, "unable to update flow execution")
				}
			}

			if childFlow.Priority == fe.Index {
				continue
			}

			if childFlow.Priority < 0 || childFlow.Priority > len(flowExecs) {
				return errors.Errorf("wrong flow priority, flow name: %s, priority: %d", childFlow.Alias,
					childFlow.Priority)
			}

			if err := a.adjustExecutionPriority(realmName, fe.ID, fe.Index-childFlow.Priority); err != nil {
				return errors.Wrap(err, "unable to adjust flow priority")
			}
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
	if err != nil && !IsErrNotFound(errors.Cause(err)) {
		return "", errors.Wrap(err, "unable to get auth flow")
	} else if err == nil {
		if err := a.clearFlowExecutions(realmName, flow.Alias); err != nil {
			return "", errors.Wrap(err, "unable to clear flow executions")
		}
	} else {
		id, err := a.createAuthFlow(realmName, flow)
		if err != nil {
			return "", errors.Wrap(err, "unable to create auth flow")
		}
		authFlowID = id
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

	for _, exec := range childExecs {
		if exec.AuthenticationFlow && exec.Level == 0 {
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

	for _, exec := range execs {
		if exec.AuthenticationFlow || exec.Level > 0 {
			continue
		}

		if err := a.deleteFlowExecution(realmName, exec.ID); err != nil {
			return errors.Wrap(err, "unable to delete flow execution")
		}
	}

	return nil
}

func (a GoCloakAdapter) deleteFlowExecution(realmName, id string) error {
	rsp, err := a.startRestyRequest().SetPathParams(map[string]string{
		"realm": realmName,
		"id":    id,
	}).Delete(a.basePath + authFlowExecutionDelete)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to delete flow execution")
	}

	return nil
}

func (a GoCloakAdapter) getFlowExecutionID(realmName string, flow *KeycloakAuthFlow) (string, error) {
	if flow.ParentName == "" {
		return "", errors.New("flow is not execution")
	}

	execs, err := a.getFlowExecutions(realmName, flow.ParentName)
	if err != nil {
		return "", errors.Wrap(err, "unable to get auth flow executions")
	}

	for _, e := range execs {
		if e.DisplayName == flow.Alias {
			return e.ID, nil
		}
	}

	return "", ErrNotFound("auth flow not found")
}

func (a GoCloakAdapter) getAuthFlowID(realmName string, flow *KeycloakAuthFlow) (string, error) {
	if flow.ParentName != "" {
		execs, err := a.getFlowExecutions(realmName, flow.ParentName)
		if err != nil {
			return "", errors.Wrap(err, "unable to get auth flow executions")
		}

		for _, e := range execs {
			if e.DisplayName == flow.Alias {
				return e.FlowID, nil
			}
		}

		return "", ErrNotFound("auth flow not found")
	}

	flows, err := a.getRealmAuthFlows(realmName)
	if err != nil {
		return "", errors.Wrap(err, "unable to get realm auth flows")
	}

	for _, fl := range flows {
		if fl.Alias == flow.Alias {
			return fl.ID, nil
		}
	}

	return "", ErrNotFound("auth flow not found")
}

func (a GoCloakAdapter) getRealmAuthFlows(realmName string) ([]KeycloakAuthFlow, error) {
	var flows []KeycloakAuthFlow

	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			"realm": realmName,
		}).
		SetResult(&flows).
		Get(a.basePath + authFlows)

	if err := a.checkError(err, resp); err != nil {
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
			"realm": realmName,
		}).
		SetBody(flow).
		Post(a.basePath + authFlows)

	if err := a.checkError(err, resp); err != nil {
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
			"realm": realmName,
		}).
		SetBody(KeycloakChildAuthFlow{
			Description: flow.Description,
			Alias:       flow.Alias,
			Provider:    flow.ProviderID,
			Type:        flow.ChildType,
		}).
		Post(a.basePath + path.Join(authFlows, flow.ParentName, "executions/flow"))

	if err := a.checkError(err, rsp); err != nil {
		return "", errors.Wrap(err, "unable to create child auth flow in realm")
	}

	id, err := getIDFromResponseLocation(rsp.RawResponse)
	if err != nil {
		return "", errors.Wrap(err, "unable to get flow id")
	}

	return id, nil
}

func (a GoCloakAdapter) updateFlowExecution(realmName, parentFlowAlias string, flowExec *FlowExecution) error {
	rsp, err := a.startRestyRequest().SetPathParams(map[string]string{
		"realm": realmName,
		"alias": parentFlowAlias,
	}).SetBody(flowExec).Put(a.basePath + authFlowExecutionGetUpdate)

	if err := a.checkError(err, rsp); err != nil {
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
				"realm": realmName,
				"id":    executionID,
			}).
			SetBody(map[string]string{
				"realm":     realmName,
				"execution": executionID,
			}).Post(a.basePath + route)

		if err := a.checkError(err, rsp); err != nil {
			return errors.Wrap(err, "unable to adjust execution priority")
		}
	}

	return nil
}

func (a GoCloakAdapter) getFlowExecutions(realmName, flowAlias string) ([]FlowExecution, error) {
	var execs []FlowExecution
	rsp, err := a.startRestyRequest().SetPathParams(map[string]string{
		"realm": realmName,
		"alias": flowAlias,
	}).SetResult(&execs).Get(a.basePath + authFlowExecutionGetUpdate)

	if err := a.checkError(err, rsp); err != nil {
		return nil, errors.Wrap(err, "unable get flow executions")
	}

	return execs, nil
}

func (a GoCloakAdapter) deleteAuthFlow(realmName, ID string) error {
	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			"realm": realmName,
			"id":    ID,
		}).
		Delete(a.basePath + authFlow)

	if err := a.checkError(err, resp); err != nil {
		return errors.Wrap(err, "unable to delete auth flow")
	}

	return nil
}

func (a GoCloakAdapter) addAuthFlowExecution(realmName string, flowExec *AuthenticationExecution) error {
	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			"realm": realmName,
		}).
		SetBody(flowExec).
		Post(a.basePath + authFlowExecutionCreate)

	if err := a.checkError(err, resp); err != nil {
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
			"realm": realmName,
			"id":    flowExec.ID,
		}).
		SetBody(flowExec.AuthenticatorConfig).
		Post(a.basePath + authFlowExecutionConfig)

	if err := a.checkError(err, resp); err != nil {
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

	var replaceFlow *KeycloakAuthFlow
	authFlows, err := a.getRealmAuthFlows(realmName)
	if err != nil {
		return nil, false, errors.Wrapf(err, "unable to get auth flows for realm: %s", realmName)
	}

	for _, f := range authFlows {
		if f.Alias != flowAlias {
			replaceFlow = &f
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
