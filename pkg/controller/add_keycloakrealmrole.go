package controller

import "github.com/epmd-edp/keycloak-operator/pkg/controller/keycloakrealmrole"

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, keycloakrealmrole.Add)
}
