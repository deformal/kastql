package metrics

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

// Store writes and queries the query_log table in metrics.db.
type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// QueryEntry is one recorded GraphQL operation.
type QueryEntry struct {
	OperationType  string
	OperationName  string
	DurationMs     int64
	Success        bool
	ErrorMessage   string
	ServicesCalled []string
}

// RecordQuery appends a log row to query_log.
func (s *Store) RecordQuery(e *QueryEntry) error {
	svcJSON, _ := json.Marshal(e.ServicesCalled)
	_, err := s.db.Exec(`
		INSERT INTO query_log (operation_type, operation_name, duration_ms, success, error_message, services_called)
		VALUES (?, ?, ?, ?, ?, ?)
	`, nullStr(e.OperationType), nullStr(e.OperationName), e.DurationMs, boolInt(e.Success), nullStr(e.ErrorMessage), string(svcJSON))
	return err
}

// ── Summary types ─────────────────────────────────────────────────────────────

type Summary struct {
	TotalQueries int64          `json:"total_queries"`
	SuccessCount int64          `json:"success_count"`
	ErrorCount   int64          `json:"error_count"`
	ErrorRate    float64        `json:"error_rate"`
	LatencyP50Ms int64          `json:"latency_p50_ms"`
	LatencyP95Ms int64          `json:"latency_p95_ms"`
	LatencyP99Ms int64          `json:"latency_p99_ms"`
	Operations   []*OpStat      `json:"operations"`
	RecentErrors []*RecentError `json:"recent_errors"`
}

type OpStat struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Count      int64   `json:"count"`
	ErrorCount int64   `json:"error_count"`
	AvgMs      float64 `json:"avg_ms"`
}

type RecentError struct {
	Timestamp     time.Time `json:"timestamp"`
	OperationType string    `json:"operation_type"`
	OperationName string    `json:"operation_name"`
	DurationMs    int64     `json:"duration_ms"`
	Message       string    `json:"message"`
}

// Summary returns aggregated metrics. limit caps the number of recent errors returned.
func (s *Store) Summary(limit int) (*Summary, error) {
	if limit <= 0 {
		limit = 20
	}

	totals, err := s.totals()
	if err != nil {
		return nil, fmt.Errorf("totals: %w", err)
	}

	durations, err := s.allDurations()
	if err != nil {
		return nil, fmt.Errorf("durations: %w", err)
	}

	ops, err := s.opStats()
	if err != nil {
		return nil, fmt.Errorf("op stats: %w", err)
	}

	recent, err := s.recentErrors(limit)
	if err != nil {
		return nil, fmt.Errorf("recent errors: %w", err)
	}

	sum := &Summary{
		TotalQueries: totals.total,
		SuccessCount: totals.success,
		ErrorCount:   totals.total - totals.success,
		Operations:   ops,
		RecentErrors: recent,
	}
	if totals.total > 0 {
		sum.ErrorRate = float64(sum.ErrorCount) / float64(totals.total)
	}
	sum.LatencyP50Ms = percentile(durations, 50)
	sum.LatencyP95Ms = percentile(durations, 95)
	sum.LatencyP99Ms = percentile(durations, 99)

	return sum, nil
}

// ── internal queries ──────────────────────────────────────────────────────────

type totalsRow struct{ total, success int64 }

func (s *Store) totals() (totalsRow, error) {
	var r totalsRow
	err := s.db.QueryRow(`SELECT COUNT(*), SUM(success) FROM query_log`).Scan(&r.total, &r.success)
	return r, err
}

func (s *Store) allDurations() ([]int64, error) {
	rows, err := s.db.Query(`SELECT duration_ms FROM query_log ORDER BY duration_ms`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ds []int64
	for rows.Next() {
		var d int64
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		ds = append(ds, d)
	}
	return ds, rows.Err()
}

func (s *Store) opStats() ([]*OpStat, error) {
	rows, err := s.db.Query(`
		SELECT
			COALESCE(operation_name, ''),
			COALESCE(operation_type, ''),
			COUNT(*),
			SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END),
			AVG(duration_ms)
		FROM query_log
		GROUP BY operation_name, operation_type
		ORDER BY COUNT(*) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ops []*OpStat
	for rows.Next() {
		var o OpStat
		if err := rows.Scan(&o.Name, &o.Type, &o.Count, &o.ErrorCount, &o.AvgMs); err != nil {
			return nil, err
		}
		ops = append(ops, &o)
	}
	return ops, rows.Err()
}

func (s *Store) recentErrors(limit int) ([]*RecentError, error) {
	rows, err := s.db.Query(`
		SELECT timestamp, COALESCE(operation_type,''), COALESCE(operation_name,''),
		       duration_ms, COALESCE(error_message,'')
		FROM query_log
		WHERE success = 0
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var errs []*RecentError
	for rows.Next() {
		var e RecentError
		var ts string
		if err := rows.Scan(&ts, &e.OperationType, &e.OperationName, &e.DurationMs, &e.Message); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		errs = append(errs, &e)
	}
	return errs, rows.Err()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func percentile(sorted []int64, p int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	slices.Sort(sorted)
	idx := int(float64(len(sorted)-1) * float64(p) / 100.0)
	return sorted[idx]
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
