package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

// ClusterKeycloakSpec defines the desired state of ClusterKeycloak.
// +kubebuilder:validation:XValidation:rule="!has(self.tlsServerName) || size(self.tlsServerName) == 0 || self.url.lowerAscii().startsWith('https://')",message="tlsServerName requires an https url"
type ClusterKeycloakSpec struct {
	// URL of keycloak service.
	Url string `json:"url"`

	// Secret is a secret name which contains admin credentials.
	// Deprecated: Use Auth instead. When Auth is set, this field is ignored.
	// +optional
	Secret string `json:"secret,omitempty"`

	// AdminType can be user or serviceAccount, if serviceAccount was specified,
	// then client_credentials grant type should be used for getting admin realm token.
	// Deprecated: Use Auth instead. When Auth is set, this field is ignored.
	// +optional
	// +kubebuilder:validation:Enum=serviceAccount;user
	// +kubebuilder:default=user
	AdminType string `json:"adminType,omitempty"`

	// Auth defines the authentication configuration for connecting to Keycloak.
	// When set, Secret and AdminType fields are ignored.
	// +optional
	Auth *common.AuthSpec `json:"auth,omitempty"`

	// CACert defines the root certificate authority
	// that api clients use when verifying server certificates.
	// Resources should be in the namespace defined in operator OPERATOR_NAMESPACE env.
	// +optional
	CACert *common.SourceRef `json:"caCert,omitempty"`

	// InsecureSkipVerify controls whether api client verifies the server's
	// certificate chain and host name. If InsecureSkipVerify is true, api client
	// accepts any certificate presented by the server and any host name in that
	// certificate.
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// TLSServerName is the name the server certificate is verified against instead of the Url host,
	// for example when Url is the in-cluster Service. A host name is also sent as SNI; an IP address is not.
	// The connection address and the Host header are unchanged. With InsecureSkipVerify the certificate is not
	// verified, but a host name is still sent as SNI.
	// Must be listed in allowedDestinationHosts when that list is set.
	// +optional
	// +kubebuilder:example=keycloak.example.com
	TLSServerName common.TLSServerName `json:"tlsServerName,omitempty"`
}

func (in *ClusterKeycloak) GetAdminType() string {
	if in.Spec.AdminType == "" {
		in.Spec.AdminType = keycloakApi.KeycloakAdminTypeUser
	}

	return in.Spec.AdminType
}

// ClusterKeycloakStatus defines the observed state of ClusterKeycloak.
type ClusterKeycloakStatus struct {
	// Connected shows if keycloak service is up and running.
	Connected bool `json:"connected"`

	// Value is the status message: OK, or the error of the last connection attempt.
	// +optional
	Value string `json:"value,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Connected",type="boolean",JSONPath=".status.connected",description="Is connected to keycloak"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.value"

// ClusterKeycloak is the Schema for the clusterkeycloaks API.
type ClusterKeycloak struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterKeycloakSpec `json:"spec,omitempty"`

	// +kubebuilder:default={connected:false}
	Status ClusterKeycloakStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterKeycloakList contains a list of ClusterKeycloak.
type ClusterKeycloakList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterKeycloak `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterKeycloak{}, &ClusterKeycloakList{})
}
