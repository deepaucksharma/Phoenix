// Package main provides a policy validation tool
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// validatePolicy performs structure validation on policy files
func validatePolicy(filePath string) error {
	// Get file info to check if it's empty
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("error accessing file: %w", err)
	}

	// Skip validation for empty files
	if info.Size() == 0 {
		fmt.Printf("⚠️ Policy file %s is empty\n", filepath.Base(filePath))
		return nil
	}

	// Read policy file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Handle empty YAML files (only whitespace or comments)
	if len(data) == 0 || isEmptyYAML(string(data)) {
		fmt.Printf("⚠️ Policy file %s is effectively empty\n", filepath.Base(filePath))
		return nil
	}

	// Parse YAML to verify it's valid
	var policyMap map[string]interface{}
	if err := yaml.Unmarshal(data, &policyMap); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Check for nil map (valid YAML but empty document)
	if policyMap == nil {
		fmt.Printf("⚠️ Policy file %s contains an empty YAML document\n", filepath.Base(filePath))
		return nil
	}

	// Check required top-level sections for non-test/example files
	if !isTestOrExample(filePath) {
		requiredSections := []string{"controllers", "processors", "safety_limits"}
		for _, section := range requiredSections {
			if _, ok := policyMap[section]; !ok {
				return fmt.Errorf("missing required section: %s", section)
			}
		}

		// Validate PID controllers if present
		if controllers, ok := policyMap["controllers"].(map[string]interface{}); ok {
			for name, ctrl := range controllers {
				controller, ok := ctrl.(map[string]interface{})
				if !ok {
					return fmt.Errorf("controller %s has invalid structure", name)
				}

				// Check controller fields
				requiredFields := []string{"enabled", "kpi_metric_name", "kpi_target_value"}
				for _, field := range requiredFields {
					if _, ok := controller[field]; !ok {
						return fmt.Errorf("controller %s is missing required field: %s", name, field)
					}
				}
			}
		}
	}

	fmt.Printf("✅ Policy file %s is valid\n", filepath.Base(filePath))
	return nil
}

// isEmptyYAML checks if a YAML string is effectively empty (only whitespace or comments)
func isEmptyYAML(content string) bool {
	for _, line := range []rune(content) {
		if line == '#' {
			continue // Skip comments
		}
		if line != ' ' && line != '\t' && line != '\n' && line != '\r' {
			return false
		}
	}
	return true
}

// isTestOrExample checks if a file is a test or example file
// These files may have simplified structure for testing
func isTestOrExample(filePath string) bool {
	path := filepath.ToSlash(filePath)
	return filepath.Base(path) == "example.yaml" ||
		filepath.Base(path) == "test.yaml" ||
		filepath.Base(path) == "sample.yaml" ||
		containsSubstring(path, "test/") ||
		containsSubstring(path, "example/") ||
		containsSubstring(path, "testing/")
}

// containsSubstring checks if a string contains another string
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: validate_policy <policy_file_path>")
		os.Exit(1)
	}

	policyFile := os.Args[1]
	if err := validatePolicy(policyFile); err != nil {
		fmt.Printf("❌ Validation failed: %s\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}