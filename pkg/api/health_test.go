package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHealthHandler tests the /health endpoint
func TestHealthHandler(t *testing.T) {
	hs := NewHealthServer(nil) // nil manager is OK for health check

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET request succeeds",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request fails",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "PUT request fails",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "DELETE request fails",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			w := httptest.NewRecorder()

			hs.healthHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				// Verify JSON response
				var response HealthResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, "healthy", response.Status)
				assert.NotZero(t, response.Timestamp)
			}
		})
	}
}

// TestHealthHandlerJSONFormat tests the health endpoint JSON response format
func TestHealthHandlerJSONFormat(t *testing.T) {
	hs := NewHealthServer(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	hs.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response HealthResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	// Verify required fields
	assert.Equal(t, "healthy", response.Status)
	assert.False(t, response.Timestamp.IsZero())
	assert.NotEmpty(t, response.Version)
}

// TestReadyHandlerNoManager tests readiness endpoint with no manager
func TestReadyHandlerNoManager(t *testing.T) {
	hs := NewHealthServer(nil)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	hs.readyHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ReadyResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, "not ready", response.Status)
	assert.Contains(t, response.Checks["raft"], "not initialized")
	assert.Contains(t, response.Checks["storage"], "not initialized")
	assert.NotEmpty(t, response.Message)
}

// TestReadyHandlerMethodValidation tests readiness endpoint HTTP method validation
func TestReadyHandlerMethodValidation(t *testing.T) {
	hs := NewHealthServer(nil)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET request accepted",
			method:         http.MethodGet,
			expectedStatus: http.StatusServiceUnavailable, // Not ready due to nil manager
		},
		{
			name:           "POST request rejected",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "PUT request rejected",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/ready", nil)
			w := httptest.NewRecorder()

			hs.readyHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestReadyHandlerJSONFormat tests the readiness endpoint JSON response format
func TestReadyHandlerJSONFormat(t *testing.T) {
	hs := NewHealthServer(nil)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	hs.readyHandler(w, req)

	var response ReadyResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	// Verify response structure
	assert.NotEmpty(t, response.Status)
	assert.False(t, response.Timestamp.IsZero())
	assert.NotNil(t, response.Checks)
	assert.NotEmpty(t, response.Checks)

	// Verify checks contain expected keys
	assert.Contains(t, response.Checks, "raft")
	assert.Contains(t, response.Checks, "storage")
}

// TestNewHealthServer tests health server creation
func TestNewHealthServer(t *testing.T) {
	hs := NewHealthServer(nil)

	assert.NotNil(t, hs)
	assert.NotNil(t, hs.mux)
	assert.Nil(t, hs.manager) // Nil manager is allowed

	// Verify routes are registered by testing requests
	tests := []struct {
		path           string
		expectedStatus int
	}{
		{path: "/health", expectedStatus: http.StatusOK},
		{path: "/ready", expectedStatus: http.StatusServiceUnavailable},
		{path: "/metrics", expectedStatus: http.StatusOK}, // Metrics endpoint always returns 200
		{path: "/nonexistent", expectedStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			hs.mux.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Path: %s", tt.path)
		})
	}
}

// TestGetHandler tests the GetHandler method
func TestGetHandler(t *testing.T) {
	hs := NewHealthServer(nil)

	handler := hs.GetHandler()
	assert.NotNil(t, handler)

	// Verify the handler works
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestHealthServerConcurrency tests concurrent requests to health endpoints
func TestHealthServerConcurrency(t *testing.T) {
	hs := NewHealthServer(nil)

	done := make(chan bool, 20)

	// Make 10 concurrent health requests
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()
			hs.healthHandler(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// Make 10 concurrent ready requests
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			w := httptest.NewRecorder()
			hs.readyHandler(w, req)
			// Status can be 200 or 503 depending on manager state
			assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, w.Code)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

// Benchmark tests for performance tracking
func BenchmarkHealthHandler(b *testing.B) {
	hs := NewHealthServer(nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		hs.healthHandler(w, req)
	}
}

func BenchmarkReadyHandler(b *testing.B) {
	hs := NewHealthServer(nil)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		hs.readyHandler(w, req)
	}
}
