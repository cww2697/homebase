package git

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"Homebase/internal/config"
)

// SyncResult captures the output and status of a Git operation.
type SyncResult struct {
	Success   bool      `json:"success"`
	Action    string    `json:"action"` // "pull", "push", "test"
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Syncer handles git operations for the configuration file.
type Syncer struct {
	cfgMgr *config.Manager
}

// NewSyncer creates a new Git Syncer.
func NewSyncer(cfgMgr *config.Manager) *Syncer {
	return &Syncer{
		cfgMgr: cfgMgr,
	}
}

// Pull synchronizes the config from the remote Git repository.
func (s *Syncer) Pull(ctx context.Context) SyncResult {
	cfg := s.cfgMgr.Get()
	if !cfg.Git.Enabled || strings.TrimSpace(cfg.Git.RepoURL) == "" {
		return SyncResult{
			Success:   false,
			Action:    "pull",
			Message:   "Git sync is not enabled or repo URL is empty",
			Timestamp: time.Now().UTC(),
		}
	}

	repoURL := formatAuthenticatedURL(cfg.Git.RepoURL, cfg.Git.Token)
	branch := cfg.Git.Branch
	if branch == "" {
		branch = "main"
	}

	// Try git CLI first
	if _, err := exec.LookPath("git"); err == nil {
		res, err := s.pullViaGitCLI(ctx, repoURL, branch)
		if err == nil {
			s.recordStatus(true, "Pulled successfully via Git CLI", nil)
			return res
		}
		// If CLI failed or failed to clone, attempt direct HTTP pull if GitHub/GitLab raw URL
	}

	// Fallback to HTTP download
	res, err := s.pullViaHTTP(ctx, cfg.Git.RepoURL, branch, cfg.Git.Token)
	if err != nil {
		s.recordStatus(false, "Pull failed", err)
		return SyncResult{
			Success:   false,
			Action:    "pull",
			Message:   fmt.Sprintf("Git pull failed: %v", err),
			Timestamp: time.Now().UTC(),
		}
	}

	s.recordStatus(true, "Pulled successfully via HTTP", nil)
	return res
}

// Push commits and pushes the current config to the remote Git repository.
func (s *Syncer) Push(ctx context.Context, commitMessage string) SyncResult {
	cfg := s.cfgMgr.Get()
	if !cfg.Git.Enabled || strings.TrimSpace(cfg.Git.RepoURL) == "" {
		return SyncResult{
			Success:   false,
			Action:    "push",
			Message:   "Git sync is not enabled or repo URL is empty",
			Timestamp: time.Now().UTC(),
		}
	}

	repoURL := formatAuthenticatedURL(cfg.Git.RepoURL, cfg.Git.Token)
	branch := cfg.Git.Branch
	if branch == "" {
		branch = "main"
	}

	if commitMessage == "" {
		commitMessage = fmt.Sprintf("Homebase config update %s", time.Now().UTC().Format(time.RFC3339))
	}

	// Use Git CLI
	if _, err := exec.LookPath("git"); err == nil {
		res, err := s.pushViaGitCLI(ctx, repoURL, branch, commitMessage)
		if err == nil {
			s.recordStatus(true, "Pushed successfully via Git CLI", nil)
			return res
		}
		s.recordStatus(false, "Push failed", err)
		return SyncResult{
			Success:   false,
			Action:    "push",
			Message:   fmt.Sprintf("Git push failed: %v", err),
			Timestamp: time.Now().UTC(),
		}
	}

	return SyncResult{
		Success:   false,
		Action:    "push",
		Message:   "Git CLI is required for pushing commits",
		Timestamp: time.Now().UTC(),
	}
}

func (s *Syncer) pullViaGitCLI(ctx context.Context, authRepoURL, branch string) (SyncResult, error) {
	configPath := s.cfgMgr.GetPath()
	configDir := filepath.Dir(configPath)
	gitDir := filepath.Join(configDir, ".git")

	// If not a git repo yet, initialize or clone
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		tempCloneDir, err := os.MkdirTemp("", "homebase-git-clone-*")
		if err != nil {
			return SyncResult{}, err
		}
		defer os.RemoveAll(tempCloneDir)

		cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", branch, authRepoURL, tempCloneDir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return SyncResult{}, fmt.Errorf("git clone failed: %s (%w)", string(out), err)
		}

		// Look for config.json in cloned repo
		clonedConfig := filepath.Join(tempCloneDir, "config.json")
		if _, err := os.Stat(clonedConfig); os.IsNotExist(err) {
			// Also check homebase.json
			clonedConfig = filepath.Join(tempCloneDir, "homebase.json")
		}

		data, err := os.ReadFile(clonedConfig)
		if err != nil {
			return SyncResult{}, fmt.Errorf("no config.json found in repository: %w", err)
		}

		var parsed config.AppConfig
		if err := json.Unmarshal(data, &parsed); err != nil {
			return SyncResult{}, fmt.Errorf("invalid json in git config: %w", err)
		}

		// Preserve git config so it doesn't get overridden
		current := s.cfgMgr.Get()
		parsed.Git = current.Git
		if err := s.cfgMgr.Save(parsed); err != nil {
			return SyncResult{}, err
		}

		return SyncResult{
			Success:   true,
			Action:    "pull",
			Message:   "Configuration cloned and applied successfully",
			Timestamp: time.Now().UTC(),
		}, nil
	}

	// Already a git repo in configDir
	pullCmd := exec.CommandContext(ctx, "git", "-C", configDir, "pull", authRepoURL, branch)
	out, err := pullCmd.CombinedOutput()
	if err != nil {
		return SyncResult{}, fmt.Errorf("git pull failed: %s (%w)", string(out), err)
	}

	// Reload config from disk
	data, err := os.ReadFile(configPath)
	if err == nil {
		var parsed config.AppConfig
		if err := json.Unmarshal(data, &parsed); err == nil {
			current := s.cfgMgr.Get()
			parsed.Git = current.Git
			_ = s.cfgMgr.Save(parsed)
		}
	}

	return SyncResult{
		Success:   true,
		Action:    "pull",
		Message:   fmt.Sprintf("Git pull successful: %s", strings.TrimSpace(string(out))),
		Timestamp: time.Now().UTC(),
	}, nil
}

