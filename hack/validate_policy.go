// Package main provides a policy validation tool
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// BasicPolicyValidation performs simple structure validation on policy files
func validatePolicy(filePath string) error {
	// Read policy file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Parse YAML to verify it's valid
	var policyMap map[string]interface{}
	if err := yaml.Unmarshal(data, &policyMap); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Check required top-level sections
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

	fmt.Printf("✅ Policy file %s is valid\n", filepath.Base(filePath))
	return nil
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