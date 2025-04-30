package common

import (
	v1 "k8s.io/api/core/v1"
)

// RealmRef is a reference to a Keycloak realm.
// This is new approach to reference Keycloak resources.
// After migration to this approach, we can make Name and Kind required values.
type RealmRef struct {
	// Kind specifies the kind of the Keycloak resource.
	// +kubebuilder:validation:Enum=KeycloakRealm;ClusterKeycloakRealm
	// +kubebuilder:default=KeycloakRealm
	// +optional
	Kind string `json:"kind,omitempty"`

	// Name specifies the name of the Keycloak resource.
	// +required
	Name string `json:"name,omitempty"`
}

// +kubebuilder:object:generate=false
type HasRealmRef interface {
	GetRealmRef() RealmRef
}

// +kubebuilder:object:generate=false
type HasKeycloakRef interface {
	GetKeycloakRef() KeycloakRef
}

// KeycloakRef is a reference to a Keycloak instance.
// This is new approach to reference Keycloak resources.
// After migration to this approach, we can make Name and Kind required values.
type KeycloakRef struct {
	// Kind specifies the kind of the Keycloak resource.
	// +kubebuilder:validation:Enum=Keycloak;ClusterKeycloak
	// +kubebuilder:default=Keycloak
	// +optional
	Kind string `json:"kind,omitempty"`

	// Name specifies the name of the Keycloak resource.
	// +required
	Name string `json:"name,omitempty"`
}

// SourceRef is a reference to a key in a ConfigMap or a Secret.
// +kubebuilder:object:generate=true
type SourceRef struct {
	// Selects a key of a ConfigMap.
	// +optional
	ConfigMapKeyRef *ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`

	// Selects a key of a secret.
	// +optional
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// SourceRefOrVal is reference to a key in a ConfigMap or a Secret or a direct value.
// +kubebuilder:object:generate=true
type SourceRefOrVal struct {
	// Selects a key of a ConfigMap or a Secret.
	SourceRef *SourceRef `json:",inline,omitempty"`

	// Directly specifies a value.
	// +optional
	Value string `json:"value,omitempty"`
}

type ConfigMapKeySelector struct {
	// The ConfigMap to select from.
	v1.LocalObjectReference `json:",inline"`
	// The key to select.
	Key string `json:"key"`
}

type SecretKeySelector struct {
	// The name of the secret.
	v1.LocalObjectReference `json:",inline"`
	// The key of the secret to select from.
	Key string `json:"key"`
}
