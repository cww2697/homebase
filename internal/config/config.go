package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Widget represents a single dashboard widget item.
type Widget struct {
	ID         string                 `json:"id"`
	Title      string                 `json:"title"`
	Type       string                 `json:"type"`       // service, proxmox, truenas, system, docker, weather, clock, search, iframe, custom_api, notes
	ColSpan    int                    `json:"colSpan"`    // 1 to 4 or 6
	RowSpan    int                    `json:"rowSpan"`    // 1 to 4
	Order      int                    `json:"order"`      // visual ordering
	Properties map[string]interface{} `json:"properties"` // widget-specific attributes
}

// Tab represents a dashboard page/tab.
type Tab struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Icon    string   `json:"icon"` // icon name (e.g. home, server, hard-drive, cpu)
	Widgets []Widget `json:"widgets"`
}

// GitConfig holds options for syncing configuration with a Git repository.
type GitConfig struct {
	Enabled           bool      `json:"enabled"`
	RepoURL           string    `json:"repoUrl"`
	Branch            string    `json:"branch"`
	Token             string    `json:"token"` // Personal Access Token
	AutoSyncOnStartup bool      `json:"autoSyncOnStartup"`
	AutoSyncOnSave    bool      `json:"autoSyncOnSave"`
	LastSyncStatus    string    `json:"lastSyncStatus"`
	LastSyncTime      time.Time `json:"lastSyncTime,omitempty"`
	LastError         string    `json:"lastError,omitempty"`
}

// Settings represents global dashboard preferences.
type Settings struct {
	Title           string `json:"title"`
	Theme           string `json:"theme"` // dark, cyberpunk, slate, nord, monokai, light
	BackgroundURL   string `json:"backgroundUrl"`
	BackgroundBlur  bool   `json:"backgroundBlur"`
	Columns         int    `json:"columns"`         // Default grid columns (e.g. 4, 6)
	RefreshInterval int    `json:"refreshInterval"` // In seconds for metric widgets
}

