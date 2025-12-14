package config

import (
	"strings"
	"testing"
)

func TestNewLoadTestConfig(t *testing.T) {
	cfg := NewLoadTestConfig()

	if cfg.Method != "GET" {
		t.Errorf("expected Method to be GET, got %s", cfg.Method)
	}

	if cfg.RPS != 10 {
		t.Errorf("expected RPS to be 10, got %d", cfg.RPS)
	}

	if cfg.Duration != 10 {
		t.Errorf("expected Duration to be 10, got %d", cfg.Duration)
	}

	if cfg.Timeout != 10 {
		t.Errorf("expected Timeout to be 10, got %d", cfg.Timeout)
	}

	if cfg.Headers == nil {
		t.Error("expected Headers to be initialized")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *LoadTestConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &LoadTestConfig{
				TargetURL: "https://example.com",
				Method:    "GET",
				RPS:       10,
				Duration:  10,
				Timeout:   10,
			},
			wantErr: false,
		},
		{
			name: "empty URL",
			config: &LoadTestConfig{
				TargetURL: "",
				Method:    "GET",
				RPS:       10,
				Duration:  10,
				Timeout:   10,
			},
			wantErr: true,
			errMsg:  "target URL is required",
		},
		{
			name: "invalid URL",
			config: &LoadTestConfig{
				TargetURL: "not a url",
				Method:    "GET",
				RPS:       10,
				Duration:  10,
				Timeout:   10,
			},
			wantErr: true,
			errMsg:  "URL scheme must be http or https",
		},
		{
			name: "invalid scheme",
			config: &LoadTestConfig{
				TargetURL: "ftp://example.com",
				Method:    "GET",
				RPS:       10,
				Duration:  10,
				Timeout:   10,
			},
			wantErr: true,
			errMsg:  "URL scheme must be http or https",
		},
		{
			name: "invalid method",
			config: &LoadTestConfig{
				TargetURL: "https://example.com",
				Method:    "INVALID",
				RPS:       10,
				Duration:  10,
				Timeout:   10,
			},
			wantErr: true,
			errMsg:  "invalid HTTP method",
		},
		{
			name: "lowercase method",
			config: &LoadTestConfig{
				TargetURL: "https://example.com",
				Method:    "post",
				RPS:       10,
				Duration:  10,
				Timeout:   10,
			},
			wantErr: false,
		},
		{
			name: "zero RPS",
			config: &LoadTestConfig{
				TargetURL: "https://example.com",
				Method:    "GET",
				RPS:       0,
				Duration:  10,
				Timeout:   10,
			},
			wantErr: true,
			errMsg:  "RPS must be greater than 0",
		},
		{
			name: "negative RPS",
			config: &LoadTestConfig{
				TargetURL: "https://example.com",
				Method:    "GET",
				RPS:       -1,
				Duration:  10,
				Timeout:   10,
			},
			wantErr: true,
			errMsg:  "RPS must be greater than 0",
		},
		{
			name: "zero duration",
			config: &LoadTestConfig{
				TargetURL: "https://example.com",
				Method:    "GET",
				RPS:       10,
				Duration:  0,
				Timeout:   10,
			},
			wantErr: true,
			errMsg:  "duration must be greater than 0",
		},
		{
			name: "negative duration",
			config: &LoadTestConfig{
				TargetURL: "https://example.com",
				Method:    "GET",
				RPS:       10,
				Duration:  -1,
				Timeout:   10,
			},
			wantErr: true,
			errMsg:  "duration must be greater than 0",
		},
		{
			name: "zero timeout",
			config: &LoadTestConfig{
				TargetURL: "https://example.com",
				Method:    "GET",
				RPS:       10,
				Duration:  10,
				Timeout:   0,
			},
			wantErr: true,
			errMsg:  "timeout must be greater than 0",
		},
		{
			name: "negative timeout",
			config: &LoadTestConfig{
				TargetURL: "https://example.com",
				Method:    "GET",
				RPS:       10,
				Duration:  10,
				Timeout:   -1,
			},
			wantErr: true,
			errMsg:  "timeout must be greater than 0",
		},
		{
			name: "all valid methods",
			config: &LoadTestConfig{
				TargetURL: "https://example.com",
				Method:    "DELETE",
				RPS:       10,
				Duration:  10,
				Timeout:   10,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateMethodNormalization(t *testing.T) {
	cfg := &LoadTestConfig{
		TargetURL: "https://example.com",
		Method:    "post",
		RPS:       10,
		Duration:  10,
		Timeout:   10,
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Method != "POST" {
		t.Errorf("expected Method to be normalized to POST, got %s", cfg.Method)
	}
}
