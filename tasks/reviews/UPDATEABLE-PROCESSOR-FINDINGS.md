# UpdateableProcessor Interface Review Findings

## Component Information
- **Component Type**: Interface
- **Location**: `/internal/interfaces/updateable_processor.go`
- **Primary Purpose**: Defines the interface for processors that can be dynamically reconfigured

## Issues Found

### 1. **Medium**: Lack of validation methods in ConfigPatch
- **Location**: updateable_processor.go:10-22
- **Impact**: Each implementation must implement its own validation logic
- **Remediation**: Add a Validate() method to ConfigPatch

### 2. **High**: No mechanism for versioning configuration changes
- **Location**: updateable_processor.go:10-22
- **Impact**: No way to detect conflicts when multiple changes occur concurrently
- **Remediation**: Add version or sequence number to ConfigPatch

### 3. **Medium**: Insufficient documentation for ParameterPath format
- **Location**: updateable_processor.go:14
- **Impact**: Implementations may interpret it differently
- **Remediation**: Add detailed documentation and validation for parameter path format

### 4. **Low**: ConfigStatus lacks version information
- **Location**: updateable_processor.go:24-28
- **Impact**: Can't determine if status is current or stale
- **Remediation**: Add version or timestamp field to ConfigStatus

### 5. **Medium**: Missing constants for Severity values
- **Location**: updateable_processor.go:18
- **Impact**: Inconsistent string values across implementations
- **Remediation**: Define severity constants

### 6. **Medium**: No structured error types
- **Location**: updateable_processor.go:36
- **Impact**: Difficult to distinguish different error cases
- **Remediation**: Define specific error types for different failure modes

## Improvement Recommendations

### Interface Enhancements

1. **Add validation method to ConfigPatch**
```go
// ConfigPatch defines a proposed change to a processor's configuration
type ConfigPatch struct {
    PatchID             string       `json:"patch_id"`
    TargetProcessorName component.ID  `json:"target_processor_name"`
    ParameterPath       string       `json:"parameter_path"`
    NewValue            any          `json:"new_value"`
    PrevValue           any          `json:"prev_value"`
    Reason              string       `json:"reason"`
    Severity            string       `json:"severity"`
    Source              string       `json:"source"`
    Timestamp           int64        `json:"timestamp"`
    TTLSeconds          int          `json:"ttl_seconds"`
    Version             int64        `json:"version"`  // Added version field
}

// Validate performs validation on the ConfigPatch
func (p *ConfigPatch) Validate() error {
    if p.PatchID == "" {
        return fmt.Errorf("patch ID cannot be empty")
    }
    
    if p.TargetProcessorName.Type() == "" || p.TargetProcessorName.Name() == "" {
        return fmt.Errorf("target processor name must be valid")
    }
    
    if p.ParameterPath == "" {
        return fmt.Errorf("parameter path cannot be empty")
    }
    
    // Validate parameter path format
    if !isValidParameterPath(p.ParameterPath) {
        return fmt.Errorf("invalid parameter path format: %s", p.ParameterPath)
    }
    
    // Validate severity
    switch p.Severity {
    case SeverityNormal, SeverityUrgent, SeveritySafety:
        // Valid severities
    default:
        return fmt.Errorf("invalid severity: %s", p.Severity)
    }
    
    // Validate source
    switch p.Source {
    case SourcePIDDecider, SourceOPAMP, SourceManual:
        // Valid sources
    default:
        return fmt.Errorf("invalid source: %s", p.Source)
    }
    
    // Validate TTL
    if p.TTLSeconds < 0 {
        return fmt.Errorf("TTL must not be negative")
    }
    
    // Timestamp should be in the past
    if p.Timestamp > time.Now().Unix() {
        return fmt.Errorf("timestamp is in the future")
    }
    
    return nil
}

// isValidParameterPath validates that a parameter path follows the expected format
func isValidParameterPath(path string) bool {
    // A valid path consists of dot-separated segments with alphanumeric characters
    if path == "" {
        return false
    }
    
    segments := strings.Split(path, ".")
    for _, segment := range segments {
        if segment == "" {
            return false
        }
        
        for i, c := range segment {
            if i == 0 && !unicode.IsLetter(c) {
                return false
            }
            
            if !(unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_') {
                return false
            }
        }
    }
    
    return true
}
```

2. **Add severity and source constants**
```go
// Severity constants
const (
    SeverityNormal = "normal"
    SeverityUrgent = "urgent"
    SeveritySafety = "safety"
)

// Source constants
const (
    SourcePIDDecider = "pid_decider"
    SourceOPAMP      = "opamp"
    SourceManual     = "manual"
)
```

3. **Enhance ConfigStatus with version information**
```go
type ConfigStatus struct {
    Parameters map[string]any `json:"parameters"`
    Enabled    bool           `json:"enabled"`
    Version    int64          `json:"version"`      // Added version field
    Timestamp  int64          `json:"timestamp"`    // Added timestamp field
}
```

