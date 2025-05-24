package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

type ProcessSimulator struct {
	profile      string
	processCount int
	duration     time.Duration
	processes    map[string]*SimulatedProcess
	mu           sync.RWMutex
	logger       *zap.Logger
	startTime    time.Time
}

type SimulatedProcess struct {
	Name       string
	PID        int
	CPUPattern string
	MemPattern string
	StartTime  time.Time
	Lifetime   time.Duration
	cmd        *exec.Cmd
}

type Profile struct {
	Name      string
	Patterns  []ProcessPattern
	ChurnRate float64 // Percentage of processes to restart per hour
}

type ProcessPattern struct {
	NameTemplate string
	CPUPattern   string // steady, spiky, growing, random
	MemPattern   string // steady, spiky, growing, random
	Lifetime     time.Duration
	Count        int
}

var profiles = map[string]*Profile{
	"realistic": {
		Name: "realistic",
		Patterns: []ProcessPattern{
			{NameTemplate: "nginx-worker-%d", CPUPattern: "steady", MemPattern: "steady", Count: 4},
			{NameTemplate: "postgres-%d", CPUPattern: "spiky", MemPattern: "growing", Count: 2},
			{NameTemplate: "redis-server-%d", CPUPattern: "steady", MemPattern: "steady", Count: 1},
			{NameTemplate: "python-app-%d", CPUPattern: "spiky", MemPattern: "spiky", Count: 8},
			{NameTemplate: "node-service-%d", CPUPattern: "random", MemPattern: "steady", Count: 6},
			{NameTemplate: "chrome-tab-%d", CPUPattern: "random", MemPattern: "growing", Lifetime: 5 * time.Minute, Count: 20},
			{NameTemplate: "cron-job-%d", CPUPattern: "spiky", MemPattern: "steady", Lifetime: 1 * time.Minute, Count: 5},
		},
		ChurnRate: 0.1, // 10% of processes restart per hour
	},
	"high-cardinality": {
		Name: "high-cardinality",
		Patterns: []ProcessPattern{
			{NameTemplate: "microservice-%d-%d", CPUPattern: "random", MemPattern: "random", Count: 100},
			{NameTemplate: "worker-%s-%d", CPUPattern: "spiky", MemPattern: "random", Count: 50},
			{NameTemplate: "job-%s-%s-%d", CPUPattern: "random", MemPattern: "random", Lifetime: 1 * time.Minute, Count: 200},
			{NameTemplate: "container-%d", CPUPattern: "steady", MemPattern: "growing", Count: 150},
		},
		ChurnRate: 0.5, // 50% churn rate
	},
	"process-churn": {
		Name: "process-churn",
		Patterns: []ProcessPattern{
			{NameTemplate: "short-lived-%d", CPUPattern: "spiky", MemPattern: "steady", Lifetime: 30 * time.Second, Count: 50},
			{NameTemplate: "batch-job-%d", CPUPattern: "steady", MemPattern: "growing", Lifetime: 2 * time.Minute, Count: 30},
			{NameTemplate: "temp-worker-%d", CPUPattern: "random", MemPattern: "random", Lifetime: 1 * time.Minute, Count: 40},
		},
		ChurnRate: 0.8, // 80% churn rate
	},
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Parse environment variables
	profile := os.Getenv("PROFILE")
	if profile == "" {
		profile = "realistic"
	}

	duration := os.Getenv("DURATION")
	if duration == "" {
		duration = "1h"
	}

	processCount := 100
	if pc := os.Getenv("PROCESS_COUNT"); pc != "" {
		if n, err := strconv.Atoi(pc); err == nil {
			processCount = n
		}
	}

	dur, err := time.ParseDuration(duration)
	if err != nil {
		logger.Fatal("Invalid duration", zap.Error(err))
	}

	simulator := &ProcessSimulator{
		profile:      profile,
		processCount: processCount,
		duration:     dur,
		processes:    make(map[string]*SimulatedProcess),
		logger:       logger,
		startTime:    time.Now(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		cancel()
	}()

	// Run simulation
	if err := simulator.Run(ctx); err != nil {
		logger.Error("Simulation failed", zap.Error(err))
		os.Exit(1)
	}
}

func (s *ProcessSimulator) Run(ctx context.Context) error {
	s.logger.Info("Starting process simulation",
		zap.String("profile", s.profile),
		zap.Int("processCount", s.processCount),
		zap.Duration("duration", s.duration))

	// Load profile
	profile, ok := profiles[s.profile]
	if !ok {
		return fmt.Errorf("unknown profile: %s", s.profile)
	}

	// Start initial processes
	if err := s.startInitialProcesses(profile); err != nil {
		return fmt.Errorf("failed to start initial processes: %w", err)
	}

	// Run simulation loop
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(s.duration)
	churnTicker := time.NewTicker(1 * time.Minute)
	defer churnTicker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updateProcesses()
			s.checkLifetimes(profile)

		case <-churnTicker.C:
			s.simulateChurn(profile)

		case <-timeout:
			s.logger.Info("Simulation duration reached")
			return s.cleanup()

		case <-ctx.Done():
			s.logger.Info("Context cancelled")
			return s.cleanup()
		}
	}
}

