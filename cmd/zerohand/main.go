package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/nilpoona/zerohand/internal/aggregator"
	"github.com/nilpoona/zerohand/internal/config"
	"github.com/nilpoona/zerohand/internal/runner"
	"github.com/spf13/cobra"
)

var (
	url      string
	method   string
	rps      int
	duration int
	body     string
	headers  string
	timeout  int
	output   string
)

var rootCmd = &cobra.Command{
	Use:   "zerohand",
	Short: "Distributed load testing tool",
	Long:  "A distributed load testing tool for Web APIs",
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run load test",
	RunE:  runLoadTest,
}

func init() {
	runCmd.Flags().StringVar(&url, "url", "", "Target URL (required)")
	runCmd.Flags().StringVar(&method, "method", "GET", "HTTP method")
	runCmd.Flags().IntVar(&rps, "rps", 10, "Requests per second")
	runCmd.Flags().IntVar(&duration, "duration", 10, "Test duration in seconds")
	runCmd.Flags().StringVar(&body, "body", "", "Request body")
	runCmd.Flags().StringVar(&headers, "headers", "", "Headers (key:value,key:value)")
	runCmd.Flags().IntVar(&timeout, "timeout", 10, "Timeout in seconds")
	runCmd.Flags().StringVar(&output, "output", "", "Save results to JSON file")

	runCmd.MarkFlagRequired("url")
	rootCmd.AddCommand(runCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runLoadTest(cmd *cobra.Command, args []string) error {
	// Parse headers
	headerMap, err := parseHeaders(headers)
	if err != nil {
		return fmt.Errorf("failed to parse headers: %w", err)
	}

	// Create configuration
	cfg := &config.LoadTestConfig{
		TestID:    uuid.New().String(),
		LambdaID:  0,
		TargetURL: url,
		Method:    method,
		Headers:   headerMap,
		Body:      body,
		RPS:       rps,
		Duration:  duration,
		Timeout:   timeout,
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Print test configuration
	printTestConfig(cfg)

	// Setup signal handling (Ctrl+C support)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create progress channel
	progressCh := make(chan runner.Progress, 10)

	// Start progress display goroutine
	go func() {
		for progress := range progressCh {
			printProgress(progress)
		}
	}()

	// Run load test
	fmt.Println("\nStarting load test...")
	results, err := runner.Run(ctx, cfg, progressCh)

	// Check for errors
	if err != nil && err != context.Canceled {
		return fmt.Errorf("load test failed: %w", err)
	}

	if err == context.Canceled {
		fmt.Println("\n\nLoad test interrupted")
	} else {
		fmt.Println("\n\nLoad test completed")
	}

	// Calculate statistics
	stats := aggregator.CalculateStats(results)

	// Display results
	fmt.Println(aggregator.FormatStats(stats))

	// Save to JSON file (optional)
	if output != "" {
		if err := aggregator.SaveResultsJSON(results, output); err != nil {
			return fmt.Errorf("failed to save results: %w", err)
		}
		fmt.Printf("\nResults saved to: %s\n", output)
	}

	return nil
}

// parseHeaders parses header string into a map
// Format: "key1:value1,key2:value2"
func parseHeaders(headerStr string) (map[string]string, error) {
	headers := make(map[string]string)

	if headerStr == "" {
		return headers, nil
	}

	pairs := strings.Split(headerStr, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header format: %s (expected key:value)", pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, fmt.Errorf("empty header key in: %s", pair)
		}

		headers[key] = value
	}

	return headers, nil
}

// printTestConfig displays the test configuration
func printTestConfig(cfg *config.LoadTestConfig) {
	fmt.Println("\n=== Load Test Configuration ===")
	fmt.Printf("Test ID:     %s\n", cfg.TestID)
	fmt.Printf("Target:      %s\n", cfg.TargetURL)
	fmt.Printf("Method:      %s\n", cfg.Method)
	fmt.Printf("RPS:         %d\n", cfg.RPS)
	fmt.Printf("Duration:    %d seconds\n", cfg.Duration)
	fmt.Printf("Timeout:     %d seconds\n", cfg.Timeout)

	if len(cfg.Headers) > 0 {
		fmt.Println("Headers:")
		for key, value := range cfg.Headers {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	if cfg.Body != "" {
		fmt.Printf("Body:        %s\n", cfg.Body)
	}

	fmt.Println("===============================")
}

// printProgress displays progress bar
func printProgress(p runner.Progress) {
	// Progress bar width
	barWidth := 40
	filledWidth := int(p.Progress * float64(barWidth))

	// Build progress bar
	bar := strings.Repeat("=", filledWidth)
	if filledWidth < barWidth {
		bar += ">"
		bar += strings.Repeat(" ", barWidth-filledWidth-1)
	}

	// Display progress (overwrite with \r)
	fmt.Printf("\rRunning... [%s] %.1f%% (%d/%d)", bar, p.Progress*100, p.Completed, p.Total)
}
