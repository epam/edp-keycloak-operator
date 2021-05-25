package controller

import (
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealmgroup"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, keycloakrealmgroup.Add)
}
