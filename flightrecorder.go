package flightrecorder

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/exp/trace"
)

var (
	once    sync.Once
	service *Service
)

// Service manages the flight recorder and HTTP endpoints
type Service struct {
	recorder *trace.FlightRecorder
	mu       sync.RWMutex
	period   time.Duration
	size     int
}

// StatusResponse represents the status of the flight recorder
type StatusResponse struct {
	Enabled bool          `json:"enabled"`
	Period  time.Duration `json:"period"`
	Size    int           `json:"size"`
}

// UpdateRequest represents the update request payload
type UpdateRequest struct {
	Period *time.Duration `json:"period,omitempty"`
	Size   *int           `json:"size,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// InitService creates a new global flight recorder service.
func InitService() *Service {
	once.Do(func() {
		service = &Service{
			recorder: trace.NewFlightRecorder(),
			period:   1 * time.Second,  // Default period
			size:     64 * 1024 * 1024, // Default 64MB
		}
	})
	return service
}

// Status returns the current status of the flight recorder
func (s *Service) Status() StatusResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return StatusResponse{
		Enabled: s.recorder.Enabled(),
		Period:  s.period,
		Size:    s.size,
	}
}

// Start starts the flight recorder
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.recorder.Enabled() {
		return fmt.Errorf("flight recorder is already running")
	}

	s.recorder.SetPeriod(s.period)
	s.recorder.SetSize(s.size)

	return s.recorder.Start()
}

// Stop stops the flight recorder
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.recorder.Enabled() {
		return fmt.Errorf("flight recorder is not running")
	}

	return s.recorder.Stop()
}

// Snapshot returns the current snapshot of the flight recorder
func (s *Service) Snapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.recorder.Enabled() {
		return nil, fmt.Errorf("flight recorder is not running")
	}

	var buf bytes.Buffer
	_, err := s.recorder.WriteTo(&buf)
	if err == nil {
		return buf.Bytes(), nil
	}

	if errors.Is(err, trace.ErrSnapshotActive) {
		return nil, fmt.Errorf("flight recorder snapshot already in progress")
	} else {
		return nil, fmt.Errorf("failed to write snapshot: %w", err)
	}
}

// Update updates the flight recorder configuration
func (s *Service) Update(req UpdateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.Period != nil {
		s.period = *req.Period
		if s.recorder.Enabled() {
			s.recorder.SetPeriod(s.period)
		}
	}

	if req.Size != nil {
		s.size = *req.Size
		if s.recorder.Enabled() {
			s.recorder.SetSize(s.size)
		}
	}

	return nil
}

// HTTP handlers
func (s *Service) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.Status()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Service) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := s.Start()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Service) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := s.Stop()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Service) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot, err := s.Snapshot()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(snapshot)
}

func (s *Service) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON payload"})
		return
	}

	err := s.Update(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
}

// RegisterHandlers registers the flight recorder HTTP handlers to the given mux
func (s *Service) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/recorder/status", s.handleStatus)
	mux.HandleFunc("/recorder/start", s.handleStart)
	mux.HandleFunc("/recorder/stop", s.handleStop)
	mux.HandleFunc("/recorder/snapshot", s.handleSnapshot)
	mux.HandleFunc("/recorder/update", s.handleUpdate)
}

// RegisterHandlersWithPrefix registers the flight recorder HTTP handlers with a custom prefix
func (s *Service) RegisterHandlersWithPrefix(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/status", s.handleStatus)
	mux.HandleFunc(prefix+"/start", s.handleStart)
	mux.HandleFunc(prefix+"/stop", s.handleStop)
	mux.HandleFunc(prefix+"/snapshot", s.handleSnapshot)
	mux.HandleFunc(prefix+"/update", s.handleUpdate)
}
