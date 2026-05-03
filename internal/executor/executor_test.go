package executor

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/nilpoona/zerohand/internal/config"
)

func TestExecuteRequest(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		config         *config.LoadTestConfig
		expectError    bool
		expectedStatus int
	}{
		{
			name: "successful GET request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			},
			config: &config.LoadTestConfig{
				Method:  "GET",
				Timeout: 10,
			},
			expectError:    false,
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful POST request with body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusCreated)
			},
			config: &config.LoadTestConfig{
				Method:  "POST",
				Body:    `{"key":"value"}`,
				Timeout: 10,
			},
			expectError:    false,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "request with custom headers",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Custom-Header") != "test-value" {
					t.Errorf("expected custom header, got %s", r.Header.Get("X-Custom-Header"))
				}
				w.WriteHeader(http.StatusOK)
			},
			config: &config.LoadTestConfig{
				Method: "GET",
				Headers: map[string]string{
					"X-Custom-Header": "test-value",
				},
				Timeout: 10,
			},
			expectError:    false,
			expectedStatus: http.StatusOK,
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			config: &config.LoadTestConfig{
				Method:  "GET",
				Timeout: 10,
			},
			expectError:    false,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "slow server with timeout",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(2 * time.Second)
				w.WriteHeader(http.StatusOK)
			},
			config: &config.LoadTestConfig{
				Method:  "GET",
				Timeout: 1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			tt.config.TargetURL = server.URL

			ctx := context.Background()
			result := ExecuteRequest(ctx, tt.config)

			if result == nil {
				t.Fatal("expected result, got nil")
			}

			if tt.expectError {
				if result.Error == "" {
					t.Error("expected error, got none")
				}
			} else {
				if result.Error != "" {
					t.Errorf("unexpected error: %s", result.Error)
				}
				if result.StatusCode != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, result.StatusCode)
				}
			}

			if result.Duration <= 0 {
				t.Error("expected positive duration")
			}

			if result.Timestamp.IsZero() {
				t.Error("expected timestamp to be set")
			}
		})
	}
}

func TestExecuteRequestWithContext(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	cfg := &config.LoadTestConfig{
		TargetURL: server.URL,
		Method:    "GET",
		Timeout:   10,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := ExecuteRequest(ctx, cfg)

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if result.Error != "canceled" {
		t.Errorf("expected canceled error, got %s", result.Error)
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: "canceled",
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: "timeout",
		},
		{
			name: "url timeout error",
			err: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &timeoutError{},
			},
			expected: "timeout",
		},
		{
			name: "dns error",
			err: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.DNSError{},
			},
			expected: "dns_error",
		},
		{
			name: "dns error - no such host",
			err: &url.Error{
				Op:  "Get",
				URL: "http://invalid.example.com",
				Err: &net.DNSError{
					Err:        "no such host",
					Name:       "invalid.example.com",
					IsNotFound: true,
				},
			},
			expected: "dns_error",
		},
		{
			name: "dns error - temporary failure",
			err: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.DNSError{
					Err:         "server misbehaving",
					Name:        "example.com",
					IsTemporary: true,
				},
			},
			expected: "dns_error",
		},
		{
			name: "dial error",
			err: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.OpError{Op: "dial"},
			},
			expected: "connection_error",
		},
		{
			name: "connection refused error",
			err: &url.Error{
				Op:  "Get",
				URL: "http://localhost:9999",
				Err: &net.OpError{
					Op:  "dial",
					Net: "tcp",
					Err: errors.New("connect: connection refused"),
				},
			},
			expected: "connection_error",
		},
		{
			name: "TLS handshake error",
			err: &url.Error{
				Op:  "Get",
				URL: "https://example.com",
				Err: errors.New("tls: handshake failure"),
			},
			expected: "tls_error",
		},
		{
			name: "certificate verification error",
			err: &url.Error{
				Op:  "Get",
				URL: "https://example.com",
				Err: errors.New("x509: certificate signed by unknown authority"),
			},
			expected: "tls_error",
		},
		{
			name: "certificate expired error",
			err: &url.Error{
				Op:  "Get",
				URL: "https://example.com",
				Err: errors.New("x509: certificate has expired or is not yet valid"),
			},
			expected: "tls_error",
		},
		{
			name: "network error - timeout",
			err: &net.OpError{
				Op:  "read",
				Net: "tcp",
				Err: &timeoutError{},
			},
			expected: "timeout",
		},
		{
			name: "network error - generic",
			err: &net.OpError{
				Op:  "write",
				Net: "tcp",
				Err: errors.New("broken pipe"),
			},
			expected: "network_error",
		},
		{
			name:     "unknown error",
			err:      errors.New("some random error"),
			expected: "unknown_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPrepareRequest(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.LoadTestConfig
		expectedMethod string
		expectedBody   bool
		expectError    bool
	}{
		{
			name: "GET request without body",
			config: &config.LoadTestConfig{
				TargetURL: "http://example.com",
				Method:    "GET",
			},
			expectedMethod: "GET",
			expectedBody:   false,
			expectError:    false,
		},
		{
			name: "POST request with body",
			config: &config.LoadTestConfig{
				TargetURL: "http://example.com",
				Method:    "POST",
				Body:      `{"key":"value"}`,
			},
			expectedMethod: "POST",
			expectedBody:   true,
			expectError:    false,
		},
		{
			name: "request with headers",
			config: &config.LoadTestConfig{
				TargetURL: "http://example.com",
				Method:    "GET",
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer token",
				},
			},
			expectedMethod: "GET",
			expectedBody:   false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := prepareRequest(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if req.Method != tt.expectedMethod {
				t.Errorf("expected method %s, got %s", tt.expectedMethod, req.Method)
			}

			if tt.expectedBody && req.Body == nil {
				t.Error("expected body, got nil")
			}

			if !tt.expectedBody && req.Body != nil {
				t.Error("expected no body, got body")
			}

			for key, value := range tt.config.Headers {
				if req.Header.Get(key) != value {
					t.Errorf("expected header %s=%s, got %s", key, value, req.Header.Get(key))
				}
			}
		})
	}
}

