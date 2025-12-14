package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/nilpoona/zerohand/internal/aggregator"
	"github.com/nilpoona/zerohand/internal/config"
	"github.com/nilpoona/zerohand/internal/runner"
)

func main() {
	// Create hardcoded configuration for local testing
	cfg := &config.LoadTestConfig{
		TestID:    "test-123",
		LambdaID:  1,
		TargetURL: "https://httpbin.org/get",
		Method:    "GET",
		Headers:   map[string]string{},
		Body:      "",
		RPS:       10,
		Duration:  5,
		Timeout:   10,
	}

	fmt.Println("=== Lambda Local Test ===")
	fmt.Printf("Test ID:  %s\n", cfg.TestID)
	fmt.Printf("Lambda ID: %d\n", cfg.LambdaID)
	fmt.Printf("Target:   %s\n", cfg.TargetURL)
	fmt.Printf("RPS:      %d\n", cfg.RPS)
	fmt.Printf("Duration: %d seconds\n", cfg.Duration)
	fmt.Println("========================")

	// Run load test
	ctx := context.Background()
	fmt.Println("\nRunning load test...")

	results, err := runner.Run(ctx, cfg, nil)
	if err != nil {
		log.Fatalf("Load test failed: %v", err)
	}

	fmt.Printf("Completed %d requests\n", len(results))

	// Calculate statistics
	stats := aggregator.CalculateStats(results)

	// Print stats
	fmt.Println(aggregator.FormatStats(stats))

	// Ensure testdata/results directory exists
	resultsDir := "testdata/results"
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		log.Fatalf("Failed to create results directory: %v", err)
	}

	// Save results to JSON file
	outputPath := filepath.Join(resultsDir, fmt.Sprintf("lambda-%d-%s.json", cfg.LambdaID, cfg.TestID))
	if err := aggregator.SaveResultsJSON(results, outputPath); err != nil {
		log.Printf("Failed to save results: %v", err)
	} else {
		fmt.Printf("\nResults saved to: %s\n", outputPath)
	}
}

// Future: Add AWS Lambda handler
// func HandleRequest(ctx context.Context, event config.LoadTestConfig) (Response, error) {
//     results, err := runner.Run(ctx, &event, nil)
//     if err != nil {
//         return Response{}, err
//     }
//
//     stats := aggregator.CalculateStats(results)
//     return Response{
//         Stats:   stats,
//         Results: results,
//     }, nil
// }
