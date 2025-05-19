// Package base provides the base implementation for SA-OMF processors.
package base

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/yourorg/sa-omf/internal/interfaces"
	"github.com/yourorg/sa-omf/pkg/metrics"
)

// Config is the interface that all processor configs must implement
type Config interface {
	Validate() error
}

// EnabledConfig is the interface for processor configs that can be enabled/disabled
type EnabledConfig interface {
	Config
	IsEnabled() bool
	SetEnabled(enabled bool)
}

// BaseProcessor provides a common implementation for processors that can be dynamically configured.
type BaseProcessor struct {
	logger         *zap.Logger
	next           consumer.Metrics
	lock           sync.RWMutex
	metricsEmitter *metrics.MetricsEmitter
	name           string
	id             component.ID
}

// NewBaseProcessor creates a new BaseProcessor.
func NewBaseProcessor(logger *zap.Logger, next consumer.Metrics, name string, id component.ID) *BaseProcessor {
	return &BaseProcessor{
		logger: logger,
		next:   next,
		name:   name,
		id:     id,
	}
}

// Start initializes the processor.
func (p *BaseProcessor) Start(_ context.Context, _ component.Host) error {
	// In a real implementation, we would initialize metrics here
	// For now, we'll leave this as a no-op to simplify testing
	return nil
}

// Shutdown cleans up resources.
func (p *BaseProcessor) Shutdown(_ context.Context) error {
	return nil
}

// Capabilities returns the consumer capabilities for this processor.
func (p *BaseProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// GetLogger returns the processor's logger.
func (p *BaseProcessor) GetLogger() *zap.Logger {
	return p.logger
}

// GetNext returns the next consumer in the pipeline.
func (p *BaseProcessor) GetNext() consumer.Metrics {
	return p.next
}

// GetMetricsEmitter returns the metrics emitter.
func (p *BaseProcessor) GetMetricsEmitter() *metrics.MetricsEmitter {
	return p.metricsEmitter
}

// Lock acquires a write lock on the processor.
func (p *BaseProcessor) Lock() {
	p.lock.Lock()
}

// Unlock releases a write lock on the processor.
func (p *BaseProcessor) Unlock() {
	p.lock.Unlock()
}

// RLock acquires a read lock on the processor.
func (p *BaseProcessor) RLock() {
	p.lock.RLock()
}

// RUnlock releases a read lock on the processor.
func (p *BaseProcessor) RUnlock() {
	p.lock.RUnlock()
}

// GetConfigByPath returns a configuration value from a struct by dot-separated path.
func GetConfigByPath(config interface{}, path string) (interface{}, error) {
	if path == "enabled" {
		// Special case for enabled, which is a common field
		enabledConfig, ok := config.(EnabledConfig)
		if !ok {
			return nil, fmt.Errorf("config does not implement EnabledConfig")
		}
		return enabledConfig.IsEnabled(), nil
	}

	parts := strings.Split(path, ".")
	current := reflect.ValueOf(config)

	for _, part := range parts {
		if current.Kind() == reflect.Ptr {
			current = current.Elem()
		}

		if current.Kind() != reflect.Struct {
			return nil, fmt.Errorf("cannot navigate to %s: not a struct", part)
		}

		// Handle array access
		if strings.Contains(part, "[") && strings.Contains(part, "]") {
			fieldName := part[:strings.Index(part, "[")]
			indexStr := part[strings.Index(part, "[")+1 : strings.Index(part, "]")]
			index := 0
			if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
				return nil, fmt.Errorf("invalid index: %s", indexStr)
			}

			field := current.FieldByName(fieldName)
			if !field.IsValid() {
				return nil, fmt.Errorf("field not found: %s", fieldName)
			}

			if field.Kind() != reflect.Slice && field.Kind() != reflect.Array {
				return nil, fmt.Errorf("field is not a slice or array: %s", fieldName)
			}

			if index < 0 || index >= field.Len() {
				return nil, fmt.Errorf("index out of range: %d", index)
			}

			current = field.Index(index)
		} else {
			// Regular field access
			field := current.FieldByName(part)
			if !field.IsValid() {
				return nil, fmt.Errorf("field not found: %s", part)
			}
			current = field
		}
	}

	return current.Interface(), nil
}

