package adapter

import (
	"context"

	"github.com/pkg/errors"
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
		"realm": realmName,
	}).SetBody(component).Post(a.basePath + realmComponent)
	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "error during request")
	}

	return nil
}

func (a GoCloakAdapter) UpdateComponent(ctx context.Context, realmName string, component *Component) error {
	if component.ID == "" {
		_component, err := a.GetComponent(ctx, realmName, component.Name)
		if err != nil {
			return errors.Wrap(err, "unable to get component id")
		}
		component.ID = _component.ID
	}

	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm": realmName,
		"id":    component.ID,
	}).SetBody(component).Put(a.basePath + realmComponentEntity)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "error during request")
	}

	return nil
}

func (a GoCloakAdapter) DeleteComponent(ctx context.Context, realmName, componentName string) error {
	component, err := a.GetComponent(ctx, realmName, componentName)
	if err != nil {
		return errors.Wrap(err, "unable to get component id")
	}

	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm": realmName,
		"id":    component.ID,
	}).Delete(a.basePath + realmComponentEntity)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "error during request")
	}

	return nil
}

func (a GoCloakAdapter) GetComponent(ctx context.Context, realmName, componentName string) (*Component, error) {
	var components []Component

	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm": realmName,
	}).SetResult(&components).Get(a.basePath + realmComponent)
	if err := a.checkError(err, rsp); err != nil {
		return nil, errors.Wrap(err, "error during request")
	}

	for _, c := range components {
		if c.Name == componentName {
			return &c, nil
		}
	}

	return nil, ErrNotFound("component not found")
}
