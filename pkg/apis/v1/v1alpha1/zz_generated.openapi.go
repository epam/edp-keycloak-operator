// +build !ignore_autogenerated

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1alpha1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.Keycloak":             schema_pkg_apis_v1_v1alpha1_Keycloak(ref),
		"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakClient":       schema_pkg_apis_v1_v1alpha1_KeycloakClient(ref),
		"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakClientSpec":   schema_pkg_apis_v1_v1alpha1_KeycloakClientSpec(ref),
		"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakClientStatus": schema_pkg_apis_v1_v1alpha1_KeycloakClientStatus(ref),
		"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakRealm":        schema_pkg_apis_v1_v1alpha1_KeycloakRealm(ref),
		"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakRealmSpec":    schema_pkg_apis_v1_v1alpha1_KeycloakRealmSpec(ref),
		"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakRealmStatus":  schema_pkg_apis_v1_v1alpha1_KeycloakRealmStatus(ref),
		"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakSpec":         schema_pkg_apis_v1_v1alpha1_KeycloakSpec(ref),
		"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakStatus":       schema_pkg_apis_v1_v1alpha1_KeycloakStatus(ref),
	}
}

func schema_pkg_apis_v1_v1alpha1_Keycloak(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "Keycloak is the Schema for the keycloaks API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta", "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakSpec", "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakStatus"},
	}
}

func schema_pkg_apis_v1_v1alpha1_KeycloakClient(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KeycloakClient is the Schema for the keycloakclients API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakClientSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakClientStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta", "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakClientSpec", "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakClientStatus"},
	}
}

func schema_pkg_apis_v1_v1alpha1_KeycloakClientSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KeycloakClientSpec defines the desired state of KeycloakClient",
				Properties:  map[string]spec.Schema{},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_v1_v1alpha1_KeycloakClientStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KeycloakClientStatus defines the observed state of KeycloakClient",
				Properties:  map[string]spec.Schema{},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_v1_v1alpha1_KeycloakRealm(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KeycloakRealm is the Schema for the keycloakrealms API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakRealmSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakRealmStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta", "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakRealmSpec", "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1.KeycloakRealmStatus"},
	}
}

func schema_pkg_apis_v1_v1alpha1_KeycloakRealmSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KeycloakRealmSpec defines the desired state of KeycloakRealm",
				Properties: map[string]spec.Schema{
					"realmName": {
						SchemaProps: spec.SchemaProps{
							Description: "INSERT ADDITIONAL SPEC FIELDS - desired state of cluster Important: Run \"operator-sdk generate k8s\" to regenerate code after modifying this file Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html",
							Type:        []string{"string"},
							Format:      "",
						},
					},
				},
				Required: []string{"realmName"},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_v1_v1alpha1_KeycloakRealmStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KeycloakRealmStatus defines the observed state of KeycloakRealm",
				Properties: map[string]spec.Schema{
					"available": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"boolean"},
							Format: "",
						},
					},
				},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_v1_v1alpha1_KeycloakSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KeycloakSpec defines the desired state of Keycloak",
				Properties: map[string]spec.Schema{
					"url": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"user": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"pwd": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
				Required: []string{"url", "user", "pwd"},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_v1_v1alpha1_KeycloakStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KeycloakStatus defines the observed state of Keycloak",
				Properties: map[string]spec.Schema{
					"connected": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"boolean"},
							Format: "",
						},
					},
				},
				Required: []string{"connected"},
			},
		},
		Dependencies: []string{},
	}
}
