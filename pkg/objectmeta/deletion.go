package objectmeta

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const PreserveResourcesOnDeletionAnnotation = "edp.epam.com/preserve-resources-on-deletion"
const PreserveClientSecretAnnotation = "edp.epam.com/preserve-client-secret"

// PreserveResourcesOnDeletion returns true if the object has annotation
// that indicates that resources must not be deleted.
func PreserveResourcesOnDeletion(object metav1.Object) bool {
	return object.GetAnnotations()[PreserveResourcesOnDeletionAnnotation] == "true"
}

// PreserveClientSecret returns true if the object has annotation
// that indicates that the Keycloak client secret must not be changed.
func PreserveClientSecret(object metav1.Object) bool {
	return object.GetAnnotations()[PreserveClientSecretAnnotation] == "true"
}
