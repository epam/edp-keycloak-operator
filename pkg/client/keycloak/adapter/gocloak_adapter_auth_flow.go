package adapter

import (
	"context"
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
}

type AuthenticatorConfig struct {
	Alias  string            `json:"alias"`
	Config map[string]string `json:"config"`
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

	if _, _, err := a.unsetBrowserFlow(realmName, alias); err != nil {
		return errors.Wrapf(err, "unable to unset browser flow for realm: %s, alias: %s", realmName, alias)
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

func (a GoCloakAdapter) syncBaseAuthFlow(realmName string, flow *KeycloakAuthFlow) (id string, err error) {
	var (
		realm              *gocloak.RealmRepresentation
		isBrowserFlowUnset bool
	)

	authFlow, err := a.getAuthFlow(realmName, flow.Alias)
	if err != nil && !IsErrNotFound(errors.Cause(err)) {
		return "", errors.Wrap(err, "unable to get auth flow")
	} else if err == nil {
		realm, isBrowserFlowUnset, err = a.unsetBrowserFlow(realmName, flow.Alias)
		if err != nil {
			return "", errors.Wrapf(err, "unable to check if alias [%s] is set for browser flow in realm [%s]",
				flow.Alias, realmName)
		}

		if err := a.deleteAuthFlow(realmName, authFlow.ID); err != nil {
			return "", errors.Wrap(err, "unable to delete auth flow")
		}
	}

	id, err = a.createAuthFlow(realmName, flow)
	if err != nil {
		return "", errors.Wrap(err, "unable to create auth flow")
	}

	if isBrowserFlowUnset {
		realm.BrowserFlow = &flow.Alias
		if err := a.client.UpdateRealm(context.Background(), a.token.AccessToken, *realm); err != nil {
			return "", errors.Wrapf(err, "unable to set back auth flow [%s] as browser flow for realm [%s]",
				flow.Alias, realmName)
		}
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

	if err := a.checkError(err, resp); err != nil {
		return nil, errors.Wrap(err, "unable to list auth flow by realm")
	}

	return flows, nil
}

func (a GoCloakAdapter) createAuthFlow(realmName string, flow *KeycloakAuthFlow) (id string, err error) {
	var (
		body       interface{} = flow
		requestURL             = a.basePath + authFlows
	)

	if flow.ParentName != "" {
		requestURL = a.basePath + path.Join(authFlows, flow.ParentName, "executions/flow")
		body = KeycloakChildAuthFlow{
			Description: flow.Description,
			Alias:       flow.Alias,
			Provider:    flow.ProviderID,
			Type:        flow.ChildType,
		}
	}

	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			"realm": realmName,
		}).
		SetBody(body).
		Post(requestURL)

	if err := a.checkError(err, resp); err != nil {
		return "", errors.Wrap(err, "unable to create auth flow in realm")
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
