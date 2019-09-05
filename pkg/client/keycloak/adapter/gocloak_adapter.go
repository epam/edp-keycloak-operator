package adapter

import (
	"gopkg.in/nerzal/gocloak.v2"
	"keycloak-operator/pkg/apis/v1/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
)

var log = logf.Log.WithName("gocloak_adapter")

type GoCloakAdapter struct {
	client gocloak.GoCloak
	token  gocloak.JWT
}

func (a GoCloakAdapter) ExistRealm(spec v1alpha1.KeycloakRealmSpec) (*bool, error) {
	reqLog := log.WithValues("realm spec", spec)
	reqLog.Info("Start check existing realm...")

	_, err := a.client.GetRealm(a.token.AccessToken, spec.RealmName)

	res, err := strip404(err)

	if err != nil {
		return nil, err
	}

	reqLog.Info("Check existing realm has been finished", "result", res)
	return &res, nil
}

func strip404(in error) (bool, error) {
	if in == nil {
		return true, nil
	}
	if is404(in) {
		return false, nil
	}
	return false, in
}

func is404(e error) bool {
	return strings.Contains(e.Error(), "404")
}

func (a GoCloakAdapter) CreateRealmWithDefaultConfig(spec v1alpha1.KeycloakRealmSpec) error {
	reqLog := log.WithValues("realm spec", spec)
	reqLog.Info("Start creating realm with default config...")

	err := a.client.CreateRealm(a.token.AccessToken, getDefaultRealm(spec.RealmName))
	if err != nil {
		return err
	}

	reqLog.Info("End creating realm with default config")
	return nil
}

func getDefaultRealm(realmName string) gocloak.RealmRepresentation {
	return gocloak.RealmRepresentation{
		Realm:        realmName,
		Enabled:      true,
		DefaultRoles: []string{"developer"},
		Roles: map[string][]map[string]interface{}{
			"realm": {
				{
					"name": "administrator",
				},
				{
					"name": "developer",
				},
			},
		},
	}
}
