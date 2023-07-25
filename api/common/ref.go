package common

// RealmRef is a reference to a Keycloak realm.
// This is new approach to reference Keycloak resources.
// After migration to this approach, we can make Name and Kind required values.
type RealmRef struct {
	// Kind specifies the kind of the Keycloak resource.
	// +kubebuilder:validation:Enum=KeycloakRealm;ClusterKeycloakRealm
	// +optional
	Kind string `json:"kind,omitempty"`

	// Name specifies the name of the Keycloak resource.
	// +optional
	Name string `json:"name,omitempty"`
}

type HasRealmRef interface {
	GetRealmRef() RealmRef
}

type HasKeycloakRef interface {
	GetKeycloakRef() KeycloakRef
}

// KeycloakRef is a reference to a Keycloak instance.
// This is new approach to reference Keycloak resources.
// After migration to this approach, we can make Name and Kind required values.
type KeycloakRef struct {
	// Kind specifies the kind of the Keycloak resource.
	// +kubebuilder:validation:Enum=Keycloak;ClusterKeycloak
	// +optional
	Kind string `json:"kind,omitempty"`

	// Name specifies the name of the Keycloak resource.
	// +optional
	Name string `json:"name,omitempty"`
}
