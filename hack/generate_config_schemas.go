package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// ConfigSchema represents the structure of the configuration schema
type ConfigSchema struct {
	Title       string                 `json:"title"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Properties  map[string]interface{} `json:"properties"`
	Required    []string               `json:"required,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run hack/generate_config_schemas.go [output_dir]")
		os.Exit(1)
	}

	outputDir := os.Args[1]

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate config schemas for different components
	generateProcessorSchemas(outputDir)
	generateExtensionSchemas(outputDir)
	generateReceiverSchemas(outputDir)
	generateExporterSchemas(outputDir)
	generateRootSchema(outputDir)

	fmt.Println("Configuration schemas generated successfully.")
}

func generateProcessorSchemas(outputDir string) {
	processors := []string{
		"adaptive_pid",
		"adaptive_topk",
		"priority_tagger",
		"cardinality_guardian",
		"multi_temporal_adaptive_engine",
		"others_rollup",
		"process_context_learner",
		"reservoir_sampler",
		"semantic_correlator",
	}

	for _, processor := range processors {
		schema := ConfigSchema{
			Title:       fmt.Sprintf("%s Processor Configuration", strings.Title(strings.ReplaceAll(processor, "_", " "))),
			Type:        "object",
			Description: fmt.Sprintf("Configuration for the %s processor", processor),
			Properties: map[string]interface{}{
				"enabled": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether this processor is enabled",
				},
				// Add common properties for all processors
			},
			Required: []string{"enabled"},
		}

		// Add processor-specific properties
		switch processor {
		case "adaptive_pid":
			schema.Properties["controllers"] = map[string]interface{}{
				"type":        "array",
				"description": "List of PID controllers",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the controller",
						},
						"enabled": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether this controller is enabled",
						},
						"kpi_metric_name": map[string]interface{}{
							"type":        "string",
							"description": "Metric name to monitor for KPI",
						},
						"kpi_target_value": map[string]interface{}{
							"type":        "number",
							"description": "Target value for the KPI",
						},
						"kp": map[string]interface{}{
							"type":        "number",
							"description": "Proportional gain",
						},
						"ki": map[string]interface{}{
							"type":        "number",
							"description": "Integral gain",
						},
						"kd": map[string]interface{}{
							"type":        "number",
							"description": "Derivative gain",
						},
						"hysteresis_percent": map[string]interface{}{
							"type":        "number",
							"description": "Minimum percent change required before issuing a new patch",
						},
					},
					"required": []string{"name", "enabled", "kpi_metric_name", "kpi_target_value"},
				},
			}
		case "adaptive_topk":
			schema.Properties["k_value"] = map[string]interface{}{
				"type":        "integer",
				"description": "Initial k value for the top-k algorithm",
				"minimum":     1,
			}
			schema.Properties["target_coverage"] = map[string]interface{}{
				"type":        "number",
				"description": "Target coverage percentage (0.0-1.0)",
				"minimum":     0,
				"maximum":     1,
			}
		case "priority_tagger":
			schema.Properties["rules"] = map[string]interface{}{
				"type":        "array",
				"description": "Priority tagging rules",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"match": map[string]interface{}{
							"type":        "string",
							"description": "Regular expression to match against resource attributes",
						},
						"priority": map[string]interface{}{
							"type":        "string",
							"description": "Priority to assign",
							"enum":        []string{"critical", "high", "medium", "low"},
						},
					},
					"required": []string{"match", "priority"},
				},
			}
		}

		// Write schema to file
		schemaFile := filepath.Join(outputDir, fmt.Sprintf("processor_%s_schema.json", processor))
		writeSchema(schema, schemaFile)
	}
}

