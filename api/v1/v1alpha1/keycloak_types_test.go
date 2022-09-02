package v1alpha1

import "testing"

func TestKeycloak_GetAdminType(t *testing.T) {
	kc := Keycloak{}
	if kc.GetAdminType() != KeycloakAdminTypeUser {
		t.Fatal("wrong admin type returned")
	}

	kc.Spec.AdminType = KeycloakAdminTypeServiceAccount
	if kc.GetAdminType() != "serviceAccount" {
		t.Fatal("wring admin type returned")
	}
}
