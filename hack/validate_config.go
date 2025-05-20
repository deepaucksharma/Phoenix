package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run hack/validate_config.go [config_file]")
		os.Exit(1)
	}

	configFile := os.Args[1]
	schemaDir := "schemas"

	// Determine schema based on file name
	var schemaFile string
	baseName := filepath.Base(configFile)
	
	if strings.Contains(baseName, "policy") {
		schemaFile = filepath.Join(schemaDir, "policy_schema.json")
	} else {
		schemaFile = filepath.Join(schemaDir, "config_schema.json")
	}

	// Check if schema file exists
	if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
		fmt.Printf("Schema file %s does not exist. Running generator...\n", schemaFile)
		os.MkdirAll(schemaDir, 0755)
		
		if strings.Contains(baseName, "policy") {
			// Generate policy schema
			generatePolicySchema(schemaDir)
		} else {
			// Generate config schema
			runConfigSchemaGenerator(schemaDir)
		}
	}

	// Load and validate configuration
	err := validateConfig(configFile, schemaFile)
	if err != nil {
		fmt.Printf("Validation error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration file %s is valid.\n", configFile)
}

func validateConfig(configFile, schemaFile string) error {
	// Read config file
	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	// Read schema file
	schemaData, err := ioutil.ReadFile(schemaFile)
	if err != nil {
		return fmt.Errorf("error reading schema file: %w", err)
	}

	// Parse YAML config into JSON for validation
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(configData, &configMap); err != nil {
		return fmt.Errorf("error parsing config YAML: %w", err)
	}

	// Parse schema
	schemaLoader := gojsonschema.NewBytesLoader(schemaData)
	documentLoader := gojsonschema.NewGoLoader(configMap)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		errStr := "config validation errors:\n"
		for _, desc := range result.Errors() {
			errStr += fmt.Sprintf("- %s\n", desc)
		}
		return fmt.Errorf(errStr)
	}

	return nil
}

func runConfigSchemaGenerator(schemaDir string) {
	cmd := fmt.Sprintf("go run hack/generate_config_schemas.go %s", schemaDir)
	fmt.Println("Running:", cmd)
	// In a real implementation, this would execute the command
	// For now, just print what would be executed
}

func generatePolicySchema(schemaDir string) {
	// Simplified policy schema
	schema := map[string]interface{}{
		"title":       "SA-OMF Policy Configuration",
		"type":        "object",
		"description": "Policy for self-adaptive behavior of the SA-OMF collector",
		"properties": map[string]interface{}{
			"adaptive_settings": map[string]interface{}{
				"type":        "object",
				"description": "Global adaptive behavior settings",
				"properties": map[string]interface{}{
					"enabled": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether adaptive behavior is enabled globally",
					},
					"response_time_seconds": map[string]interface{}{
						"type":        "number",
						"description": "Target response time for adaptation in seconds",
					},
				},
			},
			"components": map[string]interface{}{
				"type":        "object",
				"description": "Component-specific configurations",
				"properties": map[string]interface{}{
					"processors": map[string]interface{}{
						"type":        "object",
						"description": "Processor configurations",
					},
				},
			},
			"safety": map[string]interface{}{
				"type":        "object",
				"description": "Safety configurations",
				"properties": map[string]interface{}{
					"cpu_limit_percent": map[string]interface{}{
						"type":        "number",
						"description": "CPU usage percentage that triggers safe mode",
						"minimum":     0,
						"maximum":     100,
					},
					"memory_limit_mb": map[string]interface{}{
						"type":        "number",
						"description": "Memory usage in MB that triggers safe mode",
						"minimum":     0,
					},
				},
			},
		},
		"required": []string{"adaptive_settings"},
	}

	schemaFile := filepath.Join(schemaDir, "policy_schema.json")
	schemaBytes, err := yaml.Marshal(schema)
	if err != nil {
		fmt.Printf("Error marshaling schema: %v\n", err)
		return
	}

	if err := ioutil.WriteFile(schemaFile, schemaBytes, 0644); err != nil {
		fmt.Printf("Error writing schema to file %s: %v\n", schemaFile, err)
		return
	}

	fmt.Printf("Policy schema written to %s\n", schemaFile)
}
