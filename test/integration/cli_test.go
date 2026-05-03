package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	binaryName    = "zerohand"
	testServerURL = "https://httpbin.org"
)

var binaryPath string

// TestMain builds the binary before running tests and cleans up after
func TestMain(m *testing.M) {
	// Build the binary
	fmt.Println("Building CLI binary...")

	// Get the project root directory
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	// Navigate to project root (from test/integration to .)
	projectRoot := filepath.Join(wd, "..", "..")
	buildPath := filepath.Join(projectRoot, "cmd", "zerohand", "main.go")
	binaryPath = filepath.Join(os.TempDir(), binaryName)

	buildCmd := exec.Command("go", "build", "-o", binaryPath, buildPath)
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to build binary: %v\nOutput: %s\n", err, output)
		os.Exit(1)
	}
	fmt.Printf("Binary built successfully at: %s\n", binaryPath)

	// Run tests
	code := m.Run()

	// Cleanup
	os.Remove(binaryPath)

	os.Exit(code)
}

// runCLI executes the CLI with given arguments and returns stdout, stderr, and error
func runCLI(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = filepath.Join(".", "..", "..")

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// TestCLI_Version tests the version/help output
func TestCLI_Version(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"help flag", []string{"--help"}},
		{"help command", []string{"help"}},
		{"run help", []string{"run", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _, err := runCLI(t, tt.args...)

			// Help output returns exit code 0
			if err != nil {
				t.Logf("Command output: %s", stdout)
			}

			if !strings.Contains(stdout, "zerohand") {
				t.Errorf("Expected output to contain 'zerohand', got: %s", stdout)
			}
		})
	}
}

// TestCLI_BasicGETRequest tests a simple GET request
func TestCLI_BasicGETRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	args := []string{
		"run",
		"--url", testServerURL + "/get",
		"--rps", "5",
		"--duration", "3",
	}

	stdout, stderr, err := runCLI(t, args...)

	if err != nil {
		t.Fatalf("Command failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Verify output contains expected sections
	expectedSections := []string{
		"Load Test Configuration",
		"Target:",
		"Method:",
		"GET",
		"RPS:",
		"Duration:",
		"Load test completed",
		"Total Requests:",
		"Success:",
		"Response Time:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(stdout, section) {
			t.Errorf("Expected output to contain '%s', got:\n%s", section, stdout)
		}
	}

	// Verify approximately correct number of requests (5 RPS * 3 seconds = ~15 requests)
	// Allow some margin for timing variations
	totalRequestsRegex := regexp.MustCompile(`Total Requests:\s+(\d+)`)
	matches := totalRequestsRegex.FindStringSubmatch(stdout)
	if len(matches) < 2 {
		t.Fatalf("Could not find total requests in output:\n%s", stdout)
	}

	var totalRequests int
	fmt.Sscanf(matches[1], "%d", &totalRequests)
	if totalRequests < 12 || totalRequests > 18 {
		t.Errorf("Expected approximately 15 requests (5 RPS * 3s), got %d", totalRequests)
	}
}

// TestCLI_POSTRequest tests POST request with body and headers
func TestCLI_POSTRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	args := []string{
		"run",
		"--url", testServerURL + "/post",
		"--method", "POST",
		"--rps", "5",
		"--duration", "2",
		"--body", `{"test":"data","number":123}`,
		"--headers", "Content-Type:application/json,X-Custom-Header:test-value",
	}

	stdout, stderr, err := runCLI(t, args...)

	if err != nil {
		t.Fatalf("Command failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Verify POST method is shown
	if !strings.Contains(stdout, "POST") {
		t.Errorf("Expected output to contain 'POST', got:\n%s", stdout)
	}

	// Verify success
	if !strings.Contains(stdout, "Load test completed") {
		t.Errorf("Expected output to contain 'Load test completed', got:\n%s", stdout)
	}
}

// TestCLI_JSONOutput tests JSON output file generation
func TestCLI_JSONOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary output file
	outputFile := filepath.Join(os.TempDir(), "zerohand-test-output.json")
	defer os.Remove(outputFile)

	args := []string{
		"run",
		"--url", testServerURL + "/get",
		"--rps", "5",
		"--duration", "2",
		"--output", outputFile,
	}

	stdout, stderr, err := runCLI(t, args...)

	if err != nil {
		t.Fatalf("Command failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Verify file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Output file was not created: %s", outputFile)
	}

	// Verify file contains valid JSON
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// The output is currently an array of results
	// Try to parse as array first
	var results []any
	if err := json.Unmarshal(data, &results); err != nil {
		t.Fatalf("Output file contains invalid JSON: %v\nContent: %s", err, string(data))
	}

	// Verify we have results
	if len(results) == 0 {
		t.Error("Expected non-empty results array")
	}

	// Verify each result has expected structure
	if len(results) > 0 {
		firstResult, ok := results[0].(map[string]any)
		if !ok {
			t.Fatalf("Expected result to be an object, got: %T", results[0])
		}

		requiredFields := []string{"timestamp", "status_code", "duration_us"}
		for _, field := range requiredFields {
			if _, ok := firstResult[field]; !ok {
				t.Errorf("Expected result to contain field '%s', got: %v", field, firstResult)
			}
		}
	}
}

