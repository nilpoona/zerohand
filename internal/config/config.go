package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// LoadTestConfig holds the configuration for a load test
type LoadTestConfig struct {
	TestID    string            `json:"test_id"`
	LambdaID  int               `json:"lambda_id"`
	TargetURL string            `json:"target_url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	RPS       int               `json:"rps"`      // Requests per second
	Duration  int               `json:"duration"` // Duration in seconds
	Timeout   int               `json:"timeout"`  // Timeout in seconds
}

// NewLoadTestConfig creates a new LoadTestConfig with default values
func NewLoadTestConfig() *LoadTestConfig {
	return &LoadTestConfig{
		Method:   "GET",
		Headers:  make(map[string]string),
		RPS:      10,
		Duration: 10,
		Timeout:  10,
	}
}

// Validate checks the validity of the configuration
func (c *LoadTestConfig) Validate() error {
	// Validate URL
	if c.TargetURL == "" {
		return errors.New("target URL is required")
	}

	parsedURL, err := url.Parse(c.TargetURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got: %s", parsedURL.Scheme)
	}

	// Validate HTTP method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true,
		"DELETE": true, "HEAD": true, "OPTIONS": true,
	}

	methodUpper := strings.ToUpper(c.Method)
	if !validMethods[methodUpper] {
		return fmt.Errorf("invalid HTTP method: %s", c.Method)
	}
	c.Method = methodUpper // Normalize

	// Validate RPS
	if c.RPS <= 0 {
		return errors.New("RPS must be greater than 0")
	}

	// Validate Duration
	if c.Duration <= 0 {
		return errors.New("duration must be greater than 0")
	}

	// Validate Timeout
	if c.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}

	return nil
}