// SetConfigByPath sets a configuration value in a struct by dot-separated path.
func SetConfigByPath(config interface{}, path string, value interface{}) error {
	if path == "enabled" {
		// Special case for enabled, which is a common field
		enabledConfig, ok := config.(EnabledConfig)
		if !ok {
			return fmt.Errorf("config does not implement EnabledConfig")
		}
		enabledValue, ok := value.(bool)
		if !ok {
			return fmt.Errorf("enabled value must be a boolean, got %T", value)
		}
		enabledConfig.SetEnabled(enabledValue)
		return nil
	}

	parts := strings.Split(path, ".")
	current := reflect.ValueOf(config)

	if current.Kind() != reflect.Ptr {
		return errors.New("config must be a pointer to a struct")
	}

	current = current.Elem()

	// Navigate to the parent object containing the field to update
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]

		if current.Kind() != reflect.Struct {
			return fmt.Errorf("cannot navigate to %s: not a struct", part)
		}

		// Handle array access
		if strings.Contains(part, "[") && strings.Contains(part, "]") {
			fieldName := part[:strings.Index(part, "[")]
			indexStr := part[strings.Index(part, "[")+1 : strings.Index(part, "]")]
			index := 0
			if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
				return fmt.Errorf("invalid index: %s", indexStr)
			}

			field := current.FieldByName(fieldName)
			if !field.IsValid() {
				return fmt.Errorf("field not found: %s", fieldName)
			}

			if field.Kind() != reflect.Slice && field.Kind() != reflect.Array {
				return fmt.Errorf("field is not a slice or array: %s", fieldName)
			}

			if index < 0 || index >= field.Len() {
				return fmt.Errorf("index out of range: %d", index)
			}

			current = field.Index(index)
		} else {
			// Regular field access
			field := current.FieldByName(part)
			if !field.IsValid() {
				return fmt.Errorf("field not found: %s", part)
			}
			
			if field.Kind() == reflect.Ptr {
				if field.IsNil() {
					return fmt.Errorf("field is nil: %s", part)
				}
				field = field.Elem()
			}
			
			current = field
		}
	}

	// Update the target field
	lastPart := parts[len(parts)-1]
	
	// Handle array access in the final field
	if strings.Contains(lastPart, "[") && strings.Contains(lastPart, "]") {
		fieldName := lastPart[:strings.Index(lastPart, "[")]
		indexStr := lastPart[strings.Index(lastPart, "[")+1 : strings.Index(lastPart, "]")]
		index := 0
		if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
			return fmt.Errorf("invalid index: %s", indexStr)
		}

		field := current.FieldByName(fieldName)
		if !field.IsValid() {
			return fmt.Errorf("field not found: %s", fieldName)
		}

		if field.Kind() != reflect.Slice && field.Kind() != reflect.Array {
			return fmt.Errorf("field is not a slice or array: %s", fieldName)
		}

		if index < 0 || index >= field.Len() {
			return fmt.Errorf("index out of range: %d", index)
		}

		elemField := field.Index(index)
		if !elemField.CanSet() {
			return fmt.Errorf("cannot set field: %s[%d]", fieldName, index)
		}

		valueReflect := reflect.ValueOf(value)
		if !valueReflect.Type().AssignableTo(elemField.Type()) {
			return fmt.Errorf("type mismatch: cannot assign %T to %s", value, elemField.Type())
		}

		elemField.Set(valueReflect)
	} else {
		// Regular field access
		field := current.FieldByName(lastPart)
		if !field.IsValid() {
			return fmt.Errorf("field not found: %s", lastPart)
		}

		if !field.CanSet() {
			return fmt.Errorf("cannot set field: %s", lastPart)
		}

		valueReflect := reflect.ValueOf(value)
		if !valueReflect.Type().AssignableTo(field.Type()) {
			return fmt.Errorf("type mismatch: cannot assign %T to %s", value, field.Type())
		}

		field.Set(valueReflect)
	}

	return nil
}

// Helper for processors to implement GetConfigStatus
func GetDefaultConfigStatus(config interface{}) (interfaces.ConfigStatus, error) {
	configVal := reflect.ValueOf(config)
	if configVal.Kind() == reflect.Ptr {
		configVal = configVal.Elem()
	}
	
	if configVal.Kind() != reflect.Struct {
		return interfaces.ConfigStatus{}, errors.New("config must be a struct")
	}
	
	// Get all exported fields as parameters
	params := make(map[string]any)
	configType := configVal.Type()
	
	for i := 0; i < configVal.NumField(); i++ {
		field := configVal.Field(i)
		fieldType := configType.Field(i)
		
		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}
		
		// Add field to parameters
		params[strings.ToLower(fieldType.Name)] = field.Interface()
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

// CheckIsDisabled is a helper for processor ConsumeMetrics methods
func (p *BaseProcessor) CheckIsDisabled(ctx context.Context, md pmetric.Metrics, checkEnabled func() bool) error {
	if !checkEnabled() {
		// Pass the data unmodified if processor is disabled
		return p.next.ConsumeMetrics(ctx, md)
	}
	return nil
}