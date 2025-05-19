package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"time"
)

type scenario struct {
	name string
	run  func(string, time.Duration) error
}

func main() {
	env := flag.String("env", "docker", "Test environment: docker or k8s")
	duration := flag.Duration("duration", 30*time.Minute, "Test duration")
	flag.Parse()

	scenarios := []scenario{
		{"config_oscillation", runConfigOscillation},
		{"process_explosion", runProcessExplosion},
		{"cardinality_bomb", runCardinalityBomb},
		{"resource_starvation", runResourceStarvation},
		{"network_partition", runNetworkPartition},
		{"out_of_memory", runOutOfMemory},
	}

	for _, s := range scenarios {
		log.Printf("Running chaos scenario: %s", s.name)
		if err := s.run(*env, *duration); err != nil {
			log.Printf("Scenario failed: %v", err)
		} else {
			log.Printf("Scenario passed")
		}
	}
}

func runConfigOscillation(env string, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	// Start baseline workload
	wl, err := startWorkload(ctx, d, 1000, 5000, 0.05)
	if err != nil {
		return err
	}
	defer wl.Wait()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	toggle := false
	for {
		select {
		case <-ctx.Done():
			return wl.Wait()
		case <-ticker.C:
			// Alternate between two k_value settings
			val := 20
			if toggle {
				val = 60
			}
			toggle = !toggle
			if err := sendPatch("adaptive_topk", "k_value", val); err != nil {
				log.Printf("patch error: %v", err)
			}

			if safeModeActive() {
				return fmt.Errorf("safe mode activated")
			}
		}
	}
}

func runProcessExplosion(env string, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	wl, err := startWorkload(ctx, d, 20000, 1000, 0.1)
	if err != nil {
		return err
	}
	defer wl.Wait()

	for {
		select {
		case <-ctx.Done():
			return wl.Wait()
		case <-time.After(5 * time.Second):
			if safeModeActive() {
				return fmt.Errorf("safe mode activated")
			}
		}
	}
}

func runCardinalityBomb(env string, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	wl, err := startWorkload(ctx, d, 5000, 500000, 0.05)
	if err != nil {
		return err
	}
	defer wl.Wait()

	for {
		select {
		case <-ctx.Done():
			return wl.Wait()
		case <-time.After(5 * time.Second):
			if safeModeActive() {
				return fmt.Errorf("safe mode activated")
			}
		}
	}
}

func runResourceStarvation(env string, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	wl, err := startWorkload(ctx, d, 10000, 10000, 0.9)
	if err != nil {
		return err
	}
	defer wl.Wait()

	for {
		select {
		case <-ctx.Done():
			return wl.Wait()
		case <-time.After(5 * time.Second):
			if safeModeActive() {
				return fmt.Errorf("safe mode activated")
			}
		}
	}
}

func runNetworkPartition(env string, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	wl, err := startWorkload(ctx, d, 5000, 5000, 0.05)
	if err != nil {
		return err
	}
	defer wl.Wait()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	disconnected := false
	for {
		select {
		case <-ctx.Done():
			return wl.Wait()
		case <-ticker.C:
			// Simulate network drop by disabling the priority_tagger processor
			if disconnected {
				_ = sendPatch("priority_tagger", "enabled", true)
			} else {
				_ = sendPatch("priority_tagger", "enabled", false)
			}
			disconnected = !disconnected

			if safeModeActive() {
				return fmt.Errorf("safe mode activated")
			}
		}
	}
}

func runOutOfMemory(env string, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	wl, err := startWorkload(ctx, d, 30000, 20000, 1.0)
	if err != nil {
		return err
	}
	defer wl.Wait()

	for {
		select {
		case <-ctx.Done():
			return wl.Wait()
		case <-time.After(5 * time.Second):
			if safeModeActive() {
				return fmt.Errorf("safe mode activated")
			}
		}
	}
}

// startWorkload runs the workload generator with the given parameters.
func startWorkload(ctx context.Context, d time.Duration, processes, cardinality int, spike float64) (*exec.Cmd, error) {
	args := []string{
		"run", "test/generator/workload.go",
		"--processes", fmt.Sprint(processes),
		"--cardinality", fmt.Sprint(cardinality),
		"--spike-freq", fmt.Sprintf("%.2f", spike),
		"--duration", d.String(),
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// sendPatch invokes the patch client to submit a configuration patch.
func sendPatch(target, param string, value any) error {
	val := fmt.Sprint(value)
	cmd := exec.Command("go", "run", "test/clients/patch_client.go",
		"--target", target, "--param", param, "--value", val)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

// safeModeActive checks the metrics endpoint for the safe mode gauge.
func safeModeActive() bool {
	resp, err := http.Get("http://localhost:8888/metrics")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return bytes.Contains(body, []byte("aemf_anom_safe_mode_active 1"))
}
