package timeseries_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/deepaucksharma/Phoenix/pkg/util/timeseries"
)

func TestForecastEMA(t *testing.T) {
	data := []float64{1, 2, 3, 4}
	out := timeseries.ForecastEMA(data, 0.5, 2)
	assert.Equal(t, 6, len(out))
}

func TestDetectZScore(t *testing.T) {
	data := []float64{1, 2, 3, 100}
	idx := timeseries.DetectZScore(data, 1)
	assert.Equal(t, 1, len(idx))
	assert.Equal(t, 3, idx[0])
}
