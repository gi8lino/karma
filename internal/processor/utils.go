package processor

import "strings"

// isKustomization reports whether name is a recognized kustomization file name.
func isKustomization(name string) bool {
	return name == "kustomization.yaml" || name == "kustomization.yml"
}

// isYAML returns true when the file name has a YAML extension.
func isYAML(name string) bool {
	lowered := strings.ToLower(name)
	return strings.HasSuffix(lowered, ".yaml") || strings.HasSuffix(lowered, ".yml")
}

// isRemoteResource returns true for HTTP(S) resource references.
func isRemoteResource(entry string) bool {
	return strings.HasPrefix(entry, "http://") || strings.HasPrefix(entry, "https://")
}

// equalStrings reports whether two string slices are identical.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
