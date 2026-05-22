package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/odysseythink/go-wren-ai-service/internal/config"
)

func TestForceDeploy_Success(t *testing.T) {
	var received map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/graphql" {
			t.Errorf("expected path /api/graphql, got %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		if accept := r.Header.Get("Accept"); accept != "application/json" {
			t.Errorf("expected Accept application/json, got %s", accept)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"deploy": "ok"}})
	}))
	defer ts.Close()

	cfg := &config.Config{
		WrenUIEndpoint: ts.URL,
	}
	if err := forceDeploy(cfg); err != nil {
		t.Fatalf("forceDeploy: %v", err)
	}

	query, _ := received["query"].(string)
	if query != "mutation Deploy($force: Boolean) { deploy(force: $force) }" {
		t.Errorf("unexpected query: %s", query)
	}
	vars, _ := received["variables"].(map[string]any)
	if vars["force"] != true {
		t.Errorf("expected force=true, got %v", vars["force"])
	}
}

func TestForceDeploy_RetryThenSuccess(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"deploy": "ok"}})
	}))
	defer ts.Close()

	cfg := &config.Config{WrenUIEndpoint: ts.URL}
	if err := forceDeploy(cfg); err != nil {
		t.Fatalf("forceDeploy: %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestForceDeploy_MaxRetriesExceeded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	cfg := &config.Config{WrenUIEndpoint: ts.URL}
	err := forceDeploy(cfg)
	if err == nil {
		t.Fatal("expected error after max retries")
	}
}

func TestForceDeploy_HTTPStatusError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer ts.Close()

	cfg := &config.Config{WrenUIEndpoint: ts.URL}
	err := forceDeploy(cfg)
	if err == nil {
		t.Fatal("expected error on HTTP 502")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("expected error to mention status 502, got: %v", err)
	}
}

func TestForceDeploy_GraphQLError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{{"message": "deploy failed"}},
		})
	}))
	defer ts.Close()

	cfg := &config.Config{WrenUIEndpoint: ts.URL}
	err := forceDeploy(cfg)
	if err == nil {
		t.Fatal("expected error on GraphQL error response")
	}
	if !strings.Contains(err.Error(), "graphql error") {
		t.Errorf("expected graphql error, got: %v", err)
	}
}

func TestWaitForServer(t *testing.T) {
	// Server that becomes ready after 2 requests.
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		if count < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	// Extract host:port from test server URL.
	addr := ts.Listener.Addr().String()
	if err := waitForServer(addr, 5*time.Second); err != nil {
		t.Fatalf("waitForServer: %v", err)
	}
	if count < 2 {
		t.Errorf("expected at least 2 health checks, got %d", count)
	}
}

func TestWaitForServer_Timeout(t *testing.T) {
	// Server that never becomes ready.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	addr := ts.Listener.Addr().String()
	start := time.Now()
	err := waitForServer(addr, 500*time.Millisecond)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed > 2*time.Second {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

func TestWaitForServer_WrongPath(t *testing.T) {
	// Server without /health endpoint.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	addr := ts.Listener.Addr().String()
	err := waitForServer(addr, 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when health endpoint returns 404")
	}
	if !strings.Contains(err.Error(), "did not become ready") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
