package metrics

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status     string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp  time.Time         `json:"timestamp"`
	Components map[string]string `json:"components,omitempty"`
	Message    string            `json:"message,omitempty"`
	Version    string            `json:"version,omitempty"`
	Uptime     string            `json:"uptime,omitempty"`
	StartTime  time.Time         `json:"-"`
}

var (
	healthChecker = &HealthChecker{
		components: make(map[string]ComponentHealth),
		startTime:  time.Now(),
	}
)

// ComponentHealth tracks the health of a single component
type ComponentHealth struct {
	Name    string
	Healthy bool
	Message string
	Updated time.Time
}

// HealthChecker manages health checks for various components
type HealthChecker struct {
	mu         sync.RWMutex
	components map[string]ComponentHealth
	startTime  time.Time
	version    string
}

// SetVersion sets the version string for health responses
func SetVersion(version string) {
	healthChecker.mu.Lock()
	defer healthChecker.mu.Unlock()
	healthChecker.version = version
}

// RegisterComponent registers a component for health checking
func RegisterComponent(name string, healthy bool, message string) {
	healthChecker.mu.Lock()
	defer healthChecker.mu.Unlock()

	healthChecker.components[name] = ComponentHealth{
		Name:    name,
		Healthy: healthy,
		Message: message,
		Updated: time.Now(),
	}
}

// UpdateComponent updates the health status of a component
func UpdateComponent(name string, healthy bool, message string) {
	RegisterComponent(name, healthy, message) // Same implementation
}

// GetHealth returns the overall health status
func GetHealth() HealthStatus {
	healthChecker.mu.RLock()
	defer healthChecker.mu.RUnlock()

	status := "healthy"
	components := make(map[string]string)

	for name, comp := range healthChecker.components {
		if !comp.Healthy {
			status = "unhealthy"
			components[name] = "unhealthy: " + comp.Message
		} else {
			components[name] = "healthy"
		}
	}

	uptime := time.Since(healthChecker.startTime)

	return HealthStatus{
		Status:     status,
		Timestamp:  time.Now(),
		Components: components,
		Version:    healthChecker.version,
		Uptime:     uptime.String(),
		StartTime:  healthChecker.startTime,
	}
}

// GetReadiness returns readiness status (checks if critical components are ready)
func GetReadiness() HealthStatus {
	healthChecker.mu.RLock()
	defer healthChecker.mu.RUnlock()

	status := "ready"
	message := ""
	components := make(map[string]string)

	// Check critical components
	criticalComponents := []string{"raft", "containerd", "api"}

	for _, name := range criticalComponents {
		if comp, exists := healthChecker.components[name]; exists {
			if !comp.Healthy {
				status = "not_ready"
				message = "waiting for " + name
				components[name] = "not ready: " + comp.Message
			} else {
				components[name] = "ready"
			}
		} else {
			// Component not registered yet
			status = "not_ready"
			message = "waiting for " + name + " initialization"
			components[name] = "not registered"
		}
	}

	uptime := time.Since(healthChecker.startTime)

	return HealthStatus{
		Status:     status,
		Timestamp:  time.Now(),
		Components: components,
		Message:    message,
		Version:    healthChecker.version,
		Uptime:     uptime.String(),
		StartTime:  healthChecker.startTime,
	}
}

// HealthHandler returns an HTTP handler for the /health endpoint
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := GetHealth()

		w.Header().Set("Content-Type", "application/json")

		// Set appropriate status code
		statusCode := http.StatusOK
		if health.Status == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}
		w.WriteHeader(statusCode)

		_ = json.NewEncoder(w).Encode(health)
	}
}

// ReadyHandler returns an HTTP handler for the /ready endpoint
func ReadyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		readiness := GetReadiness()

		w.Header().Set("Content-Type", "application/json")

		// Set appropriate status code
		statusCode := http.StatusOK
		if readiness.Status != "ready" {
			statusCode = http.StatusServiceUnavailable
		}
		w.WriteHeader(statusCode)

		_ = json.NewEncoder(w).Encode(readiness)
	}
}

// LivenessHandler returns a simple liveness check (always returns 200 if process is running)
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "alive",
			"uptime": time.Since(healthChecker.startTime).String(),
		})
	}
}
