package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/rclsilver-org/external-dns-usg-dns-api/internal/provider"
	"github.com/rclsilver-org/external-dns-usg-dns-api/internal/webhook"
)

const (
	mediaTypeFormat = "application/external.dns.webhook+json;version=1"
)

// Server implements the external-dns webhook HTTP server
type Server struct {
	provider   *provider.Provider
	port       int
	healthPort int
}

// NewServer creates a new webhook server
func NewServer(provider *provider.Provider, port, healthPort int) *Server {
	return &Server{
		provider:   provider,
		port:       port,
		healthPort: healthPort,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Start health server in a goroutine
	go func() {
		if err := s.startHealthServer(); err != nil {
			log.Fatalf("Health server failed: %v", err)
		}
	}()

	// Start main API server
	return s.startAPIServer()
}

// startAPIServer starts the main API server
func (s *Server) startAPIServer() error {
	mux := http.NewServeMux()

	// Provider endpoints
	mux.HandleFunc("/", s.negotiate)
	mux.HandleFunc("/records", s.handleRecords)
	mux.HandleFunc("/adjustendpoints", s.adjustEndpoints)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Starting API server on %s", addr)
	return http.ListenAndServe(addr, s.loggingMiddleware(mux))
}

// startHealthServer starts the health check server
func (s *Server) startHealthServer() error {
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/readyz", s.healthz)
	mux.HandleFunc("/livez", s.healthz)

	addr := fmt.Sprintf(":%d", s.healthPort)
	log.Printf("Starting health server on %s", addr)
	return http.ListenAndServe(addr, s.loggingMiddleware(mux))
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) negotiate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check Accept header
	accept := r.Header.Get("Accept")
	if accept != "" && accept != mediaTypeFormat && accept != "*/*" {
		http.Error(w, "Not acceptable", http.StatusNotAcceptable)
		return
	}

	domainFilter := s.provider.GetDomainFilter()

	w.Header().Set("Content-Type", mediaTypeFormat)
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(domainFilter); err != nil {
		log.Printf("Failed to encode domain filter: %v", err)
	}
}

func (s *Server) handleRecords(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getRecords(w, r)
	case http.MethodPost:
		s.applyChanges(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getRecords(w http.ResponseWriter, r *http.Request) {
	endpoints, err := s.provider.GetRecords()
	if err != nil {
		log.Printf("Failed to get records: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get records: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", mediaTypeFormat)
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(endpoints); err != nil {
		log.Printf("Failed to encode endpoints: %v", err)
	}
}

func (s *Server) applyChanges(w http.ResponseWriter, r *http.Request) {
	var changes webhook.Changes

	if err := json.NewDecoder(r.Body).Decode(&changes); err != nil {
		log.Printf("Failed to decode changes: %v", err)
		http.Error(w, fmt.Sprintf("Failed to decode changes: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.provider.ApplyChanges(&changes); err != nil {
		log.Printf("Failed to apply changes: %v", err)
		http.Error(w, fmt.Sprintf("Failed to apply changes: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) adjustEndpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var endpoints []*webhook.Endpoint

	if err := json.NewDecoder(r.Body).Decode(&endpoints); err != nil {
		log.Printf("Failed to decode endpoints: %v", err)
		http.Error(w, fmt.Sprintf("Failed to decode endpoints: %v", err), http.StatusBadRequest)
		return
	}

	adjusted, err := s.provider.AdjustEndpoints(endpoints)
	if err != nil {
		log.Printf("Failed to adjust endpoints: %v", err)
		http.Error(w, fmt.Sprintf("Failed to adjust endpoints: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", mediaTypeFormat)
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(adjusted); err != nil {
		log.Printf("Failed to encode adjusted endpoints: %v", err)
	}
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
