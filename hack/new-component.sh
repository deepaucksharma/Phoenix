#!/bin/bash
# new-component.sh - Create a new OpenTelemetry component with proper boilerplate

set -e

if [ $# -ne 2 ]; then
  echo "Usage: $0 <type> <name>"
  echo "  type: processor, extension, connector"
  echo "  name: snake_case component name"
  echo ""
  echo "Example: $0 processor adaptive_sampler"
  exit 1
fi

TYPE=$1
NAME=$2

# Check for valid type
if [[ "$TYPE" != "processor" && "$TYPE" != "extension" && "$TYPE" != "connector" ]]; then
  echo "Error: Type must be one of: processor, extension, connector"
  exit 1
fi

# Check for valid name format (snake_case)
if ! [[ $NAME =~ ^[a-z]+(_[a-z]+)*$ ]]; then
  echo "Error: Name must be in snake_case format (e.g., adaptive_sampler)"
  exit 1
fi

# Convert snake_case to CamelCase
CAMEL_NAME=""
for part in ${NAME//_/ }; do
  CAMEL_NAME+="$(tr '[:lower:]' '[:upper:]' <<< ${part:0:1})${part:1}"
done

# Create directory structure
DIR="internal/$TYPE/$NAME"
mkdir -p "$DIR"

echo "Creating component in $DIR"

# Create factory.go
cat > "$DIR/factory.go" << EOL
// Package $NAME implements a $TYPE for the SA-OMF system.
package $NAME

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/$TYPE"
)

const (
	typeStr = "$NAME"
)

// NewFactory creates a factory for the $NAME $TYPE
func NewFactory() $TYPE.Factory {
	return $TYPE.NewFactory(
		typeStr,
		createDefaultConfig,
		$TYPE.With$(tr '[:lower:]' '[:upper:]' <<< ${TYPE:0:1})${TYPE:1}s(create${CAMEL_NAME}, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates the default configuration for the $TYPE
func createDefaultConfig() component.Config {
	return &Config{
		// Default configuration values
	}
}

// create${CAMEL_NAME} creates a $TYPE instance based on the config
func create${CAMEL_NAME}(
	ctx context.Context,
	set $TYPE.CreateSettings,
	cfg component.Config,
	// Add necessary consumer parameter here based on type
) ($TYPE.$(tr '[:lower:]' '[:upper:]' <<< ${TYPE:0:1})${TYPE:1}s, error) {
	return new$CAMEL_NAME(cfg.(*Config), set)
}
EOL

# Create config.go
cat > "$DIR/config.go" << EOL
package $NAME

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

// Config defines the configuration for the $NAME $TYPE
type Config struct {
	// Add configuration fields here
	Enabled bool \`mapstructure:"enabled"\`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the $TYPE configuration is valid
func (cfg *Config) Validate() error {
	// Add validation logic here
	return nil
}
EOL

# Create implementation file based on type
if [ "$TYPE" == "processor" ]; then
  cat > "$DIR/processor.go" << EOL
package $NAME

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/yourorg/sa-omf/internal/interfaces"
	"github.com/yourorg/sa-omf/pkg/metrics"
)

// ${CAMEL_NAME}Processor implements the $NAME processor
type ${CAMEL_NAME}Processor struct {
	logger      *zap.Logger
	nextConsumer consumer.Metrics
	config      *Config
	lock        sync.RWMutex
	metrics     *metrics.MetricsEmitter
}

// Ensure the processor implements required interfaces
var _ processor.Metrics = (*${CAMEL_NAME}Processor)(nil)
var _ interfaces.UpdateableProcessor = (*${CAMEL_NAME}Processor)(nil)

// new$CAMEL_NAME creates a new $NAME processor
func new$CAMEL_NAME(config *Config, settings processor.CreateSettings) (*${CAMEL_NAME}Processor, error) {
	p := &${CAMEL_NAME}Processor{
		logger: settings.Logger,
		config: config,
	}
	
	return p, nil
}

// Start implements the Component interface
func (p *${CAMEL_NAME}Processor) Start(ctx context.Context, host component.Host) error {
	// Implementation here
	return nil
}

// Shutdown implements the Component interface
func (p *${CAMEL_NAME}Processor) Shutdown(ctx context.Context) error {
	// Implementation here
	return nil
}

// Capabilities implements the processor.Metrics interface
func (p *${CAMEL_NAME}Processor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// ConsumeMetrics processes incoming metrics
func (p *${CAMEL_NAME}Processor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.lock.RLock()
	defer p.lock.RUnlock()
	
	if !p.config.Enabled {
		// Pass through if disabled
		return p.nextConsumer.ConsumeMetrics(ctx, md)
	}
	
	// Process metrics here
	
	// Forward to next consumer
	return p.nextConsumer.ConsumeMetrics(ctx, md)
}

// OnConfigPatch implements the UpdateableProcessor interface
func (p *${CAMEL_NAME}Processor) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	// Handle configuration patches
	return nil
}

// GetConfigStatus implements the UpdateableProcessor interface
func (p *${CAMEL_NAME}Processor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	
	return interfaces.ConfigStatus{
		Parameters: map[string]any{
			// Return current parameters
		},
		Enabled: p.config.Enabled,
	}, nil
}
EOL
elif [ "$TYPE" == "extension" ]; then
  cat > "$DIR/extension.go" << EOL
package $NAME

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"
)

// ${CAMEL_NAME}Extension implements the $NAME extension
type ${CAMEL_NAME}Extension struct {
	logger *zap.Logger
	config *Config
	lock   sync.RWMutex
}

// Ensure the extension implements required interfaces
var _ extension.Extension = (*${CAMEL_NAME}Extension)(nil)

// new$CAMEL_NAME creates a new $NAME extension
func new$CAMEL_NAME(config *Config, settings extension.CreateSettings) (*${CAMEL_NAME}Extension, error) {
	return &${CAMEL_NAME}Extension{
		logger: settings.Logger,
		config: config,
	}, nil
}

// Start implements the Component interface
func (e *${CAMEL_NAME}Extension) Start(ctx context.Context, host component.Host) error {
	// Implementation here
	return nil
}

// Shutdown implements the Component interface
func (e *${CAMEL_NAME}Extension) Shutdown(ctx context.Context) error {
	// Implementation here
	return nil
}
EOL
elif [ "$TYPE" == "connector" ]; then
  cat > "$DIR/connector.go" << EOL
package $NAME

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// ${CAMEL_NAME}Connector implements the $NAME connector
type ${CAMEL_NAME}Connector struct {
	logger *zap.Logger
	config *Config
	lock   sync.RWMutex
}

// Ensure the connector implements required interfaces
var _ exporter.Metrics = (*${CAMEL_NAME}Connector)(nil)

// new$CAMEL_NAME creates a new $NAME connector
func new$CAMEL_NAME(config *Config, settings exporter.CreateSettings) (*${CAMEL_NAME}Connector, error) {
	return &${CAMEL_NAME}Connector{
		logger: settings.Logger,
		config: config,
	}, nil
}

// Start implements the Component interface
func (c *${CAMEL_NAME}Connector) Start(ctx context.Context, host component.Host) error {
	// Implementation here
	return nil
}

// Shutdown implements the Component interface
func (c *${CAMEL_NAME}Connector) Shutdown(ctx context.Context) error {
	// Implementation here
	return nil
}

// ConsumeMetrics processes incoming metrics
func (c *${CAMEL_NAME}Connector) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	// Implementation here
	return nil
}
EOL
fi

# Create test file
cat > "$DIR/${NAME}_test.go" << EOL
package $NAME

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

func Test${CAMEL_NAME}(t *testing.T) {
	// Create test configuration
	cfg := &Config{
		Enabled: true,
	}
	
	// Create test ${TYPE}
	factory := NewFactory()
	
	// Test factory
	assert.Equal(t, typeStr, factory.Type().String())
	
	// Test creation
	ctx := context.Background()
	
	// Add type-specific test setup and assertions
}
EOL

# Update main.go to register the new component
MAIN_FILE="cmd/sa-omf-otelcol/main.go"

# Add import if needed
if ! grep -q "\"github.com/yourorg/sa-omf/internal/$TYPE/$NAME\"" $MAIN_FILE; then
  sed -i "/Add more component imports/i \\\t\"github.com/yourorg/sa-omf/internal/$TYPE/$NAME\"," $MAIN_FILE
fi

# Add factory to the appropriate section
case $TYPE in
  processor)
    if ! grep -q "$NAME.NewFactory()" $MAIN_FILE; then
      sed -i "/Add custom processors as they are implemented:/a \\\t\t$NAME.NewFactory()," $MAIN_FILE
    fi
    ;;
  extension)
    if ! grep -q "$NAME.NewFactory()" $MAIN_FILE; then
      sed -i "/Add more extensions as needed/a \\\t\t$NAME.NewFactory()," $MAIN_FILE
    fi
    ;;
  connector|exporter)
    if ! grep -q "$NAME.NewFactory()" $MAIN_FILE; then
      sed -i "/Add custom exporters as they are implemented:/a \\\t\t$NAME.NewFactory()," $MAIN_FILE
    fi
    ;;
esac

echo "Component $NAME created successfully in $DIR!"
echo "Main component registered in $MAIN_FILE"
echo ""
echo "Next steps:"
echo "1. Implement the component's business logic"
echo "2. Write comprehensive tests"
echo "3. Update configuration in config/config.yaml"
