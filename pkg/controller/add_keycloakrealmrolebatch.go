package controller

import (
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealmrolebatch"
)

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, keycloakrealmrolebatch.Add)
}
