package pic_control_ext

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/pkg/policy"
)

// loadPolicy loads the policy file
func (e *Extension) loadPolicy(path string) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	// For testing, use the policy file from the config if specified
	if e.config.PolicyFile != "" {
		path = e.config.PolicyFile
	}

	// Check if file exists
	_, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("policy file not found: %w", err)
	}

	// Load policy
	p, err := policy.LoadPolicy(path)
	if err != nil {
		return errPolicyLoadFailed
	}

	e.policy = p
	e.logger.Info("Loaded policy", zap.String("path", path))

	// Add metrics for policy load
	if e.emitMetric != nil {
		e.emitMetric("aemf_policy_loaded_total", 1.0)
	}

	return nil
}

// startWatcher sets up filesystem watching of the policy file
func (e *Extension) startWatcher() error {
	// For testing, skip watching if disabled
	if !e.config.WatchPolicy {
		return nil
	}

	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	e.watcher = watcher

	// Add policy file to watch
	policePath := e.config.PolicyFilePath
	if e.config.PolicyFile != "" {
		policePath = e.config.PolicyFile
	}
	err = watcher.Add(policePath)
	if err != nil {
		return fmt.Errorf("adding policy file to watcher: %w", err)
	}

	// Start watch goroutine
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelWatcher = cancel

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					e.logger.Info("Policy file modified, reloading", zap.String("path", event.Name))
					// Wait a moment for the file to finish being written
					time.Sleep(100 * time.Millisecond)
					err := e.loadPolicy(event.Name)
					if err != nil {
						e.logger.Error("Failed to reload policy", zap.Error(err))
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				e.logger.Error("Policy watcher error", zap.Error(err))
			case <-ctx.Done():
				return
			}
		}
	}()

	e.logger.Info("Started policy file watcher", zap.String("path", policePath))
	return nil
}