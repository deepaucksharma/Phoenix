// Package config provides standardized configuration management utilities
// for Phoenix processors to reduce duplication across processor implementations.
package config

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

// Manager provides a standardized approach to configuration management
// for processors implementing the UpdateableProcessor interface.
type Manager struct {
	logger    *zap.Logger
	processor interfaces.UpdateableProcessor
	config    interface{}
}

// NewManager creates a new configuration manager
func NewManager(logger *zap.Logger, processor interfaces.UpdateableProcessor, config interface{}) *Manager {
	return &Manager{
		logger:    logger,
		processor: processor,
		config:    config,
	}
}

// HandleConfigPatch provides a standard implementation of the OnConfigPatch method
// that can be reused across processors.
func (m *Manager) HandleConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	// Special case for enabled, which is a common field
	if patch.ParameterPath == "enabled" {
		enabledConfig, ok := m.config.(EnabledConfig)
		if !ok {
			return fmt.Errorf("config does not implement EnabledConfig")
		}
		
		enabledValue, ok := patch.NewValue.(bool)
		if !ok {
			return fmt.Errorf("enabled value must be a boolean, got %T", patch.NewValue)
		}
		
		enabledConfig.SetEnabled(enabledValue)
		m.logger.Info("Configuration updated", 
			zap.String("processor", m.processor.GetName()),
			zap.String("parameter", "enabled"),
			zap.Bool("value", enabledValue))
		return nil
	}

	// Handle nested parameters
	parts := strings.Split(patch.ParameterPath, ".")
	current := reflect.ValueOf(m.config)

	if current.Kind() != reflect.Ptr {
		return errors.New("config must be a pointer to a struct")
	}

	// Navigate to the field to update
	err := setConfigByPath(current, parts, patch.NewValue)
	if err != nil {
		return err
	}

	m.logger.Info("Configuration updated",
		zap.String("processor", m.processor.GetName()),
		zap.String("parameter", patch.ParameterPath),
		zap.Any("value", patch.NewValue))

	return nil
}

// GetConfigStatus provides a standard implementation of GetConfigStatus
// that can be reused across processors.
func (m *Manager) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	return GetDefaultConfigStatus(m.config)
}

// setConfigByPath sets a value in a struct based on a dot-separated path
func setConfigByPath(configValue reflect.Value, pathParts []string, value interface{}) error {
	if configValue.Kind() != reflect.Ptr {
		return errors.New("config must be a pointer to a struct")
	}

	current := configValue.Elem()

	// Navigate to the parent object containing the field to update
	for i := 0; i < len(pathParts)-1; i++ {
		current = navigateToField(current, pathParts[i])
		if !current.IsValid() {
			return fmt.Errorf("invalid path part: %s", pathParts[i])
		}

		// Handle pointer indirection
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return fmt.Errorf("nil pointer in path: %s", pathParts[i])
			}
			current = current.Elem()
		}
	}

	// Update the final field
	lastPart := pathParts[len(pathParts)-1]
	field := findField(current, lastPart)
	if !field.IsValid() {
		return fmt.Errorf("field not found: %s", lastPart)
	}

	if !field.CanSet() {
		return fmt.Errorf("cannot set field: %s", lastPart)
	}

	// Set the value with type checking
	valueReflect := reflect.ValueOf(value)
	if !valueReflect.Type().AssignableTo(field.Type()) {
		return fmt.Errorf("type mismatch: cannot assign %T to %s", value, field.Type())
	}

	field.Set(valueReflect)
	return nil
}

// navigateToField navigates to a field in a struct by name
func navigateToField(v reflect.Value, name string) reflect.Value {
	// Handle array/slice access
	if strings.Contains(name, "[") && strings.Contains(name, "]") {
		return navigateToArrayField(v, name)
	}

	// Standard field access
	return findField(v, name)
}

// navigateToArrayField handles navigation to array or slice elements
func navigateToArrayField(v reflect.Value, name string) reflect.Value {
	fieldName := name[:strings.Index(name, "[")]
	indexStr := name[strings.Index(name, "[")+1 : strings.Index(name, "]")]
	
	index := 0
	if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
		return reflect.Value{}
	}

	field := findField(v, fieldName)
	if !field.IsValid() {
		return reflect.Value{}
	}

	if field.Kind() != reflect.Slice && field.Kind() != reflect.Array {
		return reflect.Value{}
	}

	if index < 0 || index >= field.Len() {
		return reflect.Value{}
	}

	return field.Index(index)
}

// findField returns a struct field by name (case-insensitive and tag-aware)
func findField(v reflect.Value, name string) reflect.Value {
	if v.Kind() != reflect.Struct {
		return reflect.Value{}
	}

	// Direct field lookup by name
	if f := v.FieldByName(name); f.IsValid() {
		return f
	}

	// Case-insensitive and tag-based lookup
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		
		// Skip unexported fields
		if !ft.IsExported() {
			continue
		}
		
		// Case-insensitive match
		if strings.EqualFold(ft.Name, name) {
			return v.Field(i)
		}
		
		// Check mapstructure tag
		tag := ft.Tag.Get("mapstructure")
		if tag == "" {
			tag = ft.Tag.Get("yaml")
		}
		
		if tag != "" {
			tag = strings.Split(tag, ",")[0]
			if tag == name {
				return v.Field(i)
			}
		}
	}
	
	return reflect.Value{}
}

// GetDefaultConfigStatus returns a standard ConfigStatus based on a struct
func GetDefaultConfigStatus(config interface{}) (interfaces.ConfigStatus, error) {
	configVal := reflect.ValueOf(config)
	if configVal.Kind() == reflect.Ptr {
		configVal = configVal.Elem()
	}

	if configVal.Kind() != reflect.Struct {
		return interfaces.ConfigStatus{}, errors.New("config must be a struct")
	}

	// Get all exported fields as parameters
	params := make(map[string]interface{})
	configType := configVal.Type()

	// Process each field in the struct
	for i := 0; i < configVal.NumField(); i++ {
		field := configVal.Field(i)
		fieldType := configType.Field(i)

		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}

		// Get field name from mapstructure tag if available
		fieldName := fieldType.Name
		if tag := fieldType.Tag.Get("mapstructure"); tag != "" {
			name := strings.Split(tag, ",")[0]
			if name != "" {
				fieldName = name
			}
		}

		// Add field to parameters with proper name
		params[fieldName] = field.Interface()
	}

	// Check if config implements EnabledConfig
	var enabled bool
	if enabledConfig, ok := config.(EnabledConfig); ok {
		enabled = enabledConfig.IsEnabled()
	}

	return interfaces.ConfigStatus{
		Parameters: params,
		Enabled:    enabled,
	}, nil
}

// EnabledConfig is the interface for configs that can be enabled/disabled
type EnabledConfig interface {
	IsEnabled() bool
	SetEnabled(enabled bool)
}

// ValidatableConfig is the interface for configs that can be validated
type ValidatableConfig interface {
	Validate() error
}