package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"Homebase/internal/config"
)

func setupTestServer(t *testing.T) (*Server, string) {
	tempDir, err := os.MkdirTemp("", "homebase-handler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	configPath := filepath.Join(tempDir, "config.json")
	mgr, err := config.NewManager(configPath)
	if err != nil {
		t.Fatalf("failed to init config manager: %v", err)
	}

	srv := NewServer(mgr, nil)
	return srv, tempDir
}

func TestHealthEndpoint(t *testing.T) {
	srv, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var res map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if res["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", res["status"])
	}
}

func TestConfigGetAndPost(t *testing.T) {
	srv, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	// 1. GET
	reqGet := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	wGet := httptest.NewRecorder()
	srv.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wGet.Code)
	}

	var cfg config.AppConfig
	if err := json.Unmarshal(wGet.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}
	if cfg.Settings.Title != "Homebase" {
		t.Errorf("expected 'Homebase', got '%s'", cfg.Settings.Title)
	}

	// 2. POST update
	cfg.Settings.Title = "Updated Homebase"
	bodyBytes, _ := json.Marshal(cfg)
	reqPost := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(bodyBytes))
	wPost := httptest.NewRecorder()
	srv.ServeHTTP(wPost, reqPost)

	if wPost.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wPost.Code)
	}

	// 3. Verify persistence
	reqGetAgain := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	wGetAgain := httptest.NewRecorder()
	srv.ServeHTTP(wGetAgain, reqGetAgain)

	var cfgAgain config.AppConfig
	_ = json.Unmarshal(wGetAgain.Body.Bytes(), &cfgAgain)
	if cfgAgain.Settings.Title != "Updated Homebase" {
		t.Errorf("expected 'Updated Homebase', got '%s'", cfgAgain.Settings.Title)
	}
}
