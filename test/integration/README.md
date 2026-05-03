# CLI Integration Tests

This directory contains E2E (end-to-end) integration tests for the Zerohand CLI.

## Overview

Integration tests verify the CLI behavior by executing the actual compiled binary.

### Test Coverage

1. **Basic Functionality Tests**
   - `TestCLI_Version`: Help and version display
   - `TestCLI_BasicGETRequest`: Basic GET request execution
   - `TestCLI_POSTRequest`: POST request with body and headers
   - `TestCLI_JSONOutput`: JSON file output functionality

2. **Error Handling Tests**
   - `TestCLI_InvalidURL`: Invalid URL validation
   - `TestCLI_InvalidMethod`: Invalid HTTP method validation
   - `TestCLI_InvalidParameters`: Invalid parameter validation
   - `TestCLI_MissingRequiredFlag`: Required flag validation

3. **Real-World Scenario Tests**
   - `TestCLI_TimeoutHandling`: Timeout handling verification
   - `TestCLI_ErrorStatusCode`: Error status code handling
   - `TestCLI_HeadersParsing`: Custom header parsing

## Running Tests

### Normal Execution

```bash
# Run all integration tests
go test -v ./test/integration/...

# Run with timeout (recommended)
go test -v ./test/integration/... -timeout 5m
```

### Short Mode (Skip Network Tests)

```bash
# Short mode (runs only tests that don't use external APIs)
go test -v ./test/integration/... -short
```

### Run Specific Tests

```bash
# Run a specific test function
go test -v ./test/integration/... -run TestCLI_BasicGETRequest

# Run multiple tests using pattern matching
go test -v ./test/integration/... -run "TestCLI_Invalid.*"
```

## How It Works

### Automatic Build with TestMain

The `TestMain` function automatically builds the CLI binary before running tests:

```go
func TestMain(m *testing.M) {
    // 1. Build CLI binary from project root
    // 2. Place binary in temporary directory
    // 3. Run all tests
    // 4. Clean up the built binary
}
```

### runCLI Helper Function

Each test uses the `runCLI` helper function to execute the binary:

```go
stdout, stderr, err := runCLI(t, "run", "--url", "https://example.com")
```

## Adding New Tests

To add a new test case:

```go
func TestCLI_YourNewTest(t *testing.T) {
    // Skip in short mode if using external APIs
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    args := []string{
        "run",
        "--url", "https://httpbin.org/get",
        "--rps", "5",
        "--duration", "2",
    }

    stdout, stderr, err := runCLI(t, args...)

    if err != nil {
        t.Fatalf("Command failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
    }

    // Assertions
    if !strings.Contains(stdout, "Load test completed") {
        t.Errorf("Expected success message")
    }
}
```

## Important Notes

### External Dependencies

These tests use `https://httpbin.org`:

- Internet connection is required to run tests
- Tests will fail if httpbin.org is unavailable
- Consider using `-short` flag in CI/CD environments

### Timing Variations

- RPS (Requests Per Second) precision tests allow approximately ±20% margin
- Execution time may vary due to network latency

### Cleanup

- The binary built at `/tmp/zerohand` is automatically removed after test execution
- Temporary files generated during tests are cleaned up using `defer` in each test

## Troubleshooting

### Build Errors

```
Failed to build binary: exit status 1
```

- Verify that `cmd/zerohand/main.go` exists
- Ensure `go.mod` is properly configured
- Update dependencies: `go mod tidy`

### Timeout Errors

```
panic: test timed out after 2m0s
```

- Increase timeout: `go test -timeout 10m`
- Check network connection

### httpbin.org Connection Errors

```
connection refused
```

- Check internet connection
- Verify proxy settings
- Use `-short` flag to skip external API tests

## CI/CD Execution

Recommended configuration for GitHub Actions or other CI/CD environments:

```yaml
- name: Run integration tests
  run: |
    go test -v ./test/integration/... -timeout 10m
  env:
    # Set environment variables as needed
    GO111MODULE: on
```

Short mode execution (faster, no external dependencies):

```yaml
- name: Run integration tests (short)
  run: |
    go test -v ./test/integration/... -short
```

## Test Coverage

Features covered by integration tests:

- ✅ CLI flag parsing
- ✅ Configuration validation
- ✅ GET/POST request execution
- ✅ Custom headers and body transmission
- ✅ JSON output functionality
- ✅ Error handling
- ✅ Timeout handling
- ✅ Progress display
- ✅ Statistics calculation and display

## Related Documentation

- [Project README](../../README.md)
- [Developer Documentation](../../CLAUDE.md)
- [Unit Tests](../../internal/)