// TestCLI_InvalidURL tests error handling for invalid URL
func TestCLI_InvalidURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{"missing scheme", "httpbin.org/get", true},
		{"invalid scheme", "ftp://httpbin.org/get", true},
		{"empty url", "", true},
		{"valid url", "https://httpbin.org/get", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{
				"run",
				"--url", tt.url,
				"--rps", "1",
				"--duration", "1",
			}

			_, stderr, err := runCLI(t, args...)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URL '%s', but command succeeded", tt.url)
				}
				// Verify error message contains useful information
				combinedOutput := stderr
				if !strings.Contains(combinedOutput, "invalid") && !strings.Contains(combinedOutput, "error") && !strings.Contains(combinedOutput, "Error") {
					t.Logf("Warning: Error message may not be descriptive enough: %s", combinedOutput)
				}
			} else if err != nil && testing.Short() == false {
				t.Errorf("Expected success for URL '%s', but got error: %v\nStderr: %s", tt.url, err, stderr)
			}
		})
	}
}

// TestCLI_InvalidMethod tests error handling for invalid HTTP method
func TestCLI_InvalidMethod(t *testing.T) {
	args := []string{
		"run",
		"--url", testServerURL + "/get",
		"--method", "INVALID",
		"--rps", "1",
		"--duration", "1",
	}

	_, stderr, err := runCLI(t, args...)

	if err == nil {
		t.Error("Expected error for invalid method, but command succeeded")
	}

	if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "method") {
		t.Logf("Warning: Error message may not clearly indicate method error: %s", stderr)
	}
}

// TestCLI_InvalidParameters tests error handling for invalid parameters
func TestCLI_InvalidParameters(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			"zero RPS",
			[]string{"run", "--url", testServerURL + "/get", "--rps", "0", "--duration", "1"},
		},
		{
			"negative RPS",
			[]string{"run", "--url", testServerURL + "/get", "--rps", "-1", "--duration", "1"},
		},
		{
			"zero duration",
			[]string{"run", "--url", testServerURL + "/get", "--rps", "1", "--duration", "0"},
		},
		{
			"negative duration",
			[]string{"run", "--url", testServerURL + "/get", "--rps", "1", "--duration", "-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr, err := runCLI(t, tt.args...)

			if err == nil {
				t.Errorf("Expected error for %s, but command succeeded", tt.name)
			}

			if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "must be") && !strings.Contains(stderr, "error") {
				t.Logf("Warning: Error message may not be descriptive: %s", stderr)
			}
		})
	}
}

// TestCLI_TimeoutHandling tests timeout handling
func TestCLI_TimeoutHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	args := []string{
		"run",
		"--url", testServerURL + "/delay/10", // Server delays 10 seconds
		"--rps", "2",
		"--duration", "3",
		"--timeout", "1", // Client timeout 1 second
	}

	stdout, stderr, err := runCLI(t, args...)

	if err != nil {
		t.Fatalf("Command failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Verify that timeouts are reported
	if !strings.Contains(stdout, "timeout") && !strings.Contains(stdout, "Timeout") {
		t.Logf("Warning: Expected output to mention timeouts:\n%s", stdout)
	}

	// Verify that not all requests succeeded (some should timeout)
	successRegex := regexp.MustCompile(`Success:\s+\d+\s+\((\d+\.\d+)%\)`)
	matches := successRegex.FindStringSubmatch(stdout)
	if len(matches) >= 2 {
		var successRate float64
		fmt.Sscanf(matches[1], "%f", &successRate)
		if successRate > 50.0 {
			t.Logf("Warning: Expected low success rate due to timeouts, got %.2f%%", successRate)
		}
	}
}

// TestCLI_ErrorStatusCode tests handling of error status codes
func TestCLI_ErrorStatusCode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	args := []string{
		"run",
		"--url", testServerURL + "/status/500",
		"--rps", "5",
		"--duration", "2",
	}

	stdout, stderr, err := runCLI(t, args...)

	if err != nil {
		t.Fatalf("Command failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Verify that 500 status codes are shown in distribution
	if !strings.Contains(stdout, "500") {
		t.Errorf("Expected output to show status code 500:\n%s", stdout)
	}

	// Command should still complete successfully even with 500 responses
	if !strings.Contains(stdout, "Load test completed") {
		t.Errorf("Expected output to contain 'Load test completed':\n%s", stdout)
	}
}

// TestCLI_MissingRequiredFlag tests that missing required --url flag is caught
func TestCLI_MissingRequiredFlag(t *testing.T) {
	args := []string{
		"run",
		"--rps", "10",
		"--duration", "5",
		// --url is missing
	}

	_, stderr, err := runCLI(t, args...)

	if err == nil {
		t.Error("Expected error when --url flag is missing, but command succeeded")
	}

	if !strings.Contains(stderr, "required") && !strings.Contains(stderr, "url") {
		t.Logf("Warning: Error message should indicate missing URL: %s", stderr)
	}
}

// TestCLI_HeadersParsing tests custom headers parsing
func TestCLI_HeadersParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name    string
		headers string
	}{
		{"single header", "X-Custom:value"},
		{"multiple headers", "Content-Type:application/json,Authorization:Bearer token123"},
		{"headers with spaces", "X-Custom:value with spaces,Another:test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{
				"run",
				"--url", testServerURL + "/get",
				"--rps", "2",
				"--duration", "1",
				"--headers", tt.headers,
			}

			stdout, stderr, err := runCLI(t, args...)

			if err != nil {
				t.Fatalf("Command failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
			}

			if !strings.Contains(stdout, "Load test completed") {
				t.Errorf("Expected successful completion with headers '%s':\n%s", tt.headers, stdout)
			}
		})
	}
}