func generateExtensionSchemas(outputDir string) {
	extensions := []string{
		"pic_control",
	}

	for _, extension := range extensions {
		schema := ConfigSchema{
			Title:       fmt.Sprintf("%s Extension Configuration", strings.Title(strings.ReplaceAll(extension, "_", " "))),
			Type:        "object",
			Description: fmt.Sprintf("Configuration for the %s extension", extension),
			Properties:  map[string]interface{}{},
		}

		// Add extension-specific properties
		switch extension {
		case "pic_control":
			schema.Properties["policy_file_path"] = map[string]interface{}{
				"type":        "string",
				"description": "Path to the policy file",
			}
			schema.Properties["max_patches_per_minute"] = map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of configuration patches to apply per minute",
				"minimum":     1,
			}
			schema.Properties["patch_cooldown_seconds"] = map[string]interface{}{
				"type":        "integer",
				"description": "Cooldown period in seconds between patches to the same parameter",
				"minimum":     0,
			}
			schema.Properties["safe_mode_processor_configs"] = map[string]interface{}{
				"type":        "object",
				"description": "Processor configurations to use when in safe mode",
			}
			schema.Required = []string{"policy_file_path"}
		}

		// Write schema to file
		schemaFile := filepath.Join(outputDir, fmt.Sprintf("extension_%s_schema.json", extension))
		writeSchema(schema, schemaFile)
	}
}

func generateReceiverSchemas(outputDir string) {
	// Simplified for brevity - would include actual receiver schemas
	schema := ConfigSchema{
		Title:       "Receiver Configurations",
		Type:        "object",
		Description: "Configuration for all receivers",
		Properties:  map[string]interface{}{},
	}

	schemaFile := filepath.Join(outputDir, "receivers_schema.json")
	writeSchema(schema, schemaFile)
}

func generateExporterSchemas(outputDir string) {
	exporters := []string{
		"pic_connector",
	}

	for _, exporter := range exporters {
		schema := ConfigSchema{
			Title:       fmt.Sprintf("%s Exporter Configuration", strings.Title(strings.ReplaceAll(exporter, "_", " "))),
			Type:        "object",
			Description: fmt.Sprintf("Configuration for the %s exporter", exporter),
			Properties:  map[string]interface{}{},
		}

		// Add exporter-specific properties
		switch exporter {
		case "pic_connector":
			// This exporter typically doesn't have configuration
		}

		// Write schema to file
		schemaFile := filepath.Join(outputDir, fmt.Sprintf("exporter_%s_schema.json", exporter))
		writeSchema(schema, schemaFile)
	}
}

func generateRootSchema(outputDir string) {
	schema := ConfigSchema{
		Title:       "SA-OMF Configuration",
		Type:        "object",
		Description: "Root configuration for the SA-OMF collector",
		Properties: map[string]interface{}{
			"extensions": map[string]interface{}{
				"type":        "object",
				"description": "Extension configurations",
			},
			"receivers": map[string]interface{}{
				"type":        "object",
				"description": "Receiver configurations",
			},
			"processors": map[string]interface{}{
				"type":        "object",
				"description": "Processor configurations",
			},
			"exporters": map[string]interface{}{
				"type":        "object",
				"description": "Exporter configurations",
			},
			"service": map[string]interface{}{
				"type":        "object",
				"description": "Service configuration",
				"properties": map[string]interface{}{
					"pipelines": map[string]interface{}{
						"type":        "object",
						"description": "Pipeline definitions",
					},
					"extensions": map[string]interface{}{
						"type":        "array",
						"description": "List of extensions to enable",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"pipelines"},
			},
		},
		Required: []string{"service"},
	}

	schemaFile := filepath.Join(outputDir, "config_schema.json")
	writeSchema(schema, schemaFile)
}

func writeSchema(schema ConfigSchema, filePath string) {
	// Convert to JSON
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schemaBytes, err := yaml.Marshal(schema)
	if err != nil {
		fmt.Printf("Error marshaling schema: %v\n", err)
		return
	}

	// Write to file
	if err := ioutil.WriteFile(filePath, schemaBytes, 0644); err != nil {
		fmt.Printf("Error writing schema to file %s: %v\n", filePath, err)
		return
	}

	fmt.Printf("Schema written to %s\n", filePath)
}
