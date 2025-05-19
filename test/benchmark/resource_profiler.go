package benchmark

import (
	"os"
	"runtime/pprof"
	"testing"
)

// startCPUProfile starts CPU profiling when the BENCH_PROFILE environment
// variable is set. It returns a function that should be deferred to stop
// the profile.
func startCPUProfile(b *testing.B, name string) func() {
	if os.Getenv("BENCH_PROFILE") == "" {
		return func() {}
	}
	f, err := os.Create(name)
	if err != nil {
		b.Fatalf("failed to create profile: %v", err)
	}
	pprof.StartCPUProfile(f)
	return func() {
		pprof.StopCPUProfile()
		f.Close()
	}
}