func (s *ProcessSimulator) startInitialProcesses(profile *Profile) error {
	processIdx := 0
	
	for _, pattern := range profile.Patterns {
		count := pattern.Count
		if s.processCount < 100 && pattern.Count > 10 {
			// Scale down for smaller simulations
			count = pattern.Count * s.processCount / 100
			if count < 1 {
				count = 1
			}
		}

		for i := 0; i < count && processIdx < s.processCount; i++ {
			proc := s.createProcess(pattern, i)
			if err := s.startProcess(proc); err != nil {
				s.logger.Warn("Failed to start process", 
					zap.String("name", proc.Name),
					zap.Error(err))
				continue
			}
			processIdx++
			
			// Stagger process creation
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
		}
	}

	s.logger.Info("Initial processes started", zap.Int("count", processIdx))
	return nil
}

func (s *ProcessSimulator) createProcess(pattern ProcessPattern, index int) *SimulatedProcess {
	name := fmt.Sprintf(pattern.NameTemplate, index)
	if len(name) > 2 && name[len(name)-2:] == "%!" {
		// Handle templates with multiple placeholders
		name = fmt.Sprintf(pattern.NameTemplate, randomString(6), index)
	}

	lifetime := pattern.Lifetime
	if lifetime == 0 {
		lifetime = s.duration // Default to full simulation duration
	}

	return &SimulatedProcess{
		Name:       name,
		CPUPattern: pattern.CPUPattern,
		MemPattern: pattern.MemPattern,
		StartTime:  time.Now(),
		Lifetime:   lifetime,
	}
}

func (s *ProcessSimulator) startProcess(proc *SimulatedProcess) error {
	// Use stress-ng to simulate CPU and memory usage
	args := []string{
		"--cpu", "1",
		"--cpu-load", s.getCPULoad(proc.CPUPattern),
		"--vm", "1",
		"--vm-bytes", s.getMemorySize(proc.MemPattern),
		"--timeout", "0", // Run indefinitely
		"--metrics-brief",
	}

	cmd := exec.Command("stress-ng", args...)
	
	// Set process name in environment
	cmd.Env = append(os.Environ(), fmt.Sprintf("PROCESS_NAME=%s", proc.Name))
	
	// Set process group so we can kill all children
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		// If stress-ng is not available, create a simple busy process
		cmd = exec.Command("sh", "-c", fmt.Sprintf(
			`while true; do 
				echo "Process %s running" > /dev/null
				sleep 1
			done`, proc.Name))
		
		cmd.Env = append(os.Environ(), fmt.Sprintf("PROCESS_NAME=%s", proc.Name))
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		
		if err := cmd.Start(); err != nil {
			return err
		}
	}

	proc.cmd = cmd
	proc.PID = cmd.Process.Pid

	s.mu.Lock()
	s.processes[proc.Name] = proc
	s.mu.Unlock()

	s.logger.Debug("Started process",
		zap.String("name", proc.Name),
		zap.Int("pid", proc.PID))

	return nil
}

