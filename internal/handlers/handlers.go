package handlers

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"Homebase/internal/config"
	"Homebase/internal/git"
	"Homebase/internal/integrations"
)

// Server provides HTTP routing and API handling for Homebase.
type Server struct {
	cfgMgr    *config.Manager
	gitSyncer *git.Syncer
	webFS     fs.FS
	mux       *http.ServeMux
}

// NewServer constructs the HTTP server with all routes.
func NewServer(cfgMgr *config.Manager, webFS fs.FS) *Server {
	s := &Server{
		cfgMgr:    cfgMgr,
		gitSyncer: git.NewSyncer(cfgMgr),
		webFS:     webFS,
		mux:       http.NewServeMux(),
	}

	s.registerRoutes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for development convenience and API consumers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	s.mux.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	// API routes
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/config", s.handleConfig)
	s.mux.HandleFunc("/api/config/reset", s.handleConfigReset)
	s.mux.HandleFunc("/api/git/sync", s.handleGitSync)
	s.mux.HandleFunc("/api/proxy/proxmox", s.handleProxyProxmox)
	s.mux.HandleFunc("/api/proxy/truenas", s.handleProxyTrueNAS)
	s.mux.HandleFunc("/api/proxy/ping", s.handleProxyPing)
	s.mux.HandleFunc("/api/system/stats", s.handleSystemStats)
	s.mux.HandleFunc("/api/proxy/docker", s.handleProxyDocker)
	s.mux.HandleFunc("/api/weather", s.handleWeather)

	// Static web assets
	if s.webFS != nil {
		fileServer := http.FileServer(http.FS(s.webFS))
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Don't intercept API routes
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}

			// Check if file exists in webFS
			cleanPath := strings.TrimPrefix(r.URL.Path, "/")
			if cleanPath == "" {
				cleanPath = "index.html"
			}

			if f, err := s.webFS.Open(cleanPath); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}

			// SPA Fallback: serve index.html for unknown non-API routes
			if idx, err := s.webFS.Open("index.html"); err == nil {
				_ = idx.Close()
				r.URL.Path = "/"
				fileServer.ServeHTTP(w, r)
				return
			}

			http.NotFound(w, r)
		})
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "app": "Homebase"})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := s.cfgMgr.Get()
		respondJSON(w, http.StatusOK, cfg)

	case http.MethodPost:
		var updated config.AppConfig
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}

		if err := s.cfgMgr.Save(updated); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to save configuration: "+err.Error())
			return
		}

		// If Git auto-sync on save is enabled, trigger push in background
		if updated.Git.Enabled && updated.Git.AutoSyncOnSave {
			go func() {
				s.gitSyncer.Push(r.Context(), "Auto-sync config update from Homebase WYSIWYG editor")
			}()
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"config":  s.cfgMgr.Get(),
		})

	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleConfigReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := s.cfgMgr.Reset(); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to reset config: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"config":  s.cfgMgr.Get(),
	})
}

func (s *Server) handleGitSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Action  string `json:"action"` // "pull" or "push"
		Message string `json:"message"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Action == "" {
		req.Action = "pull"
	}

	var res git.SyncResult
	if strings.ToLower(req.Action) == "push" {
		res = s.gitSyncer.Push(r.Context(), req.Message)
	} else {
		res = s.gitSyncer.Pull(r.Context())
	}

	respondJSON(w, http.StatusOK, res)
}

func (s *Server) handleProxyProxmox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var q integrations.ProxmoxQuery
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid query: "+err.Error())
		return
	}

	res := integrations.QueryProxmox(q)
	respondJSON(w, http.StatusOK, res)
}

func (s *Server) handleProxyTrueNAS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var q integrations.TrueNASQuery
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid query: "+err.Error())
		return
	}

	res := integrations.QueryTrueNAS(q)
	respondJSON(w, http.StatusOK, res)
}

func (s *Server) handleProxyPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var q integrations.PingQuery
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid query: "+err.Error())
		return
	}

	res := integrations.CheckServicePing(r.Context(), q)
	respondJSON(w, http.StatusOK, res)
}

func (s *Server) handleSystemStats(w http.ResponseWriter, r *http.Request) {
	stats := integrations.GetSystemStats()
	respondJSON(w, http.StatusOK, stats)
}

func (s *Server) handleProxyDocker(w http.ResponseWriter, r *http.Request) {
	var q integrations.DockerQuery
	if r.Method == http.MethodPost {
		_ = json.NewDecoder(r.Body).Decode(&q)
	}

	res := integrations.QueryDocker(r.Context(), q)
	respondJSON(w, http.StatusOK, res)
}

func (s *Server) handleWeather(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	lat, _ := strconv.ParseFloat(query.Get("lat"), 64)
	lon, _ := strconv.ParseFloat(query.Get("lon"), 64)
	city := query.Get("city")
	units := query.Get("units")

	q := integrations.WeatherQuery{
		City:      city,
		Latitude:  lat,
		Longitude: lon,
		Units:     units,
	}

	res := integrations.QueryWeather(r.Context(), q)
	respondJSON(w, http.StatusOK, res)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"error":   message,
		"status":  status,
		"success": false,
	})
}
