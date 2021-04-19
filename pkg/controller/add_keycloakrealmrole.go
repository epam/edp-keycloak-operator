package controller

import "github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealmrole"

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, keycloakrealmrole.Add)
}
