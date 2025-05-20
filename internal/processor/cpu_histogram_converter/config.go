// Package cpu_histogram_converter provides a processor that converts process CPU time metrics
// into CPU utilization histogram metrics, making them suitable for visualization and analysis.
package cpu_histogram_converter

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

// Config defines the configuration for the cpu_histogram_converter processor.
type Config struct {
	// ProcessorSettings has the common settings for a processor.
	processorhelper.ProcessorSettings `mapstructure:",squash"`

	// Enabled determines whether this processor is enabled.
	Enabled bool `mapstructure:"enabled"`

	// InputMetricName is the name of the CPU time metric to convert.
	// Default: "process.cpu.time"
	InputMetricName string `mapstructure:"input_metric_name"`

	// OutputMetricName is the name of the histogram metric to output.
	// Default: "process.cpu.utilization.histogram"
	OutputMetricName string `mapstructure:"output_metric_name"`

	// CollectionIntervalSeconds is the interval in seconds at which metrics are collected.
	// This is used to calculate utilization from cumulative counters.
	// Default: 60
	CollectionIntervalSeconds int `mapstructure:"collection_interval_seconds"`

	// HostCPUCount is the number of CPU cores on the host.
	// If set to 0, the processor will auto-detect.
	// Default: 0 (auto-detect)
	HostCPUCount int `mapstructure:"host_cpu_count"`

	// TopKOnly determines whether to only generate histograms for processes
	// that are tagged as being in the top-k set.
	// Default: false (generate for all processes)
	TopKOnly bool `mapstructure:"top_k_only"`

	// HistogramBuckets defines the bucket boundaries for the utilization histogram.
	// Values are in percentage of a single CPU core (0-100).
	// Default: [0.1, 0.5, 1, 2, 5, 10, 25, 50, 75, 100, 200, 400, 800]
	HistogramBuckets []float64 `mapstructure:"histogram_buckets"`

	// StateStoragePath is the path to store state between restarts.
	// If empty, state is only kept in memory.
	// Default: "" (in-memory only)
	StateStoragePath string `mapstructure:"state_storage_path"`

	// StateFlushIntervalSeconds is how often to flush state to disk if storage path is set.
	// Default: 300 (5 minutes)
	StateFlushIntervalSeconds int `mapstructure:"state_flush_interval_seconds"`

	// MaxProcessesInMemory is the maximum number of processes to track in memory.
	// If exceeded, least recently used processes will be evicted.
	// Default: 10000
	MaxProcessesInMemory int `mapstructure:"max_processes_in_memory"`
}

// Validate checks if the processor configuration is valid.
func (cfg *Config) Validate() error {
	if !cfg.Enabled {
		return nil
	}

	if cfg.InputMetricName == "" {
		return fmt.Errorf("input_metric_name must be specified")
	}

	if cfg.OutputMetricName == "" {
		return fmt.Errorf("output_metric_name must be specified")
	}

	if cfg.CollectionIntervalSeconds <= 0 {
		return fmt.Errorf("collection_interval_seconds must be greater than 0")
	}

	if cfg.HostCPUCount < 0 {
		return fmt.Errorf("host_cpu_count must be greater than or equal to 0")
	}

	if len(cfg.HistogramBuckets) == 0 {
		return fmt.Errorf("histogram_buckets must not be empty")
	}

	if cfg.StateFlushIntervalSeconds <= 0 {
		return fmt.Errorf("state_flush_interval_seconds must be greater than 0")
	}

	if cfg.MaxProcessesInMemory <= 0 {
		return fmt.Errorf("max_processes_in_memory must be greater than 0")
	}

	// Verify bucket boundaries are in ascending order
	for i := 1; i < len(cfg.HistogramBuckets); i++ {
		if cfg.HistogramBuckets[i] <= cfg.HistogramBuckets[i-1] {
			return fmt.Errorf("histogram_buckets must be in ascending order")
		}
	}

	return nil
}

// IsEnabled returns whether this processor is enabled.
func (cfg *Config) IsEnabled() bool {
	return cfg.Enabled
}

// CreateDefaultConfig creates the default configuration for the processor.
func createDefaultConfig() component.Config {
	return &Config{
		ProcessorSettings: processorhelper.NewProcessorSettings(component.NewID(typeStr)),
		Enabled:           true,
		InputMetricName:   "process.cpu.time",
		OutputMetricName:  "process.cpu.utilization.histogram",
		CollectionIntervalSeconds: 10,
		HostCPUCount:      0, // Auto-detect
		TopKOnly:          false,
		HistogramBuckets:  []float64{0.1, 0.5, 1, 2, 5, 10, 25, 50, 75, 100, 200, 400, 800},
		StateStoragePath:  "",
		StateFlushIntervalSeconds: 300,
		MaxProcessesInMemory: 10000,
	}
}