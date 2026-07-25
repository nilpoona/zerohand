package aggregator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nilpoona/zerohand/internal/executor"
)

func TestCalculateStats(t *testing.T) {
	tests := []struct {
		name     string
		results  []*executor.Result
		validate func(t *testing.T, stats *Stats)
	}{
		{
			name:    "empty results",
			results: []*executor.Result{},
			validate: func(t *testing.T, stats *Stats) {
				if stats.TotalRequests != 0 {
					t.Errorf("expected 0 total requests, got %d", stats.TotalRequests)
				}
			},
		},
		{
			name: "all successful requests",
			results: []*executor.Result{
				{Timestamp: time.Now(), StatusCode: 200, Duration: 10000, Error: ""},
				{Timestamp: time.Now(), StatusCode: 200, Duration: 20000, Error: ""},
				{Timestamp: time.Now(), StatusCode: 201, Duration: 30000, Error: ""},
			},
			validate: func(t *testing.T, stats *Stats) {
				if stats.TotalRequests != 3 {
					t.Errorf("expected 3 total requests, got %d", stats.TotalRequests)
				}
				if stats.Request.SuccessCount != 3 {
					t.Errorf("expected 3 successful requests, got %d", stats.Request.SuccessCount)
				}
				if stats.Request.FailureCount != 0 {
					t.Errorf("expected 0 failed requests, got %d", stats.Request.FailureCount)
				}
				if stats.Request.SuccessRate != 100.0 {
					t.Errorf("expected 100%% success rate, got %.2f", stats.Request.SuccessRate)
				}
				if stats.Response.AvgDuration != 20.0 {
					t.Errorf("expected 20ms avg duration, got %.2f", stats.Response.AvgDuration)
				}
			},
		},
		{
			name: "mixed success and failure",
			results: []*executor.Result{
				{Timestamp: time.Now(), StatusCode: 200, Duration: 10000, Error: ""},
				{Timestamp: time.Now(), StatusCode: 500, Duration: 20000, Error: ""},
				{Timestamp: time.Now(), StatusCode: 0, Duration: 15000, Error: "timeout"},
			},
			validate: func(t *testing.T, stats *Stats) {
				if stats.TotalRequests != 3 {
					t.Errorf("expected 3 total requests, got %d", stats.TotalRequests)
				}
				if stats.Request.SuccessCount != 1 {
					t.Errorf("expected 1 successful request, got %d", stats.Request.SuccessCount)
				}
				if stats.Request.FailureCount != 2 {
					t.Errorf("expected 2 failed requests, got %d", stats.Request.FailureCount)
				}
				if stats.Request.TimeoutCount != 1 {
					t.Errorf("expected 1 timeout, got %d", stats.Request.TimeoutCount)
				}
			},
		},
		{
			name: "error distribution",
			results: []*executor.Result{
				{Timestamp: time.Now(), StatusCode: 0, Duration: 10000, Error: "timeout"},
				{Timestamp: time.Now(), StatusCode: 0, Duration: 10000, Error: "timeout"},
				{Timestamp: time.Now(), StatusCode: 0, Duration: 10000, Error: "dns_error"},
			},
			validate: func(t *testing.T, stats *Stats) {
				if stats.ErrorDist["timeout"] != 2 {
					t.Errorf("expected 2 timeout errors, got %d", stats.ErrorDist["timeout"])
				}
				if stats.ErrorDist["dns_error"] != 1 {
					t.Errorf("expected 1 dns_error, got %d", stats.ErrorDist["dns_error"])
				}
			},
		},
		{
			name: "status code distribution",
			results: []*executor.Result{
				{Timestamp: time.Now(), StatusCode: 200, Duration: 10000, Error: ""},
				{Timestamp: time.Now(), StatusCode: 200, Duration: 10000, Error: ""},
				{Timestamp: time.Now(), StatusCode: 404, Duration: 10000, Error: ""},
				{Timestamp: time.Now(), StatusCode: 500, Duration: 10000, Error: ""},
			},
			validate: func(t *testing.T, stats *Stats) {
				if stats.StatusCodeDist[200] != 2 {
					t.Errorf("expected 2 200 status codes, got %d", stats.StatusCodeDist[200])
				}
				if stats.StatusCodeDist[404] != 1 {
					t.Errorf("expected 1 404 status code, got %d", stats.StatusCodeDist[404])
				}
				if stats.StatusCodeDist[500] != 1 {
					t.Errorf("expected 1 500 status code, got %d", stats.StatusCodeDist[500])
				}
			},
		},
		{
			name: "actual RPS calculation",
			results: []*executor.Result{
				{Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), StatusCode: 200, Duration: 10000, Error: ""},
				{Timestamp: time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC), StatusCode: 200, Duration: 10000, Error: ""},
				{Timestamp: time.Date(2024, 1, 1, 0, 0, 2, 0, time.UTC), StatusCode: 200, Duration: 10000, Error: ""},
				{Timestamp: time.Date(2024, 1, 1, 0, 0, 3, 0, time.UTC), StatusCode: 200, Duration: 10000, Error: ""},
				{Timestamp: time.Date(2024, 1, 1, 0, 0, 4, 0, time.UTC), StatusCode: 200, Duration: 10000, Error: ""},
			},
			validate: func(t *testing.T, stats *Stats) {
				// 5 requests over 4 seconds = 1.25 RPS
				expectedRPS := 1.25
				if stats.ActualRPS != expectedRPS {
					t.Errorf("expected %.2f RPS, got %.2f", expectedRPS, stats.ActualRPS)
				}
				expectedDuration := 4.0
				if stats.TestDuration != expectedDuration {
					t.Errorf("expected %.2fs duration, got %.2f", expectedDuration, stats.TestDuration)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := CalculateStats(tt.results)
			if stats == nil {
				t.Fatal("expected stats, got nil")
			}
			tt.validate(t, stats)
		})
	}
}

