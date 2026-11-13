package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"Homebase/internal/config"
)

func TestURLHelpers(t *testing.T) {
	// 1. Raw URL conversion
	gh := convertToRawURL("https://github.com/myuser/homelab-dash.git", "main", "config.json")
	expectedGH := "https://raw.githubusercontent.com/myuser/homelab-dash/main/config.json"
	if gh != expectedGH {
		t.Errorf("expected %s, got %s", expectedGH, gh)
	}

	gl := convertToRawURL("https://gitlab.com/myuser/homelab-dash", "production", "config.json")
	expectedGL := "https://gitlab.com/myuser/homelab-dash/-/raw/production/config.json"
	if gl != expectedGL {
		t.Errorf("expected %s, got %s", expectedGL, gl)
	}

	// 2. Auth URL formatting
	authURL := formatAuthenticatedURL("https://github.com/myuser/homelab-dash.git", "ghp_1234567890")
	if authURL != "https://x-access-token:ghp_1234567890@github.com/myuser/homelab-dash.git" {
		t.Errorf("unexpected auth url: %s", authURL)
	}

	noToken := formatAuthenticatedURL("https://github.com/myuser/homelab-dash.git", "")
	if noToken != "https://github.com/myuser/homelab-dash.git" {
		t.Errorf("unexpected no-token url: %s", noToken)
	}
}

func TestSyncerUnconfigured(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "homebase-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")
	mgr, err := config.NewManager(configPath)
	if err != nil {
		t.Fatalf("failed to create config manager: %v", err)
	}

	syncer := NewSyncer(mgr)
	res := syncer.Pull(context.Background())
	if res.Success {
		t.Errorf("expected pull to fail when unconfigured")
	}

	resPush := syncer.Push(context.Background(), "test")
	if resPush.Success {
		t.Errorf("expected push to fail when unconfigured")
	}
}
