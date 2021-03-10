package controller

import (
	"github.com/epmd-edp/keycloak-operator/pkg/controller/keycloakrealmrolebatch"
)

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, keycloakrealmrolebatch.Add)
}