func TestCalculatePercentile(t *testing.T) {
	tests := []struct {
		name       string
		durations  []int64
		percentile float64
		expected   float64
	}{
		{
			name:       "empty durations",
			durations:  []int64{},
			percentile: 0.5,
			expected:   0,
		},
		{
			name:       "single value",
			durations:  []int64{10000},
			percentile: 0.5,
			expected:   10.0,
		},
		{
			name:       "median of odd count",
			durations:  []int64{10000, 20000, 30000},
			percentile: 0.5,
			expected:   20.0,
		},
		{
			name:       "median of even count",
			durations:  []int64{10000, 20000, 30000, 40000},
			percentile: 0.5,
			expected:   20.0,
		},
		{
			name:       "p95",
			durations:  []int64{10000, 20000, 30000, 40000, 50000},
			percentile: 0.95,
			expected:   40.0,
		},
		{
			name:       "p99",
			durations:  []int64{10000, 20000, 30000, 40000, 50000},
			percentile: 0.99,
			expected:   40.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePercentile(tt.durations, tt.percentile)
			if result != tt.expected {
				t.Errorf("expected %.2f, got %.2f", tt.expected, result)
			}
		})
	}
}

func TestResponseStats(t *testing.T) {
	results := []*executor.Result{
		{Timestamp: time.Now(), StatusCode: 200, Duration: 5000, Error: ""},
		{Timestamp: time.Now(), StatusCode: 200, Duration: 10000, Error: ""},
		{Timestamp: time.Now(), StatusCode: 200, Duration: 15000, Error: ""},
		{Timestamp: time.Now(), StatusCode: 200, Duration: 20000, Error: ""},
		{Timestamp: time.Now(), StatusCode: 200, Duration: 100000, Error: ""},
	}

	stats := CalculateStats(results)

	if stats.Response.MinDuration != 5.0 {
		t.Errorf("expected min duration 5.0ms, got %.2f", stats.Response.MinDuration)
	}

	if stats.Response.MaxDuration != 100.0 {
		t.Errorf("expected max duration 100.0ms, got %.2f", stats.Response.MaxDuration)
	}

	expectedAvg := (5.0 + 10.0 + 15.0 + 20.0 + 100.0) / 5.0
	if stats.Response.AvgDuration != expectedAvg {
		t.Errorf("expected avg duration %.2fms, got %.2f", expectedAvg, stats.Response.AvgDuration)
	}
}

func TestFormatStats(t *testing.T) {
	stats := &Stats{
		TotalRequests: 100,
		ActualRPS:     10.5,
		TestDuration:  9.52,
		Request: RequestStats{
			SuccessCount: 95,
			FailureCount: 5,
			SuccessRate:  95.0,
			FailureRate:  5.0,
			TimeoutCount: 2,
		},
		Response: ResponseStats{
			AvgDuration:    45.5,
			MinDuration:    10.0,
			MedianDuration: 42.0,
			P95Duration:    78.0,
			P99Duration:    120.0,
			MaxDuration:    150.0,
		},
		StatusCodeDist: map[int]int{
			200: 95,
			500: 3,
		},
		ErrorDist: map[string]int{
			"timeout": 2,
		},
	}

	result := FormatStats(stats)

	expectedStrings := []string{
		"Total Requests:  100",
		"Test Duration:   9.52s",
		"Actual RPS:      10.50",
		"Success:         95 (95.00%)",
		"Failure:         5 (5.00%)",
		"Average:  45.50ms",
		"Min:      10.00ms",
		"Median:   42.00ms",
		"P95:      78.00ms",
		"P99:      120.00ms",
		"Max:      150.00ms",
		"200: 95",
		"500: 3",
		"timeout: 2",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("expected output to contain %q, but it didn't.\nGot: %s", expected, result)
		}
	}
}

func TestSaveResultsJSON(t *testing.T) {
	tempDir := t.TempDir()
	filepath := filepath.Join(tempDir, "results.json")

	results := []*executor.Result{
		{
			Timestamp:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			StatusCode: 200,
			Duration:   10000,
			Error:      "",
		},
		{
			Timestamp:  time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
			StatusCode: 500,
			Duration:   20000,
			Error:      "",
		},
	}

	err := SaveResultsJSON(results, filepath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		t.Error("expected file to be created")
	}

	content, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "200") {
		t.Error("expected JSON to contain status code 200")
	}
	if !strings.Contains(contentStr, "500") {
		t.Error("expected JSON to contain status code 500")
	}
}

func TestSaveResultsJSONInvalidPath(t *testing.T) {
	results := []*executor.Result{
		{
			Timestamp:  time.Now(),
			StatusCode: 200,
			Duration:   10000,
			Error:      "",
		},
	}

	err := SaveResultsJSON(results, "/invalid/path/results.json")
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}
