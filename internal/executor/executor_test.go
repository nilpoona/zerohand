package executor

import (
	"context"
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
			name: "dial error",
			err: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.OpError{Op: "dial"},
			},
			expected: "connection_error",
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