func (s *Syncer) pushViaGitCLI(ctx context.Context, authRepoURL, branch, message string) (SyncResult, error) {
	configPath := s.cfgMgr.GetPath()
	configDir := filepath.Dir(configPath)
	gitDir := filepath.Join(configDir, ".git")

	// If not initialized, git init and add remote
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		initCmd := exec.CommandContext(ctx, "git", "-C", configDir, "init")
		if out, err := initCmd.CombinedOutput(); err != nil {
			return SyncResult{}, fmt.Errorf("git init failed: %s", string(out))
		}

		_ = exec.CommandContext(ctx, "git", "-C", configDir, "config", "user.name", "Homebase Dashboard").Run()
		_ = exec.CommandContext(ctx, "git", "-C", configDir, "config", "user.email", "homebase@local").Run()
		_ = exec.CommandContext(ctx, "git", "-C", configDir, "remote", "add", "origin", authRepoURL).Run()
		_ = exec.CommandContext(ctx, "git", "-C", configDir, "branch", "-M", branch).Run()
	} else {
		// Update remote URL with current credentials
		_ = exec.CommandContext(ctx, "git", "-C", configDir, "remote", "set-url", "origin", authRepoURL).Run()
	}

	addCmd := exec.CommandContext(ctx, "git", "-C", configDir, "add", filepath.Base(configPath))
	if out, err := addCmd.CombinedOutput(); err != nil {
		return SyncResult{}, fmt.Errorf("git add failed: %s", string(out))
	}

	commitCmd := exec.CommandContext(ctx, "git", "-C", configDir, "commit", "-m", message)
	_ = commitCmd.Run() // May have nothing to commit if unchanged

	pushCmd := exec.CommandContext(ctx, "git", "-C", configDir, "push", "-u", "origin", branch)
	out, err := pushCmd.CombinedOutput()
	if err != nil {
		return SyncResult{}, fmt.Errorf("git push failed: %s (%w)", string(out), err)
	}

	return SyncResult{
		Success:   true,
		Action:    "push",
		Message:   fmt.Sprintf("Git push successful: %s", strings.TrimSpace(string(out))),
		Timestamp: time.Now().UTC(),
	}, nil
}

func (s *Syncer) pullViaHTTP(ctx context.Context, repoURL, branch, token string) (SyncResult, error) {
	// Support raw GitHub or GitLab links directly
	rawURL := convertToRawURL(repoURL, branch, "config.json")
	if rawURL == "" {
		return SyncResult{}, fmt.Errorf("unable to construct raw URL for repo %s", repoURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return SyncResult{}, err
	}

	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return SyncResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return SyncResult{}, fmt.Errorf("HTTP status %d: %s", resp.StatusCode, string(body))
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return SyncResult{}, err
	}

	var parsed config.AppConfig
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		return SyncResult{}, fmt.Errorf("downloaded file is not valid JSON config: %w", err)
	}

	current := s.cfgMgr.Get()
	parsed.Git = current.Git
	if err := s.cfgMgr.Save(parsed); err != nil {
		return SyncResult{}, err
	}

	return SyncResult{
		Success:   true,
		Action:    "pull",
		Message:   fmt.Sprintf("Successfully fetched config via HTTP from %s", rawURL),
		Timestamp: time.Now().UTC(),
	}, nil
}

func (s *Syncer) recordStatus(success bool, msg string, err error) {
	cfg := s.cfgMgr.Get()
	cfg.Git.LastSyncTime = time.Now().UTC()
	if success {
		cfg.Git.LastSyncStatus = "success"
		cfg.Git.LastError = ""
	} else {
		cfg.Git.LastSyncStatus = "error"
		if err != nil {
			cfg.Git.LastError = err.Error()
		} else {
			cfg.Git.LastError = msg
		}
	}
	_ = s.cfgMgr.Save(cfg)
}

func formatAuthenticatedURL(rawURL, token string) string {
	if token == "" {
		return rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	if u.Scheme == "https" || u.Scheme == "http" {
		u.User = url.UserPassword("x-access-token", token)
		return u.String()
	}

	return rawURL
}

func convertToRawURL(repoURL, branch, filename string) string {
	trimmed := strings.TrimSuffix(repoURL, ".git")
	trimmed = strings.TrimSuffix(trimmed, "/")

	// GitHub: https://github.com/owner/repo -> https://raw.githubusercontent.com/owner/repo/branch/config.json
	if strings.Contains(trimmed, "github.com/") {
		parts := strings.Split(trimmed, "github.com/")
		if len(parts) == 2 {
			return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", parts[1], branch, filename)
		}
	}

	// GitLab: https://gitlab.com/owner/repo -> https://gitlab.com/owner/repo/-/raw/branch/config.json
	if strings.Contains(trimmed, "gitlab.com/") {
		return fmt.Sprintf("%s/-/raw/%s/%s", trimmed, branch, filename)
	}

	return ""
}
