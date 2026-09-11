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

// isManagedLocalResource reports whether Karma owns a direct local resource
// entry. Paths that point outside or below the current directory are preserved
// as opaque user-managed references.
func isManagedLocalResource(entry string) bool {
	if entry == "" || isRemoteResource(entry) {
		return false
	}

	entry = strings.TrimPrefix(entry, "./")
	entry = strings.TrimSuffix(entry, "/")
	if entry == "" || entry == ".." || strings.HasPrefix(entry, "../") {
		return false
	}
	return !strings.ContainsAny(entry, `/\`)
}
