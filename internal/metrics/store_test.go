package metrics

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := t.TempDir() + "/metrics.db"
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE query_log (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp       TEXT NOT NULL DEFAULT (datetime('now')),
			operation_type  TEXT,
			operation_name  TEXT,
			duration_ms     INTEGER NOT NULL,
			success         INTEGER NOT NULL,
			error_message   TEXT,
			services_called TEXT NOT NULL DEFAULT '[]'
		)
	`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(path)
	})
	return db
}

func TestRecordAndSummary(t *testing.T) {
	s := New(openTestDB(t))

	entries := []*QueryEntry{
		{OperationType: "query", OperationName: "GetUser", DurationMs: 10, Success: true, ServicesCalled: []string{"users"}},
		{OperationType: "query", OperationName: "GetUser", DurationMs: 20, Success: true, ServicesCalled: []string{"users"}},
		{OperationType: "query", OperationName: "GetUser", DurationMs: 50, Success: false, ErrorMessage: "not found", ServicesCalled: []string{"users"}},
		{OperationType: "mutation", OperationName: "CreateOrder", DurationMs: 80, Success: true, ServicesCalled: []string{"orders"}},
		{OperationType: "mutation", OperationName: "CreateOrder", DurationMs: 100, Success: false, ErrorMessage: "timeout", ServicesCalled: []string{"orders"}},
	}

	for _, e := range entries {
		if err := s.RecordQuery(e); err != nil {
			t.Fatal(err)
		}
	}

	sum, err := s.Summary(10)
	if err != nil {
		t.Fatal(err)
	}

	if sum.TotalQueries != 5 {
		t.Errorf("expected 5 total queries, got %d", sum.TotalQueries)
	}
	if sum.ErrorCount != 2 {
		t.Errorf("expected 2 errors, got %d", sum.ErrorCount)
	}
	if sum.SuccessCount != 3 {
		t.Errorf("expected 3 successes, got %d", sum.SuccessCount)
	}
	if sum.ErrorRate < 0.39 || sum.ErrorRate > 0.41 {
		t.Errorf("expected error rate ~0.4, got %f", sum.ErrorRate)
	}
	if sum.LatencyP50Ms == 0 {
		t.Error("expected non-zero p50 latency")
	}
	if len(sum.Operations) == 0 {
		t.Error("expected at least one operation stat")
	}
	if len(sum.RecentErrors) != 2 {
		t.Errorf("expected 2 recent errors, got %d", len(sum.RecentErrors))
	}
}

func TestPercentile(t *testing.T) {
	cases := []struct {
		data []int64
		p    int
		want int64
	}{
		{[]int64{10, 20, 30, 40, 50}, 50, 30},
		{[]int64{10, 20, 30, 40, 50}, 100, 50},
		{[]int64{10, 20, 30, 40, 50}, 0, 10},
		{nil, 50, 0},
	}
	for _, c := range cases {
		got := percentile(c.data, c.p)
		if got != c.want {
			t.Errorf("percentile(%v, %d) = %d, want %d", c.data, c.p, got, c.want)
		}
	}
}
