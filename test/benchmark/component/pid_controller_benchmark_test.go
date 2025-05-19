package benchmark

import (
	"testing"

	"github.com/deepaucksharma/Phoenix/internal/control/pid"
)

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
