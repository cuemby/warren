package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPChecker_HealthyEndpoint(t *testing.T) {
	// Create test HTTP server that returns 200 OK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("healthy"))
	}))
	defer server.Close()

	// Create checker
	checker := NewHTTPChecker(server.URL)

	// Perform health check
	ctx := context.Background()
	result := checker.Check(ctx)

	// Verify result
	if !result.Healthy {
		t.Errorf("Expected healthy, got unhealthy: %s", result.Message)
	}

	if result.Duration <= 0 {
		t.Error("Expected positive duration")
	}
}

func TestHTTPChecker_UnhealthyEndpoint(t *testing.T) {
	// Create test HTTP server that returns 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	}))
	defer server.Close()

	// Create checker
	checker := NewHTTPChecker(server.URL)

	// Perform health check
	ctx := context.Background()
	result := checker.Check(ctx)

	// Verify result
	if result.Healthy {
		t.Errorf("Expected unhealthy, got healthy: %s", result.Message)
	}
}

func TestHTTPChecker_CustomStatusRange(t *testing.T) {
	// Create test HTTP server that returns 201 Created
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated) // 201
	}))
	defer server.Close()

	// Create checker with custom status range (200-299)
	checker := NewHTTPChecker(server.URL).WithStatusRange(200, 299)

	// Perform health check
	ctx := context.Background()
	result := checker.Check(ctx)

	// Verify result
	if !result.Healthy {
		t.Errorf("Expected healthy for 201 status, got unhealthy: %s", result.Message)
	}
}

func TestHTTPChecker_CustomHeaders(t *testing.T) {
	// Create test HTTP server that checks for custom header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for custom header
		if r.Header.Get("X-Custom-Header") != "test-value" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create checker with custom header
	checker := NewHTTPChecker(server.URL).WithHeader("X-Custom-Header", "test-value")

	// Perform health check
	ctx := context.Background()
	result := checker.Check(ctx)

	// Verify result
	if !result.Healthy {
		t.Errorf("Expected healthy with custom header, got unhealthy: %s", result.Message)
	}
}

func TestHTTPChecker_Timeout(t *testing.T) {
	// Create test HTTP server that sleeps longer than timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create checker with short timeout
	checker := NewHTTPChecker(server.URL).WithTimeout(50 * time.Millisecond)

	// Perform health check
	ctx := context.Background()
	result := checker.Check(ctx)

	// Verify result - should timeout
	if result.Healthy {
		t.Errorf("Expected unhealthy due to timeout, got healthy: %s", result.Message)
	}
}

func TestHTTPChecker_ContextCancellation(t *testing.T) {
	// Create test HTTP server that sleeps
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create checker
	checker := NewHTTPChecker(server.URL)

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	// Perform health check
	result := checker.Check(ctx)

	// Verify result - should fail due to cancelled context
	if result.Healthy {
		t.Errorf("Expected unhealthy due to cancelled context, got healthy: %s", result.Message)
	}
}

func TestHTTPChecker_Type(t *testing.T) {
	checker := NewHTTPChecker("http://example.com")
	if checker.Type() != CheckTypeHTTP {
		t.Errorf("Expected type %s, got %s", CheckTypeHTTP, checker.Type())
	}
}