// AppConfig is the root configuration object for Homebase.
type AppConfig struct {
	Version   string    `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
	Settings  Settings  `json:"settings"`
	Git       GitConfig `json:"git"`
	Tabs      []Tab     `json:"tabs"`
}

// Manager handles reading and writing the application configuration.
type Manager struct {
	mu         sync.RWMutex
	configPath string
	config     *AppConfig
}

// NewManager creates a new config manager pointing to the specified file path.
func NewManager(path string) (*Manager, error) {
	if path == "" {
		dataDir := os.Getenv("HOMEBASE_DATA_DIR")
		if dataDir == "" {
			dataDir = "./data"
		}
		path = filepath.Join(dataDir, "config.json")
	}

	m := &Manager{
		configPath: path,
	}

	if err := m.loadOrCreate(); err != nil {
		return nil, err
	}

	return m, nil
}

// Get returns a clone of the current configuration.
func (m *Manager) Get() AppConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return copy
	var cloned AppConfig
	b, _ := json.Marshal(m.config)
	_ = json.Unmarshal(b, &cloned)
	return cloned
}

// GetPath returns the path to the config file.
func (m *Manager) GetPath() string {
	return m.configPath
}

// Save atomically saves the configuration to disk.
func (m *Manager) Save(cfg AppConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg.UpdatedAt = time.Now().UTC()
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}
	if cfg.Settings.Columns <= 0 {
		cfg.Settings.Columns = 4
	}
	if cfg.Settings.RefreshInterval <= 0 {
		cfg.Settings.RefreshInterval = 30
	}

	if err := m.writeAtomic(&cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	m.config = &cfg
	return nil
}

// Reset resets the configuration to default.
func (m *Manager) Reset() error {
	def := DefaultConfig()
	return m.Save(def)
}

func (m *Manager) loadOrCreate() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		def := DefaultConfig()
		if err := m.writeAtomic(&def); err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}
		m.config = &def
		return nil
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config json: %w", err)
	}

	// Validate baseline
	if len(cfg.Tabs) == 0 {
		def := DefaultConfig()
		cfg.Tabs = def.Tabs
	}

	m.config = &cfg
	return nil
}

func (m *Manager) writeAtomic(cfg *AppConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := fmt.Sprintf("%s.tmp-%d", m.configPath, time.Now().UnixNano())
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}

	if err := os.Rename(tmpFile, m.configPath); err != nil {
		// Fallback for filesystems where atomic rename across drives fails
		if errWrite := os.WriteFile(m.configPath, data, 0644); errWrite != nil {
			_ = os.Remove(tmpFile)
			return errWrite
		}
		_ = os.Remove(tmpFile)
	}

	return nil
}

// DefaultConfig generates a rich, ready-to-use homelab homepage configuration.
func DefaultConfig() AppConfig {
	return AppConfig{
		Version:   "1.0.0",
		UpdatedAt: time.Now().UTC(),
		Settings: Settings{
			Title:           "Homebase",
			Theme:           "dark",
			BackgroundURL:   "",
			BackgroundBlur:  true,
			Columns:         4,
			RefreshInterval: 30,
		},
		Git: GitConfig{
			Enabled:           false,
			RepoURL:           "",
			Branch:            "main",
			Token:             "",
			AutoSyncOnStartup: false,
			AutoSyncOnSave:    false,
			LastSyncStatus:    "unconfigured",
		},
		Tabs: []Tab{
			{
				ID:   "home",
				Name: "Home",
				Icon: "layout-dashboard",
				Widgets: []Widget{
					{
						ID:      "w-clock",
						Title:   "Clock & Date",
						Type:    "clock",
						ColSpan: 2,
						RowSpan: 1,
						Order:   1,
						Properties: map[string]interface{}{
							"format24":   true,
							"showDate":   true,
							"showSecond": true,
						},
					},
					{
						ID:      "w-search",
						Title:   "Web Search",
						Type:    "search",
						ColSpan: 2,
						RowSpan: 1,
						Order:   2,
						Properties: map[string]interface{}{
							"provider": "duckduckgo",
						},
					},
					{
						ID:      "w-system",
						Title:   "Host Resources",
						Type:    "system",
						ColSpan: 2,
						RowSpan: 1,
						Order:   3,
						Properties: map[string]interface{}{
							"showDisk":   true,
							"showMemory": true,
							"showCpu":    true,
						},
					},
					{
						ID:      "w-weather",
						Title:   "Local Weather",
						Type:    "weather",
						ColSpan: 2,
						RowSpan: 1,
						Order:   4,
						Properties: map[string]interface{}{
							"city":      "New York",
							"latitude":  40.7128,
							"longitude": -74.0060,
							"units":     "celsius",
						},
					},
					{
						ID:      "w-proxmox-preview",
						Title:   "Proxmox VE Cluster",
						Type:    "proxmox",
						ColSpan: 2,
						RowSpan: 1,
						Order:   5,
						Properties: map[string]interface{}{
							"url":                "https://pve.local:8006",
							"apiToken":           "root@pam!homebase=00000000-0000-0000-0000-000000000000",
							"node":               "pve",
							"insecureSkipVerify": true,
						},
					},
					{
						ID:      "w-truenas-preview",
						Title:   "TrueNAS Storage",
						Type:    "truenas",
						ColSpan: 2,
						RowSpan: 1,
						Order:   6,
						Properties: map[string]interface{}{
							"url":                "https://truenas.local",
							"apiKey":             "",
							"pool":               "tank",
							"insecureSkipVerify": true,
						},
					},
				},
			},
			{
				ID:   "infrastructure",
				Name: "Infrastructure",
				Icon: "server",
				Widgets: []Widget{
					{
						ID:      "w-pve-full",
						Title:   "Proxmox Node Status",
						Type:    "proxmox",
						ColSpan: 4,
						RowSpan: 1,
						Order:   1,
						Properties: map[string]interface{}{
							"url":                "https://pve.local:8006",
							"apiToken":           "",
							"node":               "pve",
							"insecureSkipVerify": true,
						},
					},
					{
						ID:      "w-truenas-full",
						Title:   "TrueNAS SCALE Health",
						Type:    "truenas",
						ColSpan: 4,
						RowSpan: 1,
						Order:   2,
						Properties: map[string]interface{}{
							"url":                "https://truenas.local",
							"apiKey":             "",
							"pool":               "tank",
							"insecureSkipVerify": true,
						},
					},
					{
						ID:      "w-docker-host",
						Title:   "Docker Containers",
						Type:    "docker",
						ColSpan: 4,
						RowSpan: 1,
						Order:   3,
						Properties: map[string]interface{}{
							"socket": "/var/run/docker.sock",
						},
					},
				},
			},
			{
				ID:   "services",
				Name: "Services",
				Icon: "grid",
				Widgets: []Widget{
					{
						ID:      "s-pve-link",
						Title:   "Proxmox VE",
						Type:    "service",
						ColSpan: 1,
						RowSpan: 1,
						Order:   1,
						Properties: map[string]interface{}{
							"url":         "https://pve.local:8006",
							"description": "Hypervisor management console",
							"icon":        "server",
							"pingUrl":     "https://pve.local:8006",
							"tag":         "Hypervisor",
						},
					},
					{
						ID:      "s-truenas-link",
						Title:   "TrueNAS",
						Type:    "service",
						ColSpan: 1,
						RowSpan: 1,
						Order:   2,
						Properties: map[string]interface{}{
							"url":         "https://truenas.local",
							"description": "Network attached storage",
							"icon":        "hard-drive",
							"pingUrl":     "https://truenas.local",
							"tag":         "Storage",
						},
					},
					{
						ID:      "s-pihole-link",
						Title:   "Pi-hole",
						Type:    "service",
						ColSpan: 1,
						RowSpan: 1,
						Order:   3,
						Properties: map[string]interface{}{
							"url":         "http://pi.hole/admin",
							"description": "Network-wide ad blocking",
							"icon":        "shield",
							"pingUrl":     "http://pi.hole",
							"tag":         "Network",
						},
					},
					{
						ID:      "s-portainer-link",
						Title:   "Portainer",
						Type:    "service",
						ColSpan: 1,
						RowSpan: 1,
						Order:   4,
						Properties: map[string]interface{}{
							"url":         "https://portainer.local:9443",
							"description": "Container management platform",
							"icon":        "box",
							"pingUrl":     "https://portainer.local:9443",
							"tag":         "Containers",
						},
					},
					{
						ID:      "s-plex-link",
						Title:   "Plex Media Server",
						Type:    "service",
						ColSpan: 1,
						RowSpan: 1,
						Order:   5,
						Properties: map[string]interface{}{
							"url":         "http://plex.local:32400/web",
							"description": "Personal media library",
							"icon":        "film",
							"pingUrl":     "http://plex.local:32400",
							"tag":         "Media",
						},
					},
					{
						ID:      "s-homeassistant-link",
						Title:   "Home Assistant",
						Type:    "service",
						ColSpan: 1,
						RowSpan: 1,
						Order:   6,
						Properties: map[string]interface{}{
							"url":         "http://homeassistant.local:8123",
							"description": "Open source home automation",
							"icon":        "home",
							"pingUrl":     "http://homeassistant.local:8123",
							"tag":         "Smart Home",
						},
					},
					{
						ID:      "s-grafana-link",
						Title:   "Grafana",
						Type:    "service",
						ColSpan: 1,
						RowSpan: 1,
						Order:   7,
						Properties: map[string]interface{}{
							"url":         "http://grafana.local:3000",
							"description": "Operational dashboards & alerts",
							"icon":        "activity",
							"pingUrl":     "http://grafana.local:3000",
							"tag":         "Monitoring",
						},
					},
					{
						ID:      "s-uptime-link",
						Title:   "Uptime Kuma",
						Type:    "service",
						ColSpan: 1,
						RowSpan: 1,
						Order:   8,
						Properties: map[string]interface{}{
							"url":         "http://kuma.local:3001",
							"description": "Self-hosted monitoring tool",
							"icon":        "check-circle",
							"pingUrl":     "http://kuma.local:3001",
							"tag":         "Monitoring",
						},
					},
				},
			},
			{
				ID:   "notes",
				Name: "Notes",
				Icon: "file-text",
				Widgets: []Widget{
					{
						ID:      "w-homelab-notes",
						Title:   "Homelab Runbook",
						Type:    "notes",
						ColSpan: 4,
						RowSpan: 2,
						Order:   1,
						Properties: map[string]interface{}{
							"content": "### Welcome to Homebase!\n\nThis dashboard allows you to organize all your services, monitor Proxmox nodes, TrueNAS storage pools, and host resources with a live WYSIWYG editor.\n\n- Click **Edit Dashboard** at the top right to rearrange widgets, add new widgets, or create new tabs.\n- Configure your **Proxmox** and **TrueNAS** API tokens to see real-time metrics.\n- Connect a **Git repository** in Settings to automatically back up your dashboard configuration to GitHub.",
						},
					},
				},
			},
		},
	}
}
