package safety

// MetricsProvider defines the interface for components that provide resource usage metrics
type MetricsProvider interface {
	GetCPUUsage() int // Returns CPU usage in millicores (1000 = 1 core)
	GetMemoryUsage() int // Returns memory usage in MiB
}

// DefaultMetricsProvider implements basic metrics collection
type DefaultMetricsProvider struct {
}

// GetCPUUsage returns the current CPU usage in millicores
func (p *DefaultMetricsProvider) GetCPUUsage() int {
	// In a real implementation, this would measure actual CPU usage
	// For now, we'll return a placeholder value
	return 300 // 30% of one core
}

// GetMemoryUsage returns the current memory usage in MiB
func (p *DefaultMetricsProvider) GetMemoryUsage() int {
	// In a real implementation, this would measure actual memory usage
	// For now, we'll return a placeholder value
	return 150 // 150 MiB
}