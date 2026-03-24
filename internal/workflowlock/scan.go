package workflowlock

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// DiscoverRefs scans workflow YAML files and returns supported remote uses references.
func DiscoverRefs(workflowDir string) ([]DiscoveredRef, error) {
	return DiscoverRefsForHost(workflowDir, "github.com")
}

// DiscoverRefsForHost scans workflow YAML files and applies the provided default
// host to plain owner/repo references.
func DiscoverRefsForHost(workflowDir, defaultHost string) ([]DiscoveredRef, error) {
	matches, err := filepath.Glob(filepath.Join(workflowDir, "*.y*ml"))
	if err != nil {
		return nil, fmt.Errorf("discover workflows: %w", err)
	}
	sort.Strings(matches)

	var refs []DiscoveredRef
	for _, path := range matches {
		fileRefs, err := discoverRefsInFile(path, defaultHost)
		if err != nil {
			return nil, err
		}
		refs = append(refs, fileRefs...)
	}
	return refs, nil
}

func discoverRefsInFile(path, defaultHost string) ([]DiscoveredRef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workflow %s: %w", path, err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse workflow %s: %w", path, err)
	}

	var refs []DiscoveredRef
	var walk func(*yaml.Node) error
	walk = func(node *yaml.Node) error {
		if node == nil {
			return nil
		}
		if node.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(node.Content); i += 2 {
				key := node.Content[i]
				value := node.Content[i+1]
				if key.Value == "uses" && value.Kind == yaml.ScalarNode {
					normalized, ignored, err := NormalizeRefForHost(value.Value, defaultHost)
					if err != nil {
						return fmt.Errorf("%s:%d: %w", path, value.Line, err)
					}
					if !ignored {
						refs = append(refs, DiscoveredRef{
							File:       path,
							Line:       value.Line,
							Raw:        strings.TrimSpace(value.Value),
							Normalized: normalized,
						})
					}
				}
				if err := walk(value); err != nil {
					return err
				}
			}
			return nil
		}
		for _, child := range node.Content {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}

	if err := walk(&root); err != nil {
		return nil, err
	}

	return refs, nil
}
