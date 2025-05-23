package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/phoenix-vnext/analytics/internal/analyzer"
	"github.com/phoenix-vnext/analytics/internal/visualizer"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/sirupsen/logrus"
	"gonum.org/v1/plot/plotter"
)

type Handler struct {
	promClient      v1.API
	trendAnalyzer   *analyzer.TrendAnalyzer
	corrAnalyzer    *analyzer.CorrelationAnalyzer
	chartGenerator  *visualizer.ChartGenerator
	logger          *logrus.Logger
}

type TrendRequest struct {
	Metric    string `json:"metric"`
	Duration  string `json:"duration"`
	WindowSize int    `json:"window_size,omitempty"`
}

type CorrelationRequest struct {
	Metrics    []string `json:"metrics"`
	Duration   string   `json:"duration"`
	MinSamples int      `json:"min_samples,omitempty"`
}

type VisualizationRequest struct {
	Type     string                 `json:"type"` // "timeseries", "heatmap", "scatter", "histogram"
	Query    string                 `json:"query"`
	Duration string                 `json:"duration"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

func NewHandler(promAddr string, logger *logrus.Logger) (*Handler, error) {
	client, err := api.NewClient(api.Config{
		Address: promAddr,
	})
	if err != nil {
		return nil, err
	}

	return &Handler{
		promClient:     v1.NewAPI(client),
		trendAnalyzer:  analyzer.NewTrendAnalyzer(100),
		corrAnalyzer:   analyzer.NewCorrelationAnalyzer(30),
		chartGenerator: visualizer.NewChartGenerator(10, 6),
		logger:         logger,
	}, nil
}

func (h *Handler) AnalyzeTrend(w http.ResponseWriter, r *http.Request) {
	var req TrendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.WindowSize == 0 {
		req.WindowSize = 100
	}

	// Query Prometheus
	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		h.respondError(w, "invalid duration", http.StatusBadRequest)
		return
	}

	end := time.Now()
	start := end.Add(-duration)
	
	result, _, err := h.promClient.QueryRange(r.Context(), req.Metric, v1.Range{
		Start: start,
		End:   end,
		Step:  time.Minute,
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to query prometheus")
		h.respondError(w, "query failed", http.StatusInternalServerError)
		return
	}

	// Convert to data points
	dataPoints := h.convertToDataPoints(result)
	if len(dataPoints) == 0 {
		h.respondError(w, "no data found", http.StatusNotFound)
		return
	}

	// Analyze trend
	h.trendAnalyzer = analyzer.NewTrendAnalyzer(req.WindowSize)
	trendResult := h.trendAnalyzer.AnalyzeTrend(dataPoints)

	// Detect anomalies
	anomalies := h.trendAnalyzer.DetectAnomaly(dataPoints, 3.0)
	
	response := map[string]interface{}{
		"trend":      trendResult,
		"anomalies":  anomalies,
		"data_points": len(dataPoints),
	}

	h.respondJSON(w, response)
}

func (h *Handler) AnalyzeCorrelations(w http.ResponseWriter, r *http.Request) {
	var req CorrelationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.MinSamples == 0 {
		req.MinSamples = 30
	}

	// Query each metric
	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		h.respondError(w, "invalid duration", http.StatusBadRequest)
		return
	}

	end := time.Now()
	start := end.Add(-duration)
	
	metricData := []analyzer.MetricData{}
	for _, metric := range req.Metrics {
		result, _, err := h.promClient.QueryRange(r.Context(), metric, v1.Range{
			Start: start,
			End:   end,
			Step:  time.Minute,
		})
		if err != nil {
			h.logger.WithError(err).WithField("metric", metric).Error("failed to query metric")
			continue
		}

		dataPoints := h.convertToDataPoints(result)
		if len(dataPoints) > 0 {
			values := make([]float64, len(dataPoints))
			for i, dp := range dataPoints {
				values[i] = dp.Value
			}
			metricData = append(metricData, analyzer.MetricData{
				Name:   metric,
				Values: values,
			})
		}
	}

	if len(metricData) < 2 {
		h.respondError(w, "insufficient metrics with data", http.StatusBadRequest)
		return
	}

	// Analyze correlations
	h.corrAnalyzer = analyzer.NewCorrelationAnalyzer(req.MinSamples)
	correlations := h.corrAnalyzer.AnalyzeCorrelations(metricData)

	h.respondJSON(w, correlations)
}

func (h *Handler) GenerateVisualization(w http.ResponseWriter, r *http.Request) {
	var req VisualizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "invalid request", http.StatusBadRequest)
		return
	}

	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		h.respondError(w, "invalid duration", http.StatusBadRequest)
		return
	}

	end := time.Now()
	start := end.Add(-duration)

	// Query data
	result, _, err := h.promClient.QueryRange(r.Context(), req.Query, v1.Range{
		Start: start,
		End:   end,
		Step:  time.Minute,
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to query prometheus")
		h.respondError(w, "query failed", http.StatusInternalServerError)
		return
	}

	var imageData []byte

	switch req.Type {
	case "timeseries":
		imageData, err = h.generateTimeSeriesChart(result, req.Options)
	case "histogram":
		imageData, err = h.generateHistogram(result, req.Options)
	case "scatter":
		// Would need two queries for scatter plot
		h.respondError(w, "scatter plot requires correlation endpoint", http.StatusBadRequest)
		return
	default:
		h.respondError(w, "unsupported visualization type", http.StatusBadRequest)
		return
	}

	if err != nil {
		h.logger.WithError(err).Error("failed to generate visualization")
		h.respondError(w, "visualization generation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(imageData)))
	w.Write(imageData)
}

func (h *Handler) generateTimeSeriesChart(result v1.Value, options map[string]interface{}) ([]byte, error) {
	title := "Time Series"
	if t, ok := options["title"].(string); ok {
		title = t
	}

	// Convert result to time series data
	seriesData := []visualizer.TimeSeriesData{}
	
	switch v := result.(type) {
	case v1.Matrix:
		for _, series := range v {
			points := make([]visualizer.TimePoint, len(series.Values))
			for i, sample := range series.Values {
				points[i] = visualizer.TimePoint{
					Time:  sample.Timestamp.Time(),
					Value: float64(sample.Value),
				}
			}
			seriesData = append(seriesData, visualizer.TimeSeriesData{
				Label:  series.Metric.String(),
				Points: points,
			})
		}
	}

	return h.chartGenerator.GenerateTimeSeriesChart(title, seriesData)
}

func (h *Handler) generateHistogram(result v1.Value, options map[string]interface{}) ([]byte, error) {
	title := "Histogram"
	if t, ok := options["title"].(string); ok {
		title = t
	}

	bins := 20
	if b, ok := options["bins"].(float64); ok {
		bins = int(b)
	}

	// Extract values
	values := plotter.Values{}
	
	switch v := result.(type) {
	case v1.Matrix:
		for _, series := range v {
			for _, sample := range series.Values {
				values = append(values, float64(sample.Value))
			}
		}
	}

	return h.chartGenerator.GenerateHistogram(title, "Value", "Frequency", values, bins)
}

func (h *Handler) convertToDataPoints(result v1.Value) []analyzer.DataPoint {
	dataPoints := []analyzer.DataPoint{}
	
	switch v := result.(type) {
	case v1.Matrix:
		for _, series := range v {
			for _, sample := range series.Values {
				dataPoints = append(dataPoints, analyzer.DataPoint{
					Timestamp: sample.Timestamp.Time(),
					Value:     float64(sample.Value),
				})
			}
		}
	}
	
	return dataPoints
}

func (h *Handler) respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) respondError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}