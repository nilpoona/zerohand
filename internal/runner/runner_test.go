package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/nilpoona/zerohand/internal/config"
)

func TestRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.LoadTestConfig{
		TargetURL: server.URL,
		Method:    "GET",
		RPS:       5,
		Duration:  2,
		Timeout:   10,
	}

	ctx := context.Background()
	results, err := Run(ctx, cfg, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedRequests := cfg.RPS * cfg.Duration
	if len(results) != expectedRequests {
		t.Errorf("expected %d results, got %d", expectedRequests, len(results))
	}

	for _, result := range results {
		if result == nil {
			t.Error("got nil result")
			continue
		}
		if result.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", result.StatusCode)
		}
	}
}

func TestRunWithProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.LoadTestConfig{
		TargetURL: server.URL,
		Method:    "GET",
		RPS:       5,
		Duration:  2,
		Timeout:   10,
	}

	ctx := context.Background()
	progressCh := make(chan Progress, 10)

	go func() {
		Run(ctx, cfg, progressCh)
	}()

	progressReceived := false

	timeout := time.After(5 * time.Second)
	for {
		select {
		case progress, ok := <-progressCh:
			if !ok {
				if !progressReceived {
					t.Error("no progress updates received")
				}
				return
			}
			progressReceived = true

			if progress.Progress < 0 || progress.Progress > 1 {
				t.Errorf("invalid progress value: %f", progress.Progress)
			}
			if progress.Completed > progress.Total {
				t.Errorf("completed (%d) > total (%d)", progress.Completed, progress.Total)
			}

		case <-timeout:
			t.Fatal("test timeout")
		}
	}
}

func TestRunWithCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.LoadTestConfig{
		TargetURL: server.URL,
		Method:    "GET",
		RPS:       10,
		Duration:  10,
		Timeout:   10,
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	results, err := Run(ctx, cfg, nil)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}

	expectedRequests := cfg.RPS * cfg.Duration
	if len(results) >= expectedRequests {
		t.Errorf("expected fewer than %d results due to cancellation, got %d", expectedRequests, len(results))
	}
}

func TestRunWithInvalidConfig(t *testing.T) {
	cfg := &config.LoadTestConfig{
		TargetURL: "",
		Method:    "GET",
		RPS:       10,
		Duration:  1,
		Timeout:   10,
	}

	ctx := context.Background()
	results, err := Run(ctx, cfg, nil)

	if err == nil {
		t.Error("expected error for invalid config, got nil")
	}

	if results != nil {
		t.Errorf("expected nil results on error, got %d results", len(results))
	}
}

func TestRunRPSTiming(t *testing.T) {
	requestTimes := make([]time.Time, 0)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.LoadTestConfig{
		TargetURL: server.URL,
		Method:    "GET",
		RPS:       10,
		Duration:  1,
		Timeout:   10,
	}

	ctx := context.Background()
	start := time.Now()
	results, err := Run(ctx, cfg, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 10 {
		t.Errorf("expected 10 results, got %d", len(results))
	}

	if elapsed < 1*time.Second || elapsed > 2*time.Second {
		t.Errorf("expected duration around 1 second, got %v", elapsed)
	}
}

func TestProgressValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.LoadTestConfig{
		TargetURL: server.URL,
		Method:    "GET",
		RPS:       5,
		Duration:  2,
		Timeout:   10,
	}

	ctx := context.Background()
	progressCh := make(chan Progress, 10)

	go func() {
		Run(ctx, cfg, progressCh)
	}()

	var progressValues []Progress
	for progress := range progressCh {
		progressValues = append(progressValues, progress)
	}

	if len(progressValues) == 0 {
		t.Fatal("no progress updates received")
	}

	lastProgress := progressValues[len(progressValues)-1]
	if lastProgress.Progress != 1.0 {
		t.Errorf("expected final progress to be 1.0, got %f", lastProgress.Progress)
	}

	expectedTotal := cfg.RPS * cfg.Duration
	if lastProgress.Total != expectedTotal {
		t.Errorf("expected total to be %d, got %d", expectedTotal, lastProgress.Total)
	}

	for i := 1; i < len(progressValues); i++ {
		if progressValues[i].Completed < progressValues[i-1].Completed {
			t.Errorf("progress should be monotonically increasing, but went from %d to %d",
				progressValues[i-1].Completed, progressValues[i].Completed)
		}
	}
}
