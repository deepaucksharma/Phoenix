# Base Processor

This package provides a common implementation for processors in the SA-OMF system. It reduces code duplication by extracting common functionality into reusable components.

## Features

- Common implementation of Component interface methods (Start, Shutdown)
- Lock handling for thread safety
- Metrics emitter management
- Configuration access and modification using reflection
- Support for EnabledConfig interface for processors that can be enabled/disabled

## Usage

To use the BaseProcessor in your own processor implementation:

```go
// Define your processor configuration, embedding the BaseConfig
type MyConfig struct {
    base.BaseConfig `mapstructure:",squash"` // Enables embedding in YAML/config files
    
    // Add your processor-specific configuration fields
    MyParam int `mapstructure:"my_param"`
}

// Validate the configuration
func (cfg *MyConfig) Validate() error {
    // First validate base config
    if err := cfg.BaseConfig.Validate(); err != nil {
        return err
    }
    
    // Then validate processor-specific fields
    if cfg.MyParam < 0 {
        return errors.New("my_param must be non-negative")
    }
    
    return nil
}

// Define your processor implementation
type myProcessor struct {
    *base.BaseProcessor
    config *MyConfig
    
    // Add processor-specific fields
    myCounter int
}

// Create a new processor
func newProcessor(cfg *MyConfig, settings processor.CreateSettings, nextConsumer consumer.Metrics) (*myProcessor, error) {
    p := &myProcessor{
        BaseProcessor: base.NewBaseProcessor(
            settings.Logger,
            nextConsumer,
            "my_processor",
            settings.ID,
        ),
        config: cfg,
        myCounter: 0,
    }
    
    return p, nil
}

// Implement ConsumeMetrics
func (p *myProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
    p.Lock()
    defer p.Unlock()
    
    // Check if processor is disabled, pass through data if it is
    if err := p.CheckIsDisabled(ctx, md, func() bool { return p.config.IsEnabled() }); err != nil {
        return err
    }
    
    // Process metrics...
    
    // Forward processed metrics to next consumer
    return p.GetNext().ConsumeMetrics(ctx, md)
}

// Implement OnConfigPatch for the UpdateableProcessor interface
func (p *myProcessor) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
    p.Lock()
    defer p.Unlock()
    
    // Get the previous value for potential rollback
    prevValue, err := base.GetConfigByPath(p.config, patch.ParameterPath)
    if err != nil {
        return fmt.Errorf("failed to get current config value: %w", err)
    }
    
    // Update the config
    if err := base.SetConfigByPath(p.config, patch.ParameterPath, patch.NewValue); err != nil {
        return fmt.Errorf("failed to apply config patch: %w", err)
    }
    
    // Validate the new config
    if err := p.config.Validate(); err != nil {
        // Rollback the change
        _ = base.SetConfigByPath(p.config, patch.ParameterPath, prevValue)
        return fmt.Errorf("invalid configuration after patch: %w", err)
    }
    
    p.GetLogger().Info("Applied configuration patch",
        zap.String("parameter", patch.ParameterPath),
        zap.Any("value", patch.NewValue),
    )
    
    return nil
}

// Implement GetConfigStatus for the UpdateableProcessor interface
func (p *myProcessor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
    p.RLock()
    defer p.RUnlock()
    
    return base.GetDefaultConfigStatus(p.config)
}
```

## Benefits

Using the BaseProcessor provides several benefits:

1. Reduced code duplication across processors
2. Consistent implementation of common interfaces
3. Simplified error handling and logging
4. Standard implementation of thread safety
5. Uniform configuration management

## Design Considerations

- The BaseProcessor uses reflection to access config fields, which adds some runtime overhead but makes the code much more maintainable
- The implementation follows the Go embedding pattern, which is more idiomatic than inheritance
- Error messages are designed to be developer-friendly for easier debugging
