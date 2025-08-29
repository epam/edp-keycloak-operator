package adapter

import (
	"context"
	"fmt"
	"net/http"
)

type Component struct {
	Name         string              `json:"name"`
	ParentID     string              `json:"parentId,omitempty"`
	ProviderID   string              `json:"providerId"`
	ProviderType string              `json:"providerType"`
	Config       map[string][]string `json:"config"`
	ID           string              `json:"id,omitempty"`
}

func (a GoCloakAdapter) CreateComponent(ctx context.Context, realmName string, component *Component) error {
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		keycloakApiParamRealm: realmName,
	}).SetBody(component).Post(a.buildPath(realmComponent))
	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("error during request: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) UpdateComponent(ctx context.Context, realmName string, component *Component) error {
	if component.ID == "" {
		_component, err := a.GetComponent(ctx, realmName, component.Name)
		if err != nil {
			return fmt.Errorf("unable to get component id: %w", err)
		}

		component.ID = _component.ID
	}

	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		keycloakApiParamRealm: realmName,
		keycloakApiParamId:    component.ID,
	}).SetBody(component).Put(a.buildPath(realmComponentEntity))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("error during update component request: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) DeleteComponent(ctx context.Context, realmName, componentName string) error {
	component, err := a.GetComponent(ctx, realmName, componentName)
	if err != nil {
		if IsErrNotFound(err) {
			return nil
		}

		return fmt.Errorf("unable to get component id: %w", err)
	}

	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		keycloakApiParamRealm: realmName,
		keycloakApiParamId:    component.ID,
	}).Delete(a.buildPath(realmComponentEntity))

	if rsp.StatusCode() == http.StatusNotFound {
		return nil
	}

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("error during delete component request: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) GetComponent(ctx context.Context, realmName, componentName string) (*Component, error) {
	var components []Component

	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		keycloakApiParamRealm: realmName,
	}).SetResult(&components).Get(a.buildPath(realmComponent))
	if err = a.checkError(err, rsp); err != nil {
		return nil, fmt.Errorf("error during get component request: %w", err)
	}

	for _, c := range components {
		if c.Name == componentName {
			return &c, nil
		}
	}

	return nil, NotFoundError("component not found")
}