func (s *ProcessSimulator) getCPULoad(pattern string) string {
	elapsed := time.Since(s.startTime)
	
	switch pattern {
	case "steady":
		return "20"
	case "spiky":
		// Varies between 10-80%
		return fmt.Sprintf("%d", 10+rand.Intn(70))
	case "growing":
		// Increases over time
		growth := int(elapsed.Minutes())
		return fmt.Sprintf("%d", min(80, 10+growth))
	case "random":
		return fmt.Sprintf("%d", rand.Intn(100))
	default:
		return "20"
	}
}

func (s *ProcessSimulator) getMemorySize(pattern string) string {
	elapsed := time.Since(s.startTime)
	
	switch pattern {
	case "steady":
		return "50M"
	case "spiky":
		// Varies between 20MB-200MB
		return fmt.Sprintf("%dM", 20+rand.Intn(180))
	case "growing":
		// Increases over time
		growth := int(elapsed.Minutes()) * 5
		return fmt.Sprintf("%dM", min(500, 50+growth))
	case "random":
		return fmt.Sprintf("%dM", 10+rand.Intn(200))
	default:
		return "50M"
	}
}

func (s *ProcessSimulator) updateProcesses() {
	s.mu.RLock()
	activeCount := len(s.processes)
	s.mu.RUnlock()

	if activeCount > 0 && rand.Float64() < 0.01 { // 1% chance per second
		// Log current state
		s.logger.Info("Process simulator status",
			zap.Int("activeProcesses", activeCount),
			zap.Duration("uptime", time.Since(s.startTime)))
	}
}

func (s *ProcessSimulator) checkLifetimes(profile *Profile) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, proc := range s.processes {
		if proc.Lifetime > 0 && time.Since(proc.StartTime) > proc.Lifetime {
			s.logger.Debug("Process lifetime expired",
				zap.String("name", name),
				zap.Duration("lifetime", proc.Lifetime))
			
			s.stopProcess(proc)
			delete(s.processes, name)
			
			// Start a replacement
			for _, pattern := range profile.Patterns {
				if matchesPattern(name, pattern.NameTemplate) {
					newProc := s.createProcess(pattern, rand.Intn(1000))
					go s.startProcess(newProc)
					break
				}
			}
		}
	}
}

func (s *ProcessSimulator) simulateChurn(profile *Profile) {
	s.mu.Lock()
	defer s.mu.Unlock()

	processCount := len(s.processes)
	churns := int(float64(processCount) * profile.ChurnRate / 60) // Per minute
	
	if churns == 0 {
		return
	}

	s.logger.Info("Simulating process churn",
		zap.Int("processes", churns),
		zap.Float64("rate", profile.ChurnRate))

	// Select random processes to restart
	names := make([]string, 0, processCount)
	for name := range s.processes {
		names = append(names, name)
	}

	for i := 0; i < churns && i < len(names); i++ {
		idx := rand.Intn(len(names))
		name := names[idx]
		proc := s.processes[name]
		
		if proc != nil {
			s.stopProcess(proc)
			delete(s.processes, name)
			
			// Start a replacement
			for _, pattern := range profile.Patterns {
				if matchesPattern(name, pattern.NameTemplate) {
					newProc := s.createProcess(pattern, rand.Intn(1000))
					go s.startProcess(newProc)
					break
				}
			}
		}
	}
}

func (s *ProcessSimulator) stopProcess(proc *SimulatedProcess) {
	if proc.cmd != nil && proc.cmd.Process != nil {
		// Kill the process group
		syscall.Kill(-proc.cmd.Process.Pid, syscall.SIGTERM)
		
		// Wait briefly for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- proc.cmd.Wait()
		}()
		
		select {
		case <-done:
			// Process exited
		case <-time.After(2 * time.Second):
			// Force kill if still running
			syscall.Kill(-proc.cmd.Process.Pid, syscall.SIGKILL)
		}
	}
}

func (s *ProcessSimulator) cleanup() error {
	s.logger.Info("Cleaning up processes")
	
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, proc := range s.processes {
		s.logger.Debug("Stopping process", zap.String("name", name))
		s.stopProcess(proc)
	}

	s.processes = make(map[string]*SimulatedProcess)
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func matchesPattern(name, pattern string) bool {
	// Simple pattern matching - could be improved
	return len(name) > 0 && len(pattern) > 0
}