// Package main implements a synthetic workload generator for testing the SA-OMF system.
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Process represents a simulated process with metrics
type Process struct {
	Name         string
	PID          int
	CPU          float64
	Memory       float64
	Labels       map[string]string
	ChildrenPIDs []int
}

// Generator holds the workload generation state
type Generator struct {
	Processes      map[int]*Process
	ProcessCount   int
	Cardinality    int
	SpikeFrequency float64
	StablePatterns int
	CurrentTick    int
	ExitChan       chan struct{}
}

func main() {
	// Define command-line flags
	var (
		processCount   = flag.Int("processes", 1000, "Number of processes to simulate")
		spikeFrequency = flag.Float64("spike-freq", 0.05, "Frequency of load spikes")
		cardinality    = flag.Int("cardinality", 5000, "Label cardinality to generate")
		stablePatterns = flag.Int("stable-patterns", 0, "Number of stable process patterns to maintain")
		duration       = flag.Duration("duration", 10*time.Minute, "Test duration")
		tickInterval   = flag.Duration("tick", 10*time.Second, "Interval between metric updates")
	)
	flag.Parse()

	// Create generator
	generator := &Generator{
		Processes:      make(map[int]*Process),
		ProcessCount:   *processCount,
		Cardinality:    *cardinality,
		SpikeFrequency: *spikeFrequency,
		StablePatterns: *stablePatterns,
		CurrentTick:    0,
		ExitChan:       make(chan struct{}),
	}

	// Initialize generator
	generator.initialize()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run the generator
	log.Printf("Starting workload generator with %d processes and %d cardinality", *processCount, *cardinality)
	tickerDone := make(chan struct{})

	go func() {
		ticker := time.NewTicker(*tickInterval)
		defer ticker.Stop()
		defer close(tickerDone)

		// Set end time
		endTime := time.Now().Add(*duration)

		for {
			select {
			case <-ticker.C:
				generator.tick()
				if time.Now().After(endTime) {
					log.Printf("Test duration completed")
					generator.ExitChan <- struct{}{}
					return
				}
			case <-generator.ExitChan:
				return
			}
		}
	}()

	// Wait for signal or completion
	select {
	case <-sigChan:
		log.Println("Received signal, shutting down")
	case <-generator.ExitChan:
		log.Println("Test completed")
	}

	// Ensure ticker goroutine completes
	close(generator.ExitChan)
	<-tickerDone
	log.Println("Generator shutdown complete")
}

// initialize sets up the initial process state
func (g *Generator) initialize() {
	// Create root processes
	for i := 0; i < g.ProcessCount; i++ {
		pid := 1000 + i
		g.Processes[pid] = &Process{
			Name:   getRandomProcessName(),
			PID:    pid,
			CPU:    rand.Float64() * 0.2,             // Initial CPU usage between 0-20%
			Memory: rand.Float64() * 500,             // Initial memory usage 0-500MB
			Labels: generateLabels(g.Cardinality, 5), // 5 labels per process
		}
	}

	// Create process hierarchy
	for pid, proc := range g.Processes {
		// Some processes have children
		if rand.Float64() < 0.3 {
			childCount := rand.Intn(5) + 1
			for c := 0; c < childCount; c++ {
				childPID := pid*10 + c
				if _, exists := g.Processes[childPID]; !exists {
					childProc := &Process{
						Name:   proc.Name + "-child",
						PID:    childPID,
						CPU:    rand.Float64() * 0.1,
						Memory: rand.Float64() * 200,
						Labels: proc.Labels, // Children inherit labels
					}
					g.Processes[childPID] = childProc
					proc.ChildrenPIDs = append(proc.ChildrenPIDs, childPID)
				}
			}
		}
	}

	// If stable patterns requested, mark some processes as stable
	if g.StablePatterns > 0 {
		// Implementation for stable patterns
	}
}

// tick updates all process metrics for one time step
func (g *Generator) tick() {
	g.CurrentTick++
	log.Printf("Tick %d: Simulating %d processes", g.CurrentTick, len(g.Processes))

	// Update each process
	for _, proc := range g.Processes {
		// Regular variation
		proc.CPU += (rand.Float64() - 0.5) * 0.1
		if proc.CPU < 0.01 {
			proc.CPU = 0.01
		} else if proc.CPU > 1.0 {
			proc.CPU = 1.0
		}

		proc.Memory += (rand.Float64() - 0.5) * 50
		if proc.Memory < 10 {
			proc.Memory = 10
		}

		// Occasional spikes
		if rand.Float64() < g.SpikeFrequency {
			proc.CPU *= 3
			if proc.CPU > 1.0 {
				proc.CPU = 1.0
			}
			proc.Memory *= 2
		}
	}

	// Occasionally add/remove processes to simulate churn
	if rand.Float64() < 0.1 {
		// Remove some processes
		for pid := range g.Processes {
			if rand.Float64() < 0.05 {
				delete(g.Processes, pid)
			}
		}

		// Add some new processes
		currentCount := len(g.Processes)
		if currentCount < g.ProcessCount {
			for i := 0; i < g.ProcessCount-currentCount; i++ {
				pid := 10000 + g.CurrentTick*1000 + i
				g.Processes[pid] = &Process{
					Name:   getRandomProcessName(),
					PID:    pid,
					CPU:    rand.Float64() * 0.2,
					Memory: rand.Float64() * 500,
					Labels: generateLabels(g.Cardinality, 5),
				}
			}
		}
	}

	// Simulate metric output (in real implementation, this would emit to a file or endpoint)
	// For now, just print summary
	fmt.Printf("Processes: %d, Avg CPU: %.1f%%, Avg Memory: %.1f MB\n",
		len(g.Processes),
		calculateAverageCPU(g.Processes)*100,
		calculateAverageMemory(g.Processes))
}

// Helper functions
func getRandomProcessName() string {
	processes := []string{
		"node", "java", "python", "go", "ruby", "postgres",
		"mysql", "redis", "nginx", "apache", "elasticsearch",
		"kafka", "zookeeper", "mongodb", "cassandra", "prometheus",
		"grafana", "kibana", "logstash", "fluentd", "consul",
		"etcd", "haproxy", "traefik", "envoy", "istio",
		"kubernetes", "docker", "containerd", "crio", "systemd",
	}
	return processes[rand.Intn(len(processes))]
}

func generateLabels(cardinality, count int) map[string]string {
	labels := make(map[string]string)

	// Common label keys
	keys := []string{
		"environment", "region", "zone", "service", "team",
		"version", "deployment", "cluster", "instance", "shard",
	}

	// Generate random labels
	for i := 0; i < count; i++ {
		key := keys[rand.Intn(len(keys))]
		value := fmt.Sprintf("val-%d", rand.Intn(cardinality))
		labels[key] = value
	}

	return labels
}

func calculateAverageCPU(processes map[int]*Process) float64 {
	if len(processes) == 0 {
		return 0
	}

	sum := 0.0
	for _, proc := range processes {
		sum += proc.CPU
	}
	return sum / float64(len(processes))
}

func calculateAverageMemory(processes map[int]*Process) float64 {
	if len(processes) == 0 {
		return 0
	}

	sum := 0.0
	for _, proc := range processes {
		sum += proc.Memory
	}
	return sum / float64(len(processes))
}
