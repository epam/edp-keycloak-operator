package adapter

import (
	"net/http"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

type KeycloakAuthFlow struct {
	ID                       string                    `json:"id,omitempty"`
	Alias                    string                    `json:"alias"`
	Description              string                    `json:"description"`
	ProviderID               string                    `json:"providerId"`
	TopLevel                 bool                      `json:"topLevel"`
	BuiltIn                  bool                      `json:"builtIn"`
	AuthenticationExecutions []AuthenticationExecution `json:"-"`
}

type AuthenticationExecution struct {
	Authenticator       string               `json:"authenticator"`
	Requirement         string               `json:"requirement"`
	Priority            int                  `json:"priority"`
	ParentFlow          string               `json:"parentFlow"`
	AuthenticatorConfig *AuthenticatorConfig `json:"-"`
	AutheticatorFlow    bool                 `json:"autheticatorFlow"`
	ID                  string               `json:"-"`
}

type AuthenticatorConfig struct {
	Alias  string                 `json:"alias"`
	Config map[string]interface{} `json:"config"`
}

type orderByPriority []AuthenticationExecution

func (a orderByPriority) Len() int           { return len(a) }
func (a orderByPriority) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a orderByPriority) Less(i, j int) bool { return a[i].Priority < a[j].Priority }

func (a GoCloakAdapter) DeleteAuthFlow(realmName, alias string) error {
	flow, err := a.getAuthFlow(realmName, alias)
	if err != nil {
		return errors.Wrap(err, "unable to get auth flow")
	}

	if err := a.deleteAuthFlow(realmName, flow.ID); err != nil {
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
		e.ParentFlow = id
		if err := a.addAuthFlowExecution(realmName, &e); err != nil {
			return errors.Wrap(err, "unable to add auth execution")
		}
	}

	return nil
}

func (a GoCloakAdapter) syncBaseAuthFlow(realmName string, flow *KeycloakAuthFlow) (id string, err error) {
	authFlow, err := a.getAuthFlow(realmName, flow.Alias)
	if err != nil && !isErrNotFound(errors.Cause(err)) {
		return "", errors.Wrap(err, "unable to get auth flow")
	} else if err == nil {
		if err := a.deleteAuthFlow(realmName, authFlow.ID); err != nil {
			return "", errors.Wrap(err, "unable to delete auth flow")
		}
	}

	id, err = a.createAuthFlow(realmName, flow)
	if err != nil {
		return "", errors.Wrap(err, "unable to create auth flow")
	}

	return id, nil
}

func (a GoCloakAdapter) getAuthFlow(realmName, authFlowAlias string) (*KeycloakAuthFlow, error) {
	flows, err := a.getRealmAuthFlows(realmName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get realm auth flows")
	}

	for _, fl := range flows {
		if fl.Alias == authFlowAlias {
			return &fl, nil
		}
	}

	return nil, ErrNotFound("auth flow not found")
}

func (a GoCloakAdapter) getRealmAuthFlows(realmName string) ([]KeycloakAuthFlow, error) {
	var flows []KeycloakAuthFlow

	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			"realm": realmName,
		}).
		SetResult(&flows).
		Get(a.basePath + authFlows)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list auth flow by realm")
	}

	if err := extractError(resp); err != nil {
		return nil, err
	}

	return flows, nil
}

func (a GoCloakAdapter) createAuthFlow(realmName string, flow *KeycloakAuthFlow) (id string, err error) {
	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			"realm": realmName,
		}).
		SetBody(flow).
		Post(a.basePath + authFlows)
	if err != nil {
		return "", errors.Wrap(err, "unable to create auth flow in realm")
	}

	if err := extractError(resp); err != nil {
		return "", err
	}

	id, err = getIDFromResponseLocation(resp.RawResponse)
	if err != nil {
		return "", errors.Wrap(err, "unable to get flow id")
	}

	return
}

func (a GoCloakAdapter) deleteAuthFlow(realmName, ID string) error {
	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			"realm": realmName,
			"id":    ID,
		}).
		Delete(a.basePath + authFlow)
	if err != nil {
		return errors.Wrap(err, "unable to delete auth flow")
	}

	return extractError(resp)
}

func (a GoCloakAdapter) addAuthFlowExecution(realmName string, flowExec *AuthenticationExecution) error {
	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			"realm": realmName,
		}).
		SetBody(flowExec).
		Post(a.basePath + authFlowExecutionCreate)
	if err != nil {
		return errors.Wrap(err, "unable to add auth flow execution")
	}

	if err := extractError(resp); err != nil {
		return err
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
	if err != nil {
		return errors.Wrap(err, "unable to add auth flow execution")
	}

	return extractError(resp)
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