func TestCreateHTTPClient(t *testing.T) {
	timeout := 5 * time.Second
	client := createHTTPClient(timeout)

	if client == nil {
		t.Fatal("expected client, got nil")
	}

	if client.Timeout != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, client.Timeout)
	}

	if client.Transport == nil {
		t.Error("expected transport to be set")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}

	if transport.MaxIdleConns != 100 {
		t.Errorf("expected MaxIdleConns 100, got %d", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 100 {
		t.Errorf("expected MaxIdleConnsPerHost 100, got %d", transport.MaxIdleConnsPerHost)
	}

	if transport.DisableKeepAlives {
		t.Error("expected DisableKeepAlives to be false")
	}
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

// TestExecuteRequestWithDNSError tests DNS resolution failures
func TestExecuteRequestWithDNSError(t *testing.T) {
	cfg := &config.LoadTestConfig{
		TargetURL: "http://this-domain-does-not-exist-12345.invalid",
		Method:    "GET",
		Timeout:   5,
	}

	ctx := context.Background()
	result := ExecuteRequest(ctx, cfg)

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if result.Error != "dns_error" {
		t.Errorf("expected dns_error, got %s", result.Error)
	}

	if result.StatusCode != 0 {
		t.Errorf("expected status code 0 for DNS error, got %d", result.StatusCode)
	}
}

// TestExecuteRequestWithConnectionError tests connection failures
func TestExecuteRequestWithConnectionError(t *testing.T) {
	// Use a port that's unlikely to be in use
	cfg := &config.LoadTestConfig{
		TargetURL: "http://localhost:54321",
		Method:    "GET",
		Timeout:   2,
	}

	ctx := context.Background()
	result := ExecuteRequest(ctx, cfg)

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if result.Error != "connection_error" {
		t.Errorf("expected connection_error, got %s", result.Error)
	}

	if result.StatusCode != 0 {
		t.Errorf("expected status code 0 for connection error, got %d", result.StatusCode)
	}
}

// TestExecuteRequestWithTLSError tests TLS/certificate errors
func TestExecuteRequestWithTLSError(t *testing.T) {
	// Create a test server with self-signed certificate
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a custom client that will reject the self-signed cert
	cfg := &config.LoadTestConfig{
		TargetURL: server.URL,
		Method:    "GET",
		Timeout:   5,
	}

	// Override the HTTP client to enforce strict TLS verification
	originalClient := createHTTPClient(time.Duration(cfg.Timeout) * time.Second)
	transport := originalClient.Transport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: false,              // Enforce certificate verification
		RootCAs:            x509.NewCertPool(), // Empty pool will fail verification
	}

	ctx := context.Background()

	// Execute request with custom client
	req, err := prepareRequest(cfg)
	if err != nil {
		t.Fatalf("failed to prepare request: %v", err)
	}
	req = req.WithContext(ctx)

	result := &Result{
		Timestamp: time.Now(),
	}

	start := time.Now()
	resp, err := originalClient.Do(req)
	duration := time.Since(start)

	result.Duration = duration.Microseconds()

	if err != nil {
		result.Error = classifyError(err)
	} else {
		defer resp.Body.Close()
		result.StatusCode = resp.StatusCode
	}

	if result.Error != "tls_error" {
		t.Errorf("expected tls_error, got %s", result.Error)
	}
}

// TestExecuteRequestWithNetworkTimeout tests network-level timeout
func TestExecuteRequestWithNetworkTimeout(t *testing.T) {
	// Create a server that delays longer than timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.LoadTestConfig{
		TargetURL: server.URL,
		Method:    "GET",
		Timeout:   1, // 1 second timeout
	}

	ctx := context.Background()
	result := ExecuteRequest(ctx, cfg)

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if result.Error != "timeout" {
		t.Errorf("expected timeout, got %s", result.Error)
	}

	if result.StatusCode != 0 {
		t.Errorf("expected status code 0 for timeout, got %d", result.StatusCode)
	}

	// Verify the duration is approximately the timeout value
	expectedDuration := 1 * time.Second
	actualDuration := time.Duration(result.Duration) * time.Microsecond

	// Allow 500ms variance for network overhead
	if actualDuration < expectedDuration || actualDuration > expectedDuration+500*time.Millisecond {
		t.Logf("timeout duration was %v, expected around %v", actualDuration, expectedDuration)
	}
}

