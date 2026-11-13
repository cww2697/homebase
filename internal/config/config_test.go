package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigManagerLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "homebase-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// 1. Initial creation
	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	cfg := mgr.Get()
	if cfg.Settings.Title != "Homebase" {
		t.Errorf("expected title 'Homebase', got '%s'", cfg.Settings.Title)
	}
	if len(cfg.Tabs) == 0 {
		t.Fatalf("expected default tabs, got 0")
	}

	// 2. Modify and Save
	cfg.Settings.Title = "My Custom Homelab"
	cfg.Settings.Theme = "cyberpunk"
	newTab := Tab{
		ID:   "media",
		Name: "Media",
		Icon: "film",
		Widgets: []Widget{
			{
				ID:      "w-plex",
				Title:   "Plex",
				Type:    "service",
				ColSpan: 1,
				RowSpan: 1,
				Properties: map[string]interface{}{
					"url": "http://plex.local:32400",
				},
			},
		},
	}
	cfg.Tabs = append(cfg.Tabs, newTab)

	if err := mgr.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// 3. Reload from disk with fresh manager instance
	mgrReloaded, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("failed to reload manager: %v", err)
	}

	reloadedCfg := mgrReloaded.Get()
	if reloadedCfg.Settings.Title != "My Custom Homelab" {
		t.Errorf("expected 'My Custom Homelab', got '%s'", reloadedCfg.Settings.Title)
	}
	if reloadedCfg.Settings.Theme != "cyberpunk" {
		t.Errorf("expected 'cyberpunk', got '%s'", reloadedCfg.Settings.Theme)
	}
	if len(reloadedCfg.Tabs) != len(cfg.Tabs) {
		t.Errorf("expected %d tabs, got %d", len(cfg.Tabs), len(reloadedCfg.Tabs))
	}
}

func TestConfigReset(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "homebase-reset-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")
	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	cfg := mgr.Get()
	cfg.Settings.Title = "Changed Title"
	_ = mgr.Save(cfg)

	if err := mgr.Reset(); err != nil {
		t.Fatalf("failed to reset: %v", err)
	}

	resetCfg := mgr.Get()
	if resetCfg.Settings.Title != "Homebase" {
		t.Errorf("expected 'Homebase' after reset, got '%s'", resetCfg.Settings.Title)
	}
}
