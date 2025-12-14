package executor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nilpoona/zerohand/internal/config"
)

// Result holds the result of an individual HTTP request
type Result struct {
	Timestamp  time.Time `json:"timestamp"`
	StatusCode int       `json:"status_code"`
	Duration   int64     `json:"duration_us"` // Stored in microseconds, converted to milliseconds for display
	Error      string    `json:"error,omitempty"`
}

// ExecuteRequest executes a single HTTP request and returns the result
func ExecuteRequest(ctx context.Context, cfg *config.LoadTestConfig) *Result {
	result := &Result{
		Timestamp: time.Now(),
	}

	// Create HTTP client
	client := createHTTPClient(time.Duration(cfg.Timeout) * time.Second)

	// Prepare request
	req, err := prepareRequest(cfg)
	if err != nil {
		result.Error = classifyError(err)
		return result
	}

	// Set context
	req = req.WithContext(ctx)

	// Execute request and measure time
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	result.Duration = duration.Microseconds()

	// Handle errors
	if err != nil {
		result.Error = classifyError(err)
		return result
	}
	defer resp.Body.Close()

	// Read and discard response body (to properly close the connection)
	io.Copy(io.Discard, resp.Body)

	// Set status code on success
	result.StatusCode = resp.StatusCode

	return result
}

// createHTTPClient creates an HTTP client with timeout settings
func createHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		},
	}
}

// prepareRequest creates an http.Request from the configuration
func prepareRequest(cfg *config.LoadTestConfig) (*http.Request, error) {
	var bodyReader io.Reader
	if cfg.Body != "" {
		bodyReader = bytes.NewBufferString(cfg.Body)
	}

	req, err := http.NewRequest(cfg.Method, cfg.TargetURL, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}

	return req, nil
}

// classifyError classifies errors and returns a user-friendly error message
func classifyError(err error) string {
	if err == nil {
		return ""
	}

	// Context cancellation
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}

	// Context timeout
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	// URL parsing error
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		// Timeout
		if urlErr.Timeout() {
			return "timeout"
		}

		// DNS resolution error
		var dnsErr *net.DNSError
		if errors.As(urlErr.Err, &dnsErr) {
			return "dns_error"
		}

		// Connection error
		var opErr *net.OpError
		if errors.As(urlErr.Err, &opErr) {
			if opErr.Op == "dial" {
				return "connection_error"
			}
		}

		// TLS/SSL error
		if strings.Contains(urlErr.Error(), "tls") || strings.Contains(urlErr.Error(), "certificate") {
			return "tls_error"
		}
	}

	// Other network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "timeout"
		}
		return "network_error"
	}

	// Other errors
	return "unknown_error"
}
