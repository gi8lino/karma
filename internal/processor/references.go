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

// referencedLocalEntries returns direct files or directories used by known
// Kustomization path fields outside resources. Keeping this schema-aware avoids
// treating arbitrary YAML-looking strings (for example annotations) as paths.
func referencedLocalEntries(kustomizationPath string) (map[string]struct{}, error) {
	entries := make(map[string]struct{})
	if kustomizationPath == "" {
		return entries, nil
	}

	data, err := os.ReadFile(kustomizationPath)
	if errors.Is(err, os.ErrNotExist) {
		return entries, nil
	}
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return entries, nil
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
		if key.Kind != yaml.ScalarNode {
			continue
		}

		switch key.Value {
		case "bases", "components", "configurations", "crds", "generators", "transformers", "validators", "patchesStrategicMerge":
			collectPathSequence(value, entries)
		case "patches", "patchesJson6902", "replacements":
			collectMappingPathSequence(value, "path", entries)
		case "openapi":
			collectMappingPath(value, "path", entries)
		case "helmCharts":
			collectMappingPathSequence(value, "valuesFile", entries)
			collectMappingPathSequence(value, "additionalValuesFiles", entries)
		case "helmChartInflationGenerator":
			collectMappingPathSequence(value, "values", entries)
		case "configMapGenerator", "secretGenerator":
			collectGeneratorPaths(value, entries)
		}
	}
	return entries, nil
}

func collectPathSequence(node *yaml.Node, entries map[string]struct{}) {
	if node == nil || node.Kind != yaml.SequenceNode {
		return
	}
	for _, item := range node.Content {
		if item.Kind == yaml.ScalarNode {
			addDirectLocalPath(entries, item.Value)
		}
	}
}

func collectMappingPathSequence(node *yaml.Node, field string, entries map[string]struct{}) {
	if node == nil || node.Kind != yaml.SequenceNode {
		return
	}
	for _, item := range node.Content {
		if item.Kind == yaml.ScalarNode && field == "path" {
			addDirectLocalPath(entries, item.Value)
			continue
		}
		collectMappingPath(item, field, entries)
	}
}

func collectMappingPath(node *yaml.Node, field string, entries map[string]struct{}) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if key.Kind != yaml.ScalarNode || key.Value != field {
			continue
		}
		switch value.Kind {
		case yaml.ScalarNode:
			addDirectLocalPath(entries, value.Value)
		case yaml.SequenceNode:
			for _, item := range value.Content {
				if item.Kind == yaml.ScalarNode {
					addDirectLocalPath(entries, item.Value)
				}
			}
		}
	}
}

func collectGeneratorPaths(node *yaml.Node, entries map[string]struct{}) {
	if node == nil || node.Kind != yaml.SequenceNode {
		return
	}
	for _, item := range node.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		collectMappingPath(item, "env", entries)
		collectMappingPath(item, "envs", entries)
		for i := 0; i+1 < len(item.Content); i += 2 {
			key, value := item.Content[i], item.Content[i+1]
			if key.Kind != yaml.ScalarNode || key.Value != "files" || value.Kind != yaml.SequenceNode {
				continue
			}
			for _, source := range value.Content {
				if source.Kind != yaml.ScalarNode {
					continue
				}
				path := source.Value
				if _, file, ok := strings.Cut(path, "="); ok {
					path = file
				}
				addDirectLocalPath(entries, path)
			}
		}
	}
}

func addDirectLocalPath(entries map[string]struct{}, value string) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "./")
	if value == "" || filepath.IsAbs(value) || isRemoteResource(value) {
		return
	}
	if filepath.Base(value) != value {
		return
	}
	entries[value] = struct{}{}
}
