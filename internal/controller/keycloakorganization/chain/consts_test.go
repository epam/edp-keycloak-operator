package chain

import keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"

const (
	testRealmName = "test-realm"
	testOrgID     = "org-123"
	testIdPAlias  = "idp1"
	testDomain    = "a.com"
)

func domainsV2(names ...string) []keycloakApi.OrgDomain {
	domains := make([]keycloakApi.OrgDomain, 0, len(names))
	for _, name := range names {
		domains = append(domains, keycloakApi.OrgDomain{Name: name})
	}

	return domains
}