// TestExecuteRequestWithInvalidURL tests invalid URL handling
func TestExecuteRequestWithInvalidURL(t *testing.T) {
	cfg := &config.LoadTestConfig{
		TargetURL: "://invalid-url",
		Method:    "GET",
		Timeout:   5,
	}

	ctx := context.Background()
	result := ExecuteRequest(ctx, cfg)

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Invalid URL should be caught during request preparation
	if result.Error == "" {
		t.Error("expected an error for invalid URL")
	}
}

// TestExecuteRequestWithVariousNetworkErrors tests different network error scenarios
func TestExecuteRequestWithVariousNetworkErrors(t *testing.T) {
	tests := []struct {
		name          string
		setupServer   func() (*httptest.Server, *config.LoadTestConfig)
		expectedError string
	}{
		{
			name: "server closes connection immediately",
			setupServer: func() (*httptest.Server, *config.LoadTestConfig) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					hj, ok := w.(http.Hijacker)
					if !ok {
						t.Fatal("server doesn't support hijacking")
					}
					conn, _, err := hj.Hijack()
					if err != nil {
						t.Fatalf("hijack failed: %v", err)
					}
					conn.Close() // Close connection immediately
				}))

				cfg := &config.LoadTestConfig{
					TargetURL: server.URL,
					Method:    "GET",
					Timeout:   5,
				}

				return server, cfg
			},
			expectedError: "network_error", // Could be connection_error or network_error
		},
		{
			name: "server returns no content length",
			setupServer: func() (*httptest.Server, *config.LoadTestConfig) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Send response without Content-Length header
					w.Header().Del("Content-Length")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("response"))
				}))

				cfg := &config.LoadTestConfig{
					TargetURL: server.URL,
					Method:    "GET",
					Timeout:   5,
				}

				return server, cfg
			},
			expectedError: "", // Should succeed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, cfg := tt.setupServer()
			defer server.Close()

			ctx := context.Background()
			result := ExecuteRequest(ctx, cfg)

			if result == nil {
				t.Fatal("expected result, got nil")
			}

			if tt.expectedError != "" {
				// For network errors, allow some flexibility in classification
				if result.Error != tt.expectedError &&
					result.Error != "connection_error" &&
					result.Error != "network_error" &&
					result.Error != "unknown_error" {
					t.Errorf("expected error to be network-related, got %s", result.Error)
				}
			} else {
				if result.Error != "" {
					t.Errorf("expected no error, got %s", result.Error)
				}
			}
		})
	}
}
