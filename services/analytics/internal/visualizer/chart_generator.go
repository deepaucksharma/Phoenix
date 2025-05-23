package visualizer

import (
	"fmt"
	"image/color"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

type ChartGenerator struct {
	width  vg.Length
	height vg.Length
}

type TimeSeriesData struct {
	Label  string
	Points []TimePoint
}

type TimePoint struct {
	Time  time.Time
	Value float64
}

func NewChartGenerator(width, height float64) *ChartGenerator {
	return &ChartGenerator{
		width:  vg.Length(width) * vg.Inch,
		height: vg.Length(height) * vg.Inch,
	}
}

func (cg *ChartGenerator) GenerateTimeSeriesChart(title string, data []TimeSeriesData) ([]byte, error) {
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = "Time"
	p.Y.Label.Text = "Value"
	p.X.Tick.Marker = plot.TimeTicks{Format: "15:04"}

	colors := []color.Color{
		color.RGBA{R: 31, G: 119, B: 180, A: 255},
		color.RGBA{R: 255, G: 127, B: 14, A: 255},
		color.RGBA{R: 44, G: 160, B: 44, A: 255},
		color.RGBA{R: 214, G: 39, B: 40, A: 255},
		color.RGBA{R: 148, G: 103, B: 189, A: 255},
	}

	for i, series := range data {
		pts := make(plotter.XYs, len(series.Points))
		for j, point := range series.Points {
			pts[j].X = float64(point.Time.Unix())
			pts[j].Y = point.Value
		}

		line, err := plotter.NewLine(pts)
		if err != nil {
			return nil, fmt.Errorf("failed to create line: %w", err)
		}
		line.Color = colors[i%len(colors)]
		line.Width = vg.Points(2)

		p.Add(line)
		p.Legend.Add(series.Label, line)
	}

	writer, err := p.WriterTo(cg.width, cg.height, "png")
	if err != nil {
		return nil, fmt.Errorf("failed to create writer: %w", err)
	}

	buf := make([]byte, 0)
	_, err = writer.WriteTo(&byteWriter{&buf})
	if err != nil {
		return nil, fmt.Errorf("failed to write chart: %w", err)
	}

	return buf, nil
}

func (cg *ChartGenerator) GenerateHeatmap(title string, data [][]float64, xLabels, yLabels []string) ([]byte, error) {
	p := plot.New()
	p.Title.Text = title
	
	// Create grid data
	grid := plotter.NewGridXYZ(plotter.GridXYZ{
		X: make([]float64, len(xLabels)),
		Y: make([]float64, len(yLabels)),
		Z: data,
	})

	for i := range xLabels {
		grid.X[i] = float64(i)
	}
	for i := range yLabels {
		grid.Y[i] = float64(i)
	}

	heatmap := plotter.NewHeatMap(grid, nil)
	p.Add(heatmap)

	// Add custom tick labels
	p.X.Tick.Marker = stringTicks{
		labels: xLabels,
		values: grid.X,
	}
	p.Y.Tick.Marker = stringTicks{
		labels: yLabels,
		values: grid.Y,
	}

	writer, err := p.WriterTo(cg.width, cg.height, "png")
	if err != nil {
		return nil, fmt.Errorf("failed to create writer: %w", err)
	}

	buf := make([]byte, 0)
	_, err = writer.WriteTo(&byteWriter{&buf})
	if err != nil {
		return nil, fmt.Errorf("failed to write heatmap: %w", err)
	}

	return buf, nil
}

func (cg *ChartGenerator) GenerateScatterPlot(title, xLabel, yLabel string, points plotter.XYs) ([]byte, error) {
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel

	scatter, err := plotter.NewScatter(points)
	if err != nil {
		return nil, fmt.Errorf("failed to create scatter plot: %w", err)
	}
	scatter.GlyphStyle.Color = color.RGBA{R: 31, G: 119, B: 180, A: 255}
	scatter.GlyphStyle.Radius = vg.Points(3)

	p.Add(scatter)
	p.Add(plotter.NewGrid())

	// Add trend line
	line, err := plotter.NewLine(points)
	if err == nil {
		line.Color = color.RGBA{R: 255, G: 0, B: 0, A: 128}
		line.Width = vg.Points(1)
		p.Add(line)
	}

	writer, err := p.WriterTo(cg.width, cg.height, "png")
	if err != nil {
		return nil, fmt.Errorf("failed to create writer: %w", err)
	}

	buf := make([]byte, 0)
	_, err = writer.WriteTo(&byteWriter{&buf})
	if err != nil {
		return nil, fmt.Errorf("failed to write scatter plot: %w", err)
	}

	return buf, nil
}

func (cg *ChartGenerator) GenerateHistogram(title, xLabel, yLabel string, values plotter.Values, bins int) ([]byte, error) {
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel

	hist, err := plotter.NewHist(values, bins)
	if err != nil {
		return nil, fmt.Errorf("failed to create histogram: %w", err)
	}
	hist.FillColor = color.RGBA{R: 31, G: 119, B: 180, A: 255}
	hist.LineStyle.Width = vg.Points(0)

	p.Add(hist)
	p.Add(plotter.NewGrid())

	writer, err := p.WriterTo(cg.width, cg.height, "png")
	if err != nil {
		return nil, fmt.Errorf("failed to create writer: %w", err)
	}

	buf := make([]byte, 0)
	_, err = writer.WriteTo(&byteWriter{&buf})
	if err != nil {
		return nil, fmt.Errorf("failed to write histogram: %w", err)
	}

	return buf, nil
}

// Helper types

type byteWriter struct {
	buf *[]byte
}

func (w *byteWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

type stringTicks struct {
	labels []string
	values []float64
}

func (st stringTicks) Ticks(min, max float64) []plot.Tick {
	ticks := make([]plot.Tick, len(st.labels))
	for i, label := range st.labels {
		ticks[i] = plot.Tick{
			Value: st.values[i],
			Label: label,
		}
	}
	return ticks
}