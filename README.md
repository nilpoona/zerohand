# Zerohand

A distributed load testing tool for Web APIs built with Go.

## Overview

Zerohand is a distributed load testing tool designed to help developers and DevOps engineers quickly assess API performance and capacity. With a focus on spike tests, load tests, and stress tests, Zerohand provides:

- Simple CLI interface for quick load testing
- RPS (requests per second) control
- Detailed response time metrics and statistics
- Support for various HTTP methods and custom headers
- Real-time progress tracking and graceful shutdown

Whether you're validating capacity before a product launch, testing autoscaling behavior, or finding performance bottlenecks, Zerohand makes it easy to generate controlled load and analyze results.

## Features

- **Configurable RPS**: Control requests per second to generate desired load
- **Multiple HTTP Methods**: Support for GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
- **Custom Headers & Body**: Add custom headers and request body
- **Response Time Metrics**: Measure average, median, P95, P99, min, and max response times
- **Success/Failure Tracking**: Track request outcomes and error types
- **Progress Display**: Real-time progress bar during test execution
- **JSON Export**: Save detailed results to JSON file
- **Graceful Shutdown**: Ctrl+C support with partial results display

## Installation

### Download Pre-built Binaries (Recommended)

Download the latest release for your platform from the [GitHub Releases](https://github.com/nilpoona/zerohand/releases) page.

#### Linux / macOS
```bash
# Download the binary (replace VERSION, OS, and ARCH with your values)
# Example for Linux amd64:
curl -LO https://github.com/nilpoona/zerohand/releases/download/v0.1.0/zerohand_0.1.0_linux_amd64.tar.gz

# Extract the archive
tar -xzf zerohand_0.1.0_linux_amd64.tar.gz

# Move to a directory in your PATH
sudo mv zerohand /usr/local/bin/

# Verify installation
zerohand --help
```

#### Windows
1. Download the `.zip` file for Windows from the [releases page](https://github.com/nilpoona/zerohand/releases)
2. Extract the archive
3. Add the `zerohand.exe` to your PATH or run it directly

### Build from Source

#### Prerequisites

- Go 1.25 or later

#### Steps
```bash
# Clone the repository
git clone https://github.com/nilpoona/zerohand.git
cd zerohand

# Install dependencies
go mod tidy

# Build the CLI binary
go build -o zerohand cmd/zerohand/main.go
```

## Usage

### Basic GET Request
```bash
./zerohand run --url https://httpbin.org/get --rps 10 --duration 5
```

### POST Request with Body and Headers
```bash
./zerohand run \
  --url https://httpbin.org/post \
  --method POST \
  --rps 20 \
  --duration 10 \
  --body '{"name":"test","value":123}' \
  --headers "Content-Type:application/json,Authorization:Bearer token"
```

### Save Results to JSON
```bash
./zerohand run \
  --url https://api.example.com/endpoint \
  --rps 50 \
  --duration 30 \
  --output results.json
```

### Test with Custom Timeout
```bash
./zerohand run \
  --url https://httpbin.org/delay/15 \
  --rps 5 \
  --duration 10 \
  --timeout 2
```

## Command-Line Options

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `--url` | Target URL to test | - | Yes |
| `--method` | HTTP method (GET, POST, PUT, etc.) | GET | No |
| `--rps` | Requests per second | 10 | No |
| `--duration` | Test duration in seconds | 10 | No |
| `--body` | Request body (for POST, PUT, etc.) | "" | No |
| `--headers` | Custom headers (format: key:value,key:value) | "" | No |
| `--timeout` | Request timeout in seconds | 10 | No |
| `--output` | Save results to JSON file | "" | No |

## Output Format

### Terminal Output Example
```
=== Load Test Configuration ===
Test ID:     a1b2c3d4-e5f6-7890-abcd-ef1234567890
Target:      https://httpbin.org/get
Method:      GET
RPS:         10
Duration:    5 seconds
Timeout:     10 seconds
===============================

Starting load test...
Running... [========================================] 100.0% (50/50)

Load test completed

Results:
Total Requests:  50
Success:         50 (100.00%)
Failure:         0 (0.00%)

Response Time:
  Average:   123.45ms
  Min:       89.12ms
  Median:    120.34ms
  P95:       156.78ms
  P99:       178.90ms
  Max:       189.01ms

Status Codes:
  200: 50
```

### JSON Output Structure

When using `--output` flag, results are saved in the following format:
```json
[
  {
    "timestamp": "2025-01-15T10:30:00.123456Z",
    "status_code": 200,
    "duration_us": 123456,
    "error": ""
  },
  {
    "timestamp": "2025-01-15T10:30:00.234567Z",
    "status_code": 500,
    "duration_us": 234567,
    "error": ""
  },
  {
    "timestamp": "2025-01-15T10:30:00.345678Z",
    "status_code": 0,
    "duration_us": 0,
    "error": "timeout"
  }
]
```

## Use Cases

### Spike Testing
Test how your API handles sudden traffic spikes (5-10 minutes):
```bash
./zerohand run --url https://api.example.com/endpoint --rps 1000 --duration 600
```

### Load Testing
Verify stable operation under expected traffic (20-30 minutes):
```bash
./zerohand run --url https://api.example.com/endpoint --rps 200 --duration 1800
```

### Stress Testing
Find system limits by gradually increasing load:
```bash
# Phase 1: Baseline
./zerohand run --url https://api.example.com/endpoint --rps 100 --duration 300

# Phase 2: Increased load
./zerohand run --url https://api.example.com/endpoint --rps 500 --duration 300

# Phase 3: High load
./zerohand run --url https://api.example.com/endpoint --rps 1000 --duration 300
```

## Error Classification

Errors are automatically classified into the following categories:

- `timeout`: Request or connection timeout
- `dns_error`: DNS resolution failure
- `connection_error`: Connection refused or failed
- `tls_error`: TLS/SSL certificate error
- `network_error`: Other network-related errors
- `canceled`: Request canceled (e.g., Ctrl+C)
- `unknown_error`: Unclassified errors

## Performance Notes

**Performance Goals:**
- RPS Accuracy: ±5% of requested RPS
- Efficient memory usage for up to 1000 concurrent requests
- Low CPU overhead at moderate load levels

## Testing

### Running Tests

```bash
# Run all unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests (E2E tests with actual binary)
go test -v ./test/integration/... -timeout 5m

# Run integration tests in short mode (skips network tests)
go test -v ./test/integration/... -short
```

For more details on integration tests, see [test/integration/README.md](test/integration/README.md).

### Testing Recommendations

#### Test Against Public APIs

- [httpbin.org](https://httpbin.org): Great for testing various HTTP methods and responses
- [jsonplaceholder.typicode.com](https://jsonplaceholder.typicode.com): REST API for testing

#### Start with Low RPS

Begin with low RPS (10-50) to avoid overwhelming the target server.

Example test sequence:
```bash
# Test 1: Basic connectivity
./zerohand run --url https://httpbin.org/get --rps 5 --duration 3

# Test 2: Increase load
./zerohand run --url https://httpbin.org/get --rps 20 --duration 10

# Test 3: Test error handling
./zerohand run --url https://httpbin.org/status/500 --rps 10 --duration 5
```

## Future Enhancements

- Distributed load generation across multiple machines
- Real-time metrics visualization
- WebSocket and Server-Sent Events support
- Multi-step scenario testing
- Advanced authentication support
- Result history and comparison tools

## Troubleshooting

### Connection Refused Errors

Check if the target URL is correct and the server is running.

### DNS Resolution Errors

Verify the domain name and DNS settings.

### High Timeout Count

Consider increasing the `--timeout` value or checking network connectivity.

### Progress Not Updating

Progress updates every second. For very short tests (<3 seconds), you might not see many updates.

