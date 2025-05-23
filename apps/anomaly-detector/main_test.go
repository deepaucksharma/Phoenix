package main

import (
	"context"
	"testing"
	"time"
)

func TestStatisticalDetectorDetect(t *testing.T) {
	detector := &StatisticalDetector{windowSize: 5, threshold: 2.0}
	now := time.Now()
	points := []DataPoint{}
	for i := 0; i < 5; i++ {
		points = append(points, DataPoint{Timestamp: now.Add(time.Duration(i) * time.Minute), Value: 100})
	}
	// anomaly point far from mean
	points = append(points, DataPoint{Timestamp: now.Add(6 * time.Minute), Value: 150})

	metrics := map[string][]DataPoint{"metric": points}
	anomalies, err := detector.Detect(context.Background(), metrics)
	if err != nil {
		t.Fatalf("detect returned error: %v", err)
	}
	if len(anomalies) == 0 {
		t.Fatalf("expected at least one anomaly")
	}
	if anomalies[0].DetectorName != detector.GetName() {
		t.Errorf("unexpected detector name %s", anomalies[0].DetectorName)
	}
}
