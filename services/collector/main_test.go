package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetricStore_Add(t *testing.T) {
	store := NewMetricStore()
	store.Add(Metric{Name: "cpu_usage", Value: 45.2})

	if store.Count() != 1 {
		t.Errorf("expected count 1, got %d", store.Count())
	}

	metrics := store.GetAll()
	if metrics[0].Name != "cpu_usage" {
		t.Errorf("expected name cpu_usage, got %s", metrics[0].Name)
	}
	if metrics[0].Value != 45.2 {
		t.Errorf("expected value 45.2, got %f", metrics[0].Value)
	}
	if metrics[0].Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestMetricStore_GetAll_Empty(t *testing.T) {
	store := NewMetricStore()
	metrics := store.GetAll()
	if len(metrics) != 0 {
		t.Errorf("expected empty slice, got %d items", len(metrics))
	}
}

func TestHealthEndpoint(t *testing.T) {
	store := NewMetricStore()
	server := NewServer(store, "8081")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
	if resp["service"] != "collector" {
		t.Errorf("expected service collector, got %s", resp["service"])
	}
}

func TestIngestEndpoint_Success(t *testing.T) {
	store := NewMetricStore()
	server := NewServer(store, "8081")

	metrics := []Metric{
		{Name: "cpu_usage", Value: 75.5, Tags: map[string]string{"host": "server1"}},
		{Name: "memory_usage", Value: 60.0},
	}
	body, _ := json.Marshal(metrics)

	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleIngest(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["accepted"].(float64) != 2 {
		t.Errorf("expected 2 accepted, got %v", resp["accepted"])
	}
}

func TestIngestEndpoint_InvalidMethod(t *testing.T) {
	store := NewMetricStore()
	server := NewServer(store, "8081")

	req := httptest.NewRequest(http.MethodGet, "/ingest", nil)
	w := httptest.NewRecorder()
	server.handleIngest(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestIngestEndpoint_InvalidBody(t *testing.T) {
	store := NewMetricStore()
	server := NewServer(store, "8081")

	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()
	server.handleIngest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestIngestEndpoint_EmptyMetrics(t *testing.T) {
	store := NewMetricStore()
	server := NewServer(store, "8081")

	body, _ := json.Marshal([]Metric{})
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleIngest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestIngestEndpoint_MissingName(t *testing.T) {
	store := NewMetricStore()
	server := NewServer(store, "8081")

	metrics := []Metric{{Value: 10.0}}
	body, _ := json.Marshal(metrics)
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleIngest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	store := NewMetricStore()
	store.Add(Metric{Name: "test_metric", Value: 100.0})
	server := NewServer(store, "8081")

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	server.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var metrics []Metric
	json.NewDecoder(w.Body).Decode(&metrics)
	if len(metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(metrics))
	}
}
