package processor

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// referencedYAMLFiles returns direct YAML files used outside the resources
// field, for example patches, transformer configuration, or Helm values.
func referencedYAMLFiles(kustomizationPath string) (map[string]struct{}, error) {
	files := make(map[string]struct{})
	if kustomizationPath == "" {
		return files, nil
	}

	data, err := os.ReadFile(kustomizationPath)
	if errors.Is(err, os.ErrNotExist) {
		return files, nil
	}
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return files, nil
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("decode %s: %w", kustomizationPath, err)
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) != 1 || root.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: kustomization root must be a YAML mapping", kustomizationPath)
	}

	mapping := root.Content[0]
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i], mapping.Content[i+1]
		if key.Kind == yaml.ScalarNode && key.Value == "resources" {
			continue
		}
		collectDirectYAMLReferences(value, files)
	}
	return files, nil
}

func collectDirectYAMLReferences(node *yaml.Node, files map[string]struct{}) {
	if node == nil {
		return
	}
	if node.Kind == yaml.ScalarNode {
		value := strings.TrimPrefix(node.Value, "./")
		if isYAML(value) && filepath.Base(value) == value {
			files[value] = struct{}{}
		}
		return
	}
	for _, child := range node.Content {
		collectDirectYAMLReferences(child, files)
	}
}
