package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/metrics"
)

// HealthServer provides HTTP health check endpoints
type HealthServer struct {
	manager *manager.Manager
	mux     *http.ServeMux
}

// NewHealthServer creates a new health check HTTP server
func NewHealthServer(mgr *manager.Manager) *HealthServer {
	mux := http.NewServeMux()
	hs := &HealthServer{
		manager: mgr,
		mux:     mux,
	}

	// Register endpoints
	mux.HandleFunc("/health", hs.healthHandler)
	mux.HandleFunc("/ready", hs.readyHandler)
	mux.Handle("/metrics", metrics.Handler())

	return hs
}

// Start starts the health check HTTP server
func (hs *HealthServer) Start(addr string) error {
	server := &http.Server{
		Addr:         addr,
		Handler:      hs.mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
}

// ReadyResponse represents the readiness check response
type ReadyResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
	Message   string            `json:"message,omitempty"`
}

// healthHandler implements the /health endpoint
// This is a simple liveness check - returns 200 if the process is alive
func (hs *HealthServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.3.1", // TODO: Get from build info
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// readyHandler implements the /ready endpoint
// This checks if the service is ready to accept traffic
func (hs *HealthServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	checks := make(map[string]string)
	ready := true
	var message string

	// Check 1: Raft cluster
	if hs.manager != nil {
		if hs.manager.IsLeader() {
			checks["raft"] = "leader"
		} else {
			leaderAddr := hs.manager.LeaderAddr()
			if leaderAddr != "" {
				checks["raft"] = fmt.Sprintf("follower (leader: %s)", leaderAddr)
			} else {
				checks["raft"] = "no leader elected"
				ready = false
				message = "Waiting for leader election"
			}
		}
	} else {
		checks["raft"] = "not initialized"
		ready = false
		message = "Manager not initialized"
	}

	// Check 2: Storage (basic check - manager should have store)
	if hs.manager != nil {
		// Attempt a simple read operation to verify storage
		_, err := hs.manager.ListServices()
		if err != nil {
			checks["storage"] = fmt.Sprintf("error: %v", err)
			ready = false
			if message == "" {
				message = "Storage not accessible"
			}
		} else {
			checks["storage"] = "ok"
		}
	} else {
		checks["storage"] = "not initialized"
		ready = false
	}

	// Check 3: Event broker
	// TODO: Add event broker health check when available

	// Prepare response
	status := "ready"
	statusCode := http.StatusOK

	if !ready {
		status = "not ready"
		statusCode = http.StatusServiceUnavailable
	}

	response := ReadyResponse{
		Status:    status,
		Timestamp: time.Now(),
		Checks:    checks,
		Message:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}

// GetHandler returns the HTTP handler for embedding in other servers
func (hs *HealthServer) GetHandler() http.Handler {
	return hs.mux
}
