package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRegisterComponent(t *testing.T) {
	// Reset health checker
	healthChecker = &HealthChecker{
		components: make(map[string]ComponentHealth),
		startTime:  time.Now(),
	}

	RegisterComponent("test-component", true, "running")

	if len(healthChecker.components) != 1 {
		t.Errorf("expected 1 component, got %d", len(healthChecker.components))
	}

	comp := healthChecker.components["test-component"]
	if !comp.Healthy {
		t.Error("component should be healthy")
	}

	if comp.Message != "running" {
		t.Errorf("expected message 'running', got '%s'", comp.Message)
	}
}

func TestGetHealth_AllHealthy(t *testing.T) {
	// Reset and setup
	healthChecker = &HealthChecker{
		components: make(map[string]ComponentHealth),
		startTime:  time.Now(),
		version:    "1.0.0",
	}

	RegisterComponent("api", true, "")
	RegisterComponent("raft", true, "")

	health := GetHealth()

	if health.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", health.Status)
	}

	if len(health.Components) != 2 {
		t.Errorf("expected 2 components, got %d", len(health.Components))
	}

	if health.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", health.Version)
	}
}

func TestGetHealth_OneUnhealthy(t *testing.T) {
	// Reset and setup
	healthChecker = &HealthChecker{
		components: make(map[string]ComponentHealth),
		startTime:  time.Now(),
	}

	RegisterComponent("api", true, "")
	RegisterComponent("raft", false, "not connected")

	health := GetHealth()

	if health.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got '%s'", health.Status)
	}

	if health.Components["raft"] != "unhealthy: not connected" {
		t.Errorf("unexpected raft status: %s", health.Components["raft"])
	}
}

func TestGetReadiness_AllReady(t *testing.T) {
	// Reset and setup
	healthChecker = &HealthChecker{
		components: make(map[string]ComponentHealth),
		startTime:  time.Now(),
	}

	RegisterComponent("raft", true, "")
	RegisterComponent("containerd", true, "")
	RegisterComponent("api", true, "")

	readiness := GetReadiness()

	if readiness.Status != "ready" {
		t.Errorf("expected status 'ready', got '%s'", readiness.Status)
	}
}

func TestGetReadiness_MissingCriticalComponent(t *testing.T) {
	// Reset and setup
	healthChecker = &HealthChecker{
		components: make(map[string]ComponentHealth),
		startTime:  time.Now(),
	}

	RegisterComponent("api", true, "")
	// raft and containerd not registered

	readiness := GetReadiness()

	if readiness.Status != "not_ready" {
		t.Errorf("expected status 'not_ready', got '%s'", readiness.Status)
	}

	if readiness.Message == "" {
		t.Error("expected message explaining why not ready")
	}
}

func TestGetReadiness_CriticalComponentUnhealthy(t *testing.T) {
	// Reset and setup
	healthChecker = &HealthChecker{
		components: make(map[string]ComponentHealth),
		startTime:  time.Now(),
	}

	RegisterComponent("raft", false, "leader not elected")
	RegisterComponent("containerd", true, "")
	RegisterComponent("api", true, "")

	readiness := GetReadiness()

	if readiness.Status != "not_ready" {
		t.Errorf("expected status 'not_ready', got '%s'", readiness.Status)
	}
}

func TestHealthHandler(t *testing.T) {
	// Reset and setup
	healthChecker = &HealthChecker{
		components: make(map[string]ComponentHealth),
		startTime:  time.Now(),
		version:    "test",
	}

	RegisterComponent("test", true, "")

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := HealthHandler()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var health HealthStatus
	if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("expected healthy status, got %s", health.Status)
	}

	if health.Version != "test" {
		t.Errorf("expected version 'test', got %s", health.Version)
	}
}

func TestHealthHandler_Unhealthy(t *testing.T) {
	// Reset and setup
	healthChecker = &HealthChecker{
		components: make(map[string]ComponentHealth),
		startTime:  time.Now(),
	}

	RegisterComponent("test", false, "broken")

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := HealthHandler()
	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var health HealthStatus
	if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if health.Status != "unhealthy" {
		t.Errorf("expected unhealthy status, got %s", health.Status)
	}
}

func TestReadyHandler(t *testing.T) {
	// Reset and setup
	healthChecker = &HealthChecker{
		components: make(map[string]ComponentHealth),
		startTime:  time.Now(),
	}

	RegisterComponent("raft", true, "")
	RegisterComponent("containerd", true, "")
	RegisterComponent("api", true, "")

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	handler := ReadyHandler()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var readiness HealthStatus
	if err := json.NewDecoder(w.Body).Decode(&readiness); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if readiness.Status != "ready" {
		t.Errorf("expected ready status, got %s", readiness.Status)
	}
}

func TestReadyHandler_NotReady(t *testing.T) {
	// Reset and setup
	healthChecker = &HealthChecker{
		components: make(map[string]ComponentHealth),
		startTime:  time.Now(),
	}

	RegisterComponent("api", true, "")
	// raft not registered

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	handler := ReadyHandler()
	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var readiness HealthStatus
	if err := json.NewDecoder(w.Body).Decode(&readiness); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if readiness.Status != "not_ready" {
		t.Errorf("expected not_ready status, got %s", readiness.Status)
	}
}

func TestLivenessHandler(t *testing.T) {
	// Reset
	healthChecker = &HealthChecker{
		components: make(map[string]ComponentHealth),
		startTime:  time.Now(),
	}

	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()

	handler := LivenessHandler()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "alive" {
		t.Errorf("expected status 'alive', got '%s'", response["status"])
	}

	if response["uptime"] == "" {
		t.Error("uptime should not be empty")
	}
}

func TestUpdateComponent(t *testing.T) {
	// Reset
	healthChecker = &HealthChecker{
		components: make(map[string]ComponentHealth),
		startTime:  time.Now(),
	}

	RegisterComponent("test", true, "ok")
	UpdateComponent("test", false, "error")

	comp := healthChecker.components["test"]
	if comp.Healthy {
		t.Error("component should be unhealthy after update")
	}

	if comp.Message != "error" {
		t.Errorf("expected message 'error', got '%s'", comp.Message)
	}
}
