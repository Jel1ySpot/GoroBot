package util

// CoalesceString returns the first non-empty string from the arguments
// If primary is empty or only whitespace, returns fallback
func CoalesceString(primary any, fallback string) string {
	if str, ok := primary.(string); ok && str != "" {
		return str
	}
	return fallback
}
