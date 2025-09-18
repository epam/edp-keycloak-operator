package controller

import (
	"github.com/epmd-edp/keycloak-operator/pkg/controller/keycloakrealm"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, keycloakrealm.Add)
}
