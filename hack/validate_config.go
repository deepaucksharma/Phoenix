// validate_config.go - Validate config files against JSON schemas
package main

import (
	"fmt"
	"os"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run validate_config.go <config-file> <schema-file>")
		os.Exit(1)
	}

	configFile := os.Args[1]
	schemaFile := os.Args[2]

	// Read config file
	configData, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}

	// Convert YAML to JSON
	var configJSON interface{}
	err = yaml.Unmarshal(configData, &configJSON)
	if err != nil {
		fmt.Printf("Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	// Load schema
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaFile)
	documentLoader := gojsonschema.NewGoLoader(configJSON)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		fmt.Printf("Error validating config: %v\n", err)
		os.Exit(1)
	}

	if !result.Valid() {
		fmt.Printf("Config file %s is not valid:\n", configFile)
		for _, err := range result.Errors() {
			fmt.Printf("- %s\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("Config file %s is valid\n", configFile)
}
