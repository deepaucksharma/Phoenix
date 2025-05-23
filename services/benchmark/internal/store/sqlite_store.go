package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore provides persistent storage for benchmark results
type SQLiteStore struct {
	db *sql.DB
}

type BenchmarkResult struct {
	ID               string                 `json:"id"`
	Scenario         string                 `json:"scenario"`
	StartTime        time.Time              `json:"start_time"`
	EndTime          time.Time              `json:"end_time"`
	Passed           bool                   `json:"passed"`
	Metrics          map[string]float64     `json:"metrics"`
	FailureReasons   []string               `json:"failure_reasons"`
	ResourceUsage    map[string]interface{} `json:"resource_usage"`
	ControlBehavior  []interface{}          `json:"control_behavior"`
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) initialize() error {
	schema := `
	CREATE TABLE IF NOT EXISTS benchmark_results (
		id TEXT PRIMARY KEY,
		scenario TEXT NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME NOT NULL,
		passed BOOLEAN NOT NULL,
		metrics TEXT NOT NULL,
		failure_reasons TEXT,
		resource_usage TEXT,
		control_behavior TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_scenario ON benchmark_results(scenario);
	CREATE INDEX IF NOT EXISTS idx_start_time ON benchmark_results(start_time);
	CREATE INDEX IF NOT EXISTS idx_passed ON benchmark_results(passed);

	CREATE TABLE IF NOT EXISTS benchmark_baselines (
		scenario TEXT PRIMARY KEY,
		metrics TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS performance_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		metric_name TEXT NOT NULL,
		metric_value REAL NOT NULL,
		pipeline TEXT,
		tags TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_perf_timestamp ON performance_history(timestamp);
	CREATE INDEX IF NOT EXISTS idx_perf_metric ON performance_history(metric_name);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStore) SaveResult(result *BenchmarkResult) error {
	metricsJSON, err := json.Marshal(result.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	failureReasonsJSON, err := json.Marshal(result.FailureReasons)
	if err != nil {
		return fmt.Errorf("failed to marshal failure reasons: %w", err)
	}

	resourceUsageJSON, err := json.Marshal(result.ResourceUsage)
	if err != nil {
		return fmt.Errorf("failed to marshal resource usage: %w", err)
	}

	controlBehaviorJSON, err := json.Marshal(result.ControlBehavior)
	if err != nil {
		return fmt.Errorf("failed to marshal control behavior: %w", err)
	}

	query := `
	INSERT INTO benchmark_results (
		id, scenario, start_time, end_time, passed, 
		metrics, failure_reasons, resource_usage, control_behavior
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		result.ID, result.Scenario, result.StartTime, result.EndTime, result.Passed,
		string(metricsJSON), string(failureReasonsJSON), 
		string(resourceUsageJSON), string(controlBehaviorJSON),
	)

	return err
}

func (s *SQLiteStore) GetResults(scenario string, limit int) ([]*BenchmarkResult, error) {
	query := `
	SELECT id, scenario, start_time, end_time, passed, 
		   metrics, failure_reasons, resource_usage, control_behavior
	FROM benchmark_results
	WHERE scenario = ? OR ? = ''
	ORDER BY start_time DESC
	LIMIT ?
	`

	rows, err := s.db.Query(query, scenario, scenario, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*BenchmarkResult
	for rows.Next() {
		result := &BenchmarkResult{}
		var metricsJSON, failureReasonsJSON, resourceUsageJSON, controlBehaviorJSON string

		err := rows.Scan(
			&result.ID, &result.Scenario, &result.StartTime, &result.EndTime, &result.Passed,
			&metricsJSON, &failureReasonsJSON, &resourceUsageJSON, &controlBehaviorJSON,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON fields
		json.Unmarshal([]byte(metricsJSON), &result.Metrics)
		json.Unmarshal([]byte(failureReasonsJSON), &result.FailureReasons)
		json.Unmarshal([]byte(resourceUsageJSON), &result.ResourceUsage)
		json.Unmarshal([]byte(controlBehaviorJSON), &result.ControlBehavior)

		results = append(results, result)
	}

	return results, nil
}

func (s *SQLiteStore) SaveBaseline(scenario string, metrics map[string]float64) error {
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal baseline metrics: %w", err)
	}

	query := `
	INSERT OR REPLACE INTO benchmark_baselines (scenario, metrics, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	`

	_, err = s.db.Exec(query, scenario, string(metricsJSON))
	return err
}

func (s *SQLiteStore) GetBaseline(scenario string) (map[string]float64, error) {
	var metricsJSON string
	query := `SELECT metrics FROM benchmark_baselines WHERE scenario = ?`
	
	err := s.db.QueryRow(query, scenario).Scan(&metricsJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No baseline exists
		}
		return nil, err
	}

	var metrics map[string]float64
	err = json.Unmarshal([]byte(metricsJSON), &metrics)
	return metrics, err
}

func (s *SQLiteStore) SavePerformanceMetric(timestamp time.Time, metricName string, value float64, pipeline string, tags map[string]string) error {
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
	INSERT INTO performance_history (timestamp, metric_name, metric_value, pipeline, tags)
	VALUES (?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query, timestamp, metricName, value, pipeline, string(tagsJSON))
	return err
}

func (s *SQLiteStore) GetPerformanceHistory(metricName string, since time.Time) ([]struct {
	Timestamp time.Time
	Value     float64
	Pipeline  string
}, error) {
	query := `
	SELECT timestamp, metric_value, pipeline
	FROM performance_history
	WHERE metric_name = ? AND timestamp >= ?
	ORDER BY timestamp ASC
	`

	rows, err := s.db.Query(query, metricName, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []struct {
		Timestamp time.Time
		Value     float64
		Pipeline  string
	}

	for rows.Next() {
		var h struct {
			Timestamp time.Time
			Value     float64
			Pipeline  string
		}
		err := rows.Scan(&h.Timestamp, &h.Value, &h.Pipeline)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}