4. **Add specific error types**
```go
// ConfigPatchError represents errors that occur during patch application
type ConfigPatchError struct {
    Type    ConfigPatchErrorType
    Message string
    Cause   error
}

func (e *ConfigPatchError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// ConfigPatchErrorType represents different categories of patch errors
type ConfigPatchErrorType string

const (
    // ErrorInvalidParameter indicates the parameter value is invalid
    ErrorInvalidParameter ConfigPatchErrorType = "invalid_parameter"
    
    // ErrorUnknownParameter indicates the parameter doesn't exist
    ErrorUnknownParameter ConfigPatchErrorType = "unknown_parameter"
    
    // ErrorUnsupportedOperation indicates the operation is not supported
    ErrorUnsupportedOperation ConfigPatchErrorType = "unsupported_operation"
    
    // ErrorInvalidPatch indicates the patch itself is invalid
    ErrorInvalidPatch ConfigPatchErrorType = "invalid_patch"
    
    // ErrorProcessorDisabled indicates the processor is disabled
    ErrorProcessorDisabled ConfigPatchErrorType = "processor_disabled"
    
    // ErrorVersionConflict indicates a version conflict with another patch
    ErrorVersionConflict ConfigPatchErrorType = "version_conflict"
)
```

5. **Add a parameter type validation helper**
```go
// ValidateParameterType validates that a value is of the expected type
func ValidateParameterType(paramName string, value interface{}, expectedType reflect.Type) error {
    if value == nil {
        return &ConfigPatchError{
            Type:    ErrorInvalidParameter,
            Message: fmt.Sprintf("parameter %s cannot be nil", paramName),
        }
    }
    
    actualType := reflect.TypeOf(value)
    if actualType != expectedType {
        // Check for numeric conversions
        if expectedType.Kind() == reflect.Float64 && actualType.Kind() == reflect.Int {
            // Int can be converted to float64, so this is acceptable
            return nil
        }
        
        if expectedType.Kind() == reflect.Int && actualType.Kind() == reflect.Float64 {
            // Float64 can be converted to int, but may lose precision
            floatVal := reflect.ValueOf(value).Float()
            if floatVal == float64(int(floatVal)) {
                // No precision loss, this is acceptable
                return nil
            }
        }
        
        return &ConfigPatchError{
            Type:    ErrorInvalidParameter,
            Message: fmt.Sprintf("parameter %s has wrong type, expected %v, got %v", 
                paramName, expectedType, actualType),
        }
    }
    
    return nil
}
```

### Testing Improvements

1. **Enhance interface testing utilities**
```go
// Add test helpers for different error types
func TestConfigPatchErrorTypes(t *testing.T, p interfaces.UpdateableProcessor) {
    ctx := context.Background()
    
    // Test unknown parameter
    unknownPatch := interfaces.ConfigPatch{
        PatchID:             "test-unknown",
        TargetProcessorName: component.MustNewID("test"),
        ParameterPath:       "non_existent_parameter",
        NewValue:            "test",
        Severity:            interfaces.SeverityNormal,
        Source:              interfaces.SourceManual,
        Timestamp:           time.Now().Unix(),
        TTLSeconds:          300,
    }
    
    err := p.OnConfigPatch(ctx, unknownPatch)
    require.Error(t, err)
    
    // Type assertion to check error type
    var configErr *interfaces.ConfigPatchError
    if errors.As(err, &configErr) {
        assert.Equal(t, interfaces.ErrorUnknownParameter, configErr.Type, 
            "Should return ErrorUnknownParameter for non-existent parameter")
    } else {
        t.Errorf("Expected ConfigPatchError, got %T", err)
    }
    
    // Similarly test other error types
    // ...
}
```

2. **Add versioning tests**
```go
func TestConfigVersioning(t *testing.T, p interfaces.UpdateableProcessor) {
    ctx := context.Background()
    
    // Get initial status
    status, err := p.GetConfigStatus(ctx)
    require.NoError(t, err)
    
    // Track initial version
    initialVersion := status.Version
    
    // Apply a patch with matching version
    patch := interfaces.ConfigPatch{
        PatchID:             "test-version",
        TargetProcessorName: component.MustNewID("test"),
        ParameterPath:       "enabled",
        NewValue:            true,
        Version:             initialVersion,
        Severity:            interfaces.SeverityNormal,
        Source:              interfaces.SourceManual,
        Timestamp:           time.Now().Unix(),
        TTLSeconds:          300,
    }
    
    err = p.OnConfigPatch(ctx, patch)
    require.NoError(t, err)
    
    // Get new status
    newStatus, err := p.GetConfigStatus(ctx)
    require.NoError(t, err)
    
    // Version should be incremented
    assert.Greater(t, newStatus.Version, initialVersion, 
        "Version should be incremented after patch")
    
    // Try to apply patch with outdated version
    outdatedPatch := interfaces.ConfigPatch{
        PatchID:             "test-outdated",
        TargetProcessorName: component.MustNewID("test"),
        ParameterPath:       "enabled",
        NewValue:            false,
        Version:             initialVersion, // Use old version
        Severity:            interfaces.SeverityNormal,
        Source:              interfaces.SourceManual,
        Timestamp:           time.Now().Unix(),
        TTLSeconds:          300,
    }
    
    err = p.OnConfigPatch(ctx, outdatedPatch)
    require.Error(t, err)
    
    var configErr *interfaces.ConfigPatchError
    if errors.As(err, &configErr) {
        assert.Equal(t, interfaces.ErrorVersionConflict, configErr.Type, 
            "Should return ErrorVersionConflict for outdated version")
    }
}
```

## Implementation Tasks

1. Add proper validation method to ConfigPatch
2. Define severity and source constants
3. Add version field to ConfigPatch and ConfigStatus
4. Implement detailed documentation for parameter path format
5. Add specific error types for different failure cases
6. Enhance test suite with versioning and error handling tests
7. Create helper functions for common validation operations
8. Update existing processor implementations to use new features