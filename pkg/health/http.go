package health

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPChecker performs HTTP-based health checks
type HTTPChecker struct {
	// URL is the full HTTP URL to check (e.g., "http://container-ip:8080/health")
	URL string

	// Method is the HTTP method to use (default: GET)
	Method string

	// Headers are custom HTTP headers to include in the request
	Headers map[string]string

	// ExpectedStatusMin is the minimum acceptable HTTP status code (default: 200)
	ExpectedStatusMin int

	// ExpectedStatusMax is the maximum acceptable HTTP status code (default: 399)
	ExpectedStatusMax int

	// Client is the HTTP client to use (allows custom configuration)
	Client *http.Client
}

// NewHTTPChecker creates a new HTTP health checker
func NewHTTPChecker(url string) *HTTPChecker {
	return &HTTPChecker{
		URL:               url,
		Method:            "GET",
		Headers:           make(map[string]string),
		ExpectedStatusMin: 200,
		ExpectedStatusMax: 399,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Check performs the HTTP health check
func (h *HTTPChecker) Check(ctx context.Context) Result {
	start := time.Now()

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, h.Method, h.URL, nil)
	if err != nil {
		return Result{
			Healthy:   false,
			Message:   fmt.Sprintf("failed to create request: %v", err),
			CheckedAt: start,
			Duration:  time.Since(start),
		}
	}

	// Add custom headers
	for key, value := range h.Headers {
		req.Header.Set(key, value)
	}

	// Perform HTTP request
	resp, err := h.Client.Do(req)
	if err != nil {
		return Result{
			Healthy:   false,
			Message:   fmt.Sprintf("request failed: %v", err),
			CheckedAt: start,
			Duration:  time.Since(start),
		}
	}
	defer resp.Body.Close()

	// Check status code
	healthy := resp.StatusCode >= h.ExpectedStatusMin && resp.StatusCode <= h.ExpectedStatusMax

	message := fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	if !healthy {
		message = fmt.Sprintf("%s (expected %d-%d)", message, h.ExpectedStatusMin, h.ExpectedStatusMax)
	}

	return Result{
		Healthy:   healthy,
		Message:   message,
		CheckedAt: start,
		Duration:  time.Since(start),
	}
}

// Type returns the health check type
func (h *HTTPChecker) Type() CheckType {
	return CheckTypeHTTP
}

// WithMethod sets the HTTP method
func (h *HTTPChecker) WithMethod(method string) *HTTPChecker {
	h.Method = method
	return h
}

// WithHeader adds a custom HTTP header
func (h *HTTPChecker) WithHeader(key, value string) *HTTPChecker {
	h.Headers[key] = value
	return h
}

// WithStatusRange sets the expected status code range
func (h *HTTPChecker) WithStatusRange(min, max int) *HTTPChecker {
	h.ExpectedStatusMin = min
	h.ExpectedStatusMax = max
	return h
}

// WithTimeout sets the HTTP client timeout
func (h *HTTPChecker) WithTimeout(timeout time.Duration) *HTTPChecker {
	h.Client.Timeout = timeout
	return h
}
