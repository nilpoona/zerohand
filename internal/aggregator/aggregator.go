package aggregator

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/nilpoona/zerohand/internal/executor"
)

// Stats holds aggregated statistical information
type Stats struct {
	TotalRequests  int            `json:"total_requests"`
	Request        RequestStats   `json:"request"`
	Response       ResponseStats  `json:"response"`
	StatusCodeDist map[int]int    `json:"status_code_dist"`
	ErrorDist      map[string]int `json:"error_dist"`
}

// RequestStats holds request-related statistics
type RequestStats struct {
	SuccessCount int     `json:"success_count"`
	FailureCount int     `json:"failure_count"`
	SuccessRate  float64 `json:"success_rate"` // Percentage
	FailureRate  float64 `json:"failure_rate"` // Percentage
	TimeoutCount int     `json:"timeout_count"`
}

// ResponseStats holds response time statistics (all in milliseconds)
type ResponseStats struct {
	AvgDuration    float64 `json:"avg_duration"`
	MedianDuration float64 `json:"median_duration"`
	P95Duration    float64 `json:"p95_duration"`
	P99Duration    float64 `json:"p99_duration"`
	MinDuration    float64 `json:"min_duration"`
	MaxDuration    float64 `json:"max_duration"`
}

// CalculateStats calculates statistics from the results
func CalculateStats(results []*executor.Result) *Stats {
	stats := &Stats{
		TotalRequests:  len(results),
		StatusCodeDist: make(map[int]int),
		ErrorDist:      make(map[string]int),
	}

	if len(results) == 0 {
		return stats
	}

	// Collect durations (in microseconds)
	durations := make([]int64, 0, len(results))
	var totalDuration int64

	for _, result := range results {
		// If there's an error
		if result.Error != "" {
			stats.Request.FailureCount++
			stats.ErrorDist[result.Error]++

			if result.Error == "timeout" {
				stats.Request.TimeoutCount++
			}
		} else {
			// Success (2xx status codes)
			if result.StatusCode >= 200 && result.StatusCode < 300 {
				stats.Request.SuccessCount++
			} else {
				stats.Request.FailureCount++
			}

			stats.StatusCodeDist[result.StatusCode]++
		}

		// Collect durations
		durations = append(durations, result.Duration)
		totalDuration += result.Duration
	}

	// Calculate request statistics
	if stats.TotalRequests > 0 {
		stats.Request.SuccessRate = float64(stats.Request.SuccessCount) / float64(stats.TotalRequests) * 100
		stats.Request.FailureRate = float64(stats.Request.FailureCount) / float64(stats.TotalRequests) * 100
	}

	// Calculate response time statistics
	// Average duration (in milliseconds)
	stats.Response.AvgDuration = float64(totalDuration) / float64(len(results)) / 1000.0

	// Sort durations
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	// Minimum and maximum values
	stats.Response.MinDuration = float64(durations[0]) / 1000.0
	stats.Response.MaxDuration = float64(durations[len(durations)-1]) / 1000.0

	// Calculate percentiles
	stats.Response.MedianDuration = calculatePercentile(durations, 0.50)
	stats.Response.P95Duration = calculatePercentile(durations, 0.95)
	stats.Response.P99Duration = calculatePercentile(durations, 0.99)

	return stats
}

// calculatePercentile calculates the percentile from sorted durations
// Returns the result in milliseconds
func calculatePercentile(sortedDurations []int64, percentile float64) float64 {
	if len(sortedDurations) == 0 {
		return 0
	}

	index := int(float64(len(sortedDurations)-1) * percentile)
	return float64(sortedDurations[index]) / 1000.0 // Convert from microseconds to milliseconds
}

// FormatStats formats statistics in a human-readable format
func FormatStats(stats *Stats) string {
	var sb strings.Builder

	sb.WriteString("\nResults:\n")
	sb.WriteString(fmt.Sprintf("Total Requests:  %d\n", stats.TotalRequests))

	if stats.TotalRequests > 0 {
		sb.WriteString(fmt.Sprintf("Success:         %d (%.2f%%)\n", stats.Request.SuccessCount, stats.Request.SuccessRate))
		sb.WriteString(fmt.Sprintf("Failure:         %d (%.2f%%)\n", stats.Request.FailureCount, stats.Request.FailureRate))
	}

	sb.WriteString("\nResponse Time:\n")
	sb.WriteString(fmt.Sprintf("  Average:  %.2fms\n", stats.Response.AvgDuration))
	sb.WriteString(fmt.Sprintf("  Min:      %.2fms\n", stats.Response.MinDuration))
	sb.WriteString(fmt.Sprintf("  Median:   %.2fms\n", stats.Response.MedianDuration))
	sb.WriteString(fmt.Sprintf("  P95:      %.2fms\n", stats.Response.P95Duration))
	sb.WriteString(fmt.Sprintf("  P99:      %.2fms\n", stats.Response.P99Duration))
	sb.WriteString(fmt.Sprintf("  Max:      %.2fms\n", stats.Response.MaxDuration))

	// Status code distribution
	if len(stats.StatusCodeDist) > 0 {
		sb.WriteString("\nStatus Codes:\n")

		// Sort status codes
		statusCodes := make([]int, 0, len(stats.StatusCodeDist))
		for code := range stats.StatusCodeDist {
			statusCodes = append(statusCodes, code)
		}
		sort.Ints(statusCodes)

		for _, code := range statusCodes {
			count := stats.StatusCodeDist[code]
			sb.WriteString(fmt.Sprintf("  %d: %d\n", code, count))
		}
	}

	// Error distribution
	if len(stats.ErrorDist) > 0 {
		sb.WriteString("\nErrors:\n")

		// Sort errors
		errors := make([]string, 0, len(stats.ErrorDist))
		for err := range stats.ErrorDist {
			errors = append(errors, err)
		}
		sort.Strings(errors)

		for _, err := range errors {
			count := stats.ErrorDist[err]
			sb.WriteString(fmt.Sprintf("  %s: %d\n", err, count))
		}
	}

	return sb.String()
}

// SaveResultsJSON saves results to a JSON file
func SaveResultsJSON(results []*executor.Result, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}
