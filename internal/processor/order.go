package processor

import (
	"slices"
	"strings"
)

const (
	resourceGroupExternal = "external"
	resourceGroupDirs     = "dirs"
	resourceGroupFiles    = "files"
)

var defaultResourceOrder = []string{
	resourceGroupExternal,
	resourceGroupDirs,
	resourceGroupFiles,
}

// DefaultResourceOrder returns the built-in resource ordering.
func DefaultResourceOrder() []string {
	return slices.Clone(defaultResourceOrder)
}

// ParseResourceOrder builds a resource group order from the provided CSV, appending missing groups.
func ParseResourceOrder(value string) []string {
	if strings.TrimSpace(value) == "" {
		return DefaultResourceOrder()
	}
	return normalizeResourceOrder(strings.Split(value, ","))
}

// normalizeResourceOrder normalizes the provided resource ordering.
func normalizeResourceOrder(parts []string) []string {
	if len(parts) == 0 {
		return DefaultResourceOrder()
	}

	out := make([]string, 0, len(defaultResourceOrder))
	seen := make(map[string]struct{}, len(defaultResourceOrder))

	// Parse the provided value and add each group.
	for _, part := range parts {
		group := strings.ToLower(strings.TrimSpace(part))
		if group == "" {
			continue
		}
		switch group {
		case resourceGroupExternal, resourceGroupDirs, resourceGroupFiles:
		default:
			continue
		}
		if _, ok := seen[group]; ok {
			continue
		}
		seen[group] = struct{}{}
		out = append(out, group)
	}

	// Add missing groups at the end.
	for _, group := range defaultResourceOrder {
		if _, ok := seen[group]; ok {
			continue
		}
		out = append(out, group)
	}

	return out
}
