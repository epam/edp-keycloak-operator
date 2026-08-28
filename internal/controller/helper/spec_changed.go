package helper

// Update endpoints PUT the full representation: a spec edit must force the
// write so keys removed from the spec are dropped in Keycloak.
func SpecChanged(generation, observedGeneration int64) bool {
	return generation != observedGeneration
}
