package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %s CONFIG_FILE SCHEMA_FILE\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	configFile := os.Args[1]
	schemaFile := os.Args[2]

	configData, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading config: %v\n", err)
		os.Exit(1)
	}

	schemaData, err := os.ReadFile(schemaFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading schema: %v\n", err)
		os.Exit(1)
	}

	var jsonData any
	if err := yaml.Unmarshal(configData, &jsonData); err != nil {
		fmt.Fprintf(os.Stderr, "parsing YAML: %v\n", err)
		os.Exit(1)
	}

	schemaLoader := gojsonschema.NewBytesLoader(schemaData)
	docLoader := gojsonschema.NewGoLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
		os.Exit(1)
	}
	if !result.Valid() {
		for _, err := range result.Errors() {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
