package runner

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nilpoona/zerohand/internal/config"
	"github.com/nilpoona/zerohand/internal/executor"
)

// Progress represents the progress status
type Progress struct {
	Progress  float64 // 0.0 ~ 1.0
	Completed int     // Number of completed requests
	Total     int     // Total number of requests
}

// Run executes a load test and returns all results
// If progressCh is not nil, periodically sends progress updates (closed when completed)
func Run(ctx context.Context, cfg *config.LoadTestConfig, progressCh chan<- Progress) ([]*executor.Result, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	totalRequests := cfg.RPS * cfg.Duration

	results := make([]*executor.Result, 0, totalRequests)
	var resultsMu sync.Mutex

	var completed atomic.Int32

	// Wait for goroutine completion using WaitGroup
	var wg sync.WaitGroup

	// Ticker for RPS control
	interval := time.Second / time.Duration(cfg.RPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Done signal for progress updates
	done := make(chan struct{})
	defer close(done)

	// Goroutine for progress updates
	if progressCh != nil {
		defer close(progressCh)

		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-done:
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
					current := int(completed.Load())

					progress := float64(current) / float64(totalRequests)

					select {
					case progressCh <- Progress{
						Progress:  progress,
						Completed: current,
						Total:     totalRequests,
					}:
					case <-done:
						return
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// Request send counter
	requestsSent := 0

	// Main loop: Send requests according to RPS
	for {
		select {
		case <-ctx.Done():
			// If context is cancelled, wait for all goroutines to complete
			wg.Wait()

			resultsMu.Lock()
			defer resultsMu.Unlock()
			return results, ctx.Err()

		case <-ticker.C:
			// Send request
			requestsSent++

			wg.Go(func() {
				// Execute request
				result := executor.ExecuteRequest(ctx, cfg)

				// Collect results
				resultsMu.Lock()
				results = append(results, result)
				resultsMu.Unlock()

				// Increment completion count
				completed.Add(1)
			})

			// Exit if total request count is reached
			if requestsSent >= totalRequests {
				// All requests have been sent, stop the Ticker
				ticker.Stop()

				// Wait for all goroutines to complete
				wg.Wait()

				// Report final progress
				if progressCh != nil {
					current := int(completed.Load())

					select {
					case progressCh <- Progress{
						Progress:  1.0,
						Completed: current,
						Total:     totalRequests,
					}:
					default:
						// Skip if channel is full
					}
				}

				resultsMu.Lock()
				defer resultsMu.Unlock()
				return results, nil
			}
		}
	}
}
