package common

// TLSServerName is a host name or IP address without a port or brackets, or empty.
// The rule uses the CEL IP and format libraries: Kubernetes 1.32 or later.
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:XValidation:rule="size(self) == 0 || isIP(self) || (!format.dns1123Subdomain().validate(self.lowerAscii()).hasValue() && !self.matches('^[0-9.]+$'))",message="must be a host name or IP address without a port or brackets"
type TLSServerName string
