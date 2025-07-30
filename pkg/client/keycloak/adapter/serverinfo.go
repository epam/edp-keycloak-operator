package adapter

import (
	"context"
	"fmt"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

const (
	FeatureFlagAdminFineGrainedAuthz = "ADMIN_FINE_GRAINED_AUTHZ"
)

func (a GoCloakAdapter) GetServerInfo(ctx context.Context) (dto.ServerInfo, error) {
	var result dto.ServerInfo

	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetResult(&result).
		Get(a.buildPath(serverInfo))

	if err = a.checkError(err, rsp); err != nil {
		return dto.ServerInfo{}, fmt.Errorf("unable to get server info: %w", err)
	}

	return result, nil
}

func (a GoCloakAdapter) FeatureFlagEnabled(ctx context.Context, featureFlag string) (bool, error) {
	info, err := a.GetServerInfo(ctx)
	if err != nil {
		return false, err
	}

	for _, f := range info.Features {
		if f.Name == featureFlag {
			return f.Enabled, nil
		}
	}

	return false, nil
}
