package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"Homebase/internal/config"
	"Homebase/internal/git"
	"Homebase/internal/handlers"
	"Homebase/web"
)

func main() {
	var (
		portFlag       = flag.Int("port", 8080, "HTTP server port")
		dataDirFlag    = flag.String("data", "", "Directory for persistent data storage")
		configPathFlag = flag.String("config", "", "Direct path to config.json file")
	)
	flag.Parse()

	// Environment variable overrides
	port := *portFlag
	if envPort := os.Getenv("PORT"); envPort != "" {
		var p int
		if _, err := fmt.Sscanf(envPort, "%d", &p); err == nil && p > 0 {
			port = p
		}
	}

	configPath := *configPathFlag
	if configPath == "" {
		configPath = os.Getenv("HOMEBASE_CONFIG_PATH")
	}

	if configPath == "" {
		dataDir := *dataDirFlag
		if dataDir == "" {
			dataDir = os.Getenv("HOMEBASE_DATA_DIR")
		}
		if dataDir == "" {
			dataDir = "./data"
		}
		configPath = filepath.Join(dataDir, "config.json")
	}

	log.Printf("[Homebase] Initializing configuration at %s...", configPath)
	cfgMgr, err := config.NewManager(configPath)
	if err != nil {
		log.Fatalf("[Homebase] Fatal: Failed to initialize configuration manager: %v", err)
	}

	// Trigger Git auto-sync on startup if enabled
	cfg := cfgMgr.Get()
	if cfg.Git.Enabled && cfg.Git.AutoSyncOnStartup {
		log.Printf("[Homebase] Git auto-sync on startup enabled. Pulling from %s...", cfg.Git.RepoURL)
		go func() {
			syncer := git.NewSyncer(cfgMgr)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			res := syncer.Pull(ctx)
			log.Printf("[Homebase] Startup Git sync: %s (success=%v)", res.Message, res.Success)
		}()
	}

	server := handlers.NewServer(cfgMgr, web.GetFS())

	addr := fmt.Sprintf(":%d", port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      server,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background goroutine
	go func() {
		log.Printf("=====================================================")
		log.Printf("  Homebase Homelab Dashboard & WYSIWYG Editor")
		log.Printf("  Listening on http://0.0.0.0:%d", port)
		log.Printf("  Config storage: %s", configPath)
		log.Printf("=====================================================")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Homebase] HTTP server error: %v", err)
		}
	}()

	// Wait for termination signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[Homebase] Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("[Homebase] Shutdown forced: %v", err)
	}
	log.Println("[Homebase] Server stopped.")
}
