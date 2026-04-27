package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Metric struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Tags      map[string]string `json:"tags,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

type MetricStore struct {
	mu      sync.RWMutex
	metrics []Metric
}

func NewMetricStore() *MetricStore {
	return &MetricStore{metrics: make([]Metric, 0)}
}

func (s *MetricStore) Add(m Metric) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now().UTC()
	}
	s.metrics = append(s.metrics, m)
}

func (s *MetricStore) GetAll() []Metric {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Metric, len(s.metrics))
	copy(result, s.metrics)
	return result
}

func (s *MetricStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.metrics)
}

type Server struct {
	store  *MetricStore
	port   string
	logger *log.Logger
}

func NewServer(store *MetricStore, port string) *Server {
	return &Server{
		store:  store,
		port:   port,
		logger: log.New(os.Stdout, "[collector] ", log.LstdFlags),
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "collector",
	})
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var metrics []Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		s.logger.Printf("Error decoding metrics: %v", err)
		http.Error(w, fmt.Sprintf(`{"error":"invalid request body: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		http.Error(w, `{"error":"empty metrics array"}`, http.StatusBadRequest)
		return
	}

	for _, m := range metrics {
		if m.Name == "" {
			http.Error(w, `{"error":"metric name is required"}`, http.StatusBadRequest)
			return
		}
		s.store.Add(m)
	}

	s.logger.Printf("Ingested %d metrics", len(metrics))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accepted": len(metrics),
		"total":    s.store.Count(),
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.store.GetAll())
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ingest", s.handleIngest)
	mux.HandleFunc("/metrics", s.handleMetrics)

	s.logger.Printf("Starting collector service on port %s", s.port)
	return http.ListenAndServe(":"+s.port, mux)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	port := getEnv("COLLECTOR_PORT", "8081")
	store := NewMetricStore()
	server := NewServer(store, port)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start collector: %v", err)
	}
}
