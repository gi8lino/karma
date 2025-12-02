package processor

import "strings"

const (
	resourceGroupRemote = "remote"
	resourceGroupDirs   = "dirs"
	resourceGroupFiles  = "files"
)

var defaultResourceOrder = []string{
	resourceGroupRemote,
	resourceGroupDirs,
	resourceGroupFiles,
}

// DefaultResourceOrder returns the built-in resource ordering.
func DefaultResourceOrder() []string {
	out := make([]string, len(defaultResourceOrder))
	copy(out, defaultResourceOrder)
	return out
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

	seen := map[string]struct{}{}                       // map for uniqueness
	out := make([]string, 0, len(defaultResourceOrder)) // slice to keep order

	// parse the provided value and add each group.
	for _, part := range parts {
		group := strings.ToLower(strings.TrimSpace(part))
		if group == "" {
			continue
		}
		switch group {
		case resourceGroupRemote, resourceGroupDirs, resourceGroupFiles:
		default:
			continue
		}
		if _, ok := seen[group]; ok {
			continue
		}
		seen[group] = struct{}{}
		out = append(out, group)
	}

	// add missing groups at the end.
	for _, group := range defaultResourceOrder {
		if _, ok := seen[group]; ok {
			continue
		}
		out = append(out, group)
	}

	if len(out) == 0 {
		return DefaultResourceOrder()
	}

	return out
}
