// Package v1 contains API Schema definitions for the v1 API group
// +kubebuilder:object:generate=true
// +groupName=v1.edp.epam.com
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "v1.edp.epam.com", Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	AddToScheme = SchemeBuilder.AddToScheme
)

const (
	// KeycloakRealmKind is a string value of the kind of KeycloakClient CR.
	KeycloakRealmKind = "KeycloakRealm"
	// KeycloakRealmComponentKind is a string value of the kind of KeycloakClient CR.
	KeycloakRealmComponentKind = "KeycloakRealmComponent"
)
