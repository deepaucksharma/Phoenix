package benchmark

import (
	"os"
	"runtime/pprof"
	"testing"

	"github.com/deepaucksharma/Phoenix/internal/control/pid"
)

// Helper function to start CPU profiling
func startCPUProfile(b *testing.B, name string) func() {
	f, err := os.Create(name)
	if err != nil {
		b.Fatal(err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		b.Fatal(err)
	}
	return func() {
		pprof.StopCPUProfile()
		f.Close()
	}
}

func BenchmarkPIDControllerCompute(b *testing.B) {
	ctrl := pid.NewController(0.1, 0.01, 0.05, 10)
	stop := startCPUProfile(b, "pid_cpu.prof")
	defer stop()
	b.ReportAllocs()
	var in float64
	for i := 0; i < b.N; i++ {
		in += 0.1
		ctrl.Compute(in)
	}
}
