# Zerohand

A distributed load testing tool for Web APIs built with Go.

## Overview

Zerohand is a serverless distributed load testing tool that leverages AWS Lambda to generate load. This MVP version runs locally without AWS deployment, providing a simple CLI interface to conduct load tests against HTTP endpoints.

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

### Prerequisites

- Go 1.25 or later

### Build from Source

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

### Test with Timeout

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

## Error Classification

Errors are automatically classified into the following categories:

- `timeout`: Request or connection timeout
- `dns_error`: DNS resolution failure
- `connection_error`: Connection refused or failed
- `tls_error`: TLS/SSL certificate error
- `network_error`: Other network-related errors
- `canceled`: Request canceled (e.g., Ctrl+C)
- `unknown_error`: Unclassified errors

## Project Structure

```
zerohand/
├── cmd/
│   ├── zerohand/              # CLI application
│   │   └── main.go
│   └── lambda/                # Lambda function (local testing)
│       └── main.go
├── internal/                  # Private application code
│   ├── config/               # Configuration structures
│   │   └── config.go
│   ├── executor/             # HTTP request execution logic
│   │   └── executor.go
│   ├── runner/               # RPS control and orchestration
│   │   └── runner.go
│   └── aggregator/           # Result aggregation and statistics
│       └── aggregator.go
├── testdata/
│   └── results/              # JSON result files
├── go.mod
├── go.sum
├── CLAUDE.md
└── README.md
```

## Local Lambda Testing

For testing the Lambda function locally:

```bash
go run cmd/lambda/main.go
```

This will execute a hardcoded test configuration and save results to `testdata/results/`.

## Performance Notes

**MVP Performance Goals:**
- RPS Accuracy: ±5% of requested RPS
- Memory Usage: Reasonable for up to 1000 concurrent requests
- CPU Usage: Should not saturate CPU at low RPS (<100)

**Current Limitations:**
- Spawns one goroutine per request (simple but not highly scalable)
- All results stored in memory before aggregation
- Progress updates every second

For production use with high RPS (>1000), consider implementing worker pools and result streaming.

## Testing Recommendations

### Test Against Public APIs

- httpbin.org: Great for testing various HTTP methods and responses
- jsonplaceholder.typicode.com: REST API for testing

### Start with Low RPS

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

- AWS Lambda deployment support
- Multiple Lambda orchestration for distributed load
- DynamoDB for result storage
- Real-time progress updates via WebSocket
- Web UI (React + TypeScript)
- Authentication (Cognito or similar)
- Per-user rate limiting

## Development Guidelines

- Write clean, idiomatic Go code
- Use `context.Context` for cancellation
- Handle errors explicitly
- Add comments for complex logic
- Keep functions small and focused
- Use meaningful variable names

## Troubleshooting

### Connection Refused Errors

Check if the target URL is correct and the server is running.

### DNS Resolution Errors

Verify the domain name and DNS settings.

### High Timeout Count

Consider increasing the `--timeout` value or checking network connectivity.

### Progress Not Updating

Progress updates every second. For very short tests (<3 seconds), you might not see many updates.

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]

## Contact

[Add contact information here]
