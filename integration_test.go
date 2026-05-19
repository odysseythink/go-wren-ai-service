package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/odysseythink/go-wren-ai-service/internal/handler"
	"github.com/odysseythink/go-wren-ai-service/internal/service"
)

func TestHealthEndpoint(t *testing.T) {
	// Create a minimal container for testing
	container := &service.Container{}
	router := handler.NewRouter(container)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != `{"status":"ok"}` {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}
