package timeseries

// ForecastEMA performs exponential moving average forecasting over the input
// data. Alpha controls the smoothing factor and steps determines how many
// additional points to forecast beyond the input series.
func ForecastEMA(data []float64, alpha float64, steps int) []float64 {
	if alpha <= 0 || alpha >= 1 {
		alpha = 0.5
	}
	if steps < 0 {
		steps = 0
	}
	out := make([]float64, len(data)+steps)
	if len(data) == 0 {
		return out
	}
	out[0] = data[0]
	for i := 1; i < len(data); i++ {
		out[i] = alpha*data[i] + (1-alpha)*out[i-1]
	}
	last := out[len(data)-1]
	for i := 0; i < steps; i++ {
		last = alpha*last + (1-alpha)*last
		out[len(data)+i] = last
	}
	return out
}
