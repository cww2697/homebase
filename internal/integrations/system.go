package integrations

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

var startTime = time.Now()

// SystemStats holds host resource utilization and environment information.
type SystemStats struct {
	OS            string    `json:"os"`
	Arch          string    `json:"arch"`
	NumCPU        int       `json:"numCpu"`
	NumGoroutine  int       `json:"numGoroutine"`
	MemoryAlloc   uint64    `json:"memoryAlloc"`   // Bytes currently allocated by runtime
	MemoryTotal   uint64    `json:"memoryTotal"`   // Cumulative bytes allocated
	MemorySys     uint64    `json:"memorySys"`     // Bytes obtained from OS
	UptimeSeconds uint64    `json:"uptimeSeconds"` // Seconds since Homebase started
	ServerTime    time.Time `json:"serverTime"`
}

// GetSystemStats returns the current host/process metrics.
func GetSystemStats() SystemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemStats{
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		NumCPU:        runtime.NumCPU(),
		NumGoroutine:  runtime.NumGoroutine(),
		MemoryAlloc:   m.Alloc,
		MemoryTotal:   m.TotalAlloc,
		MemorySys:     m.Sys,
		UptimeSeconds: uint64(time.Since(startTime).Seconds()),
		ServerTime:    time.Now().UTC(),
	}
}

// PingQuery contains parameters for checking health of an external service.
type PingQuery struct {
	URL     string `json:"url"`
	Timeout int    `json:"timeout"` // Milliseconds, default 3000
}

// PingResult contains service reachability and latency.
type PingResult struct {
	URL        string `json:"url"`
	Online     bool   `json:"online"`
	StatusCode int    `json:"statusCode"`
	LatencyMs  int64  `json:"latencyMs"`
	Error      string `json:"error,omitempty"`
}

// CheckServicePing performs an HTTP HEAD or GET check against a target service URL.
func CheckServicePing(ctx context.Context, query PingQuery) PingResult {
	if query.URL == "" {
		return PingResult{Online: false, Error: "Empty URL"}
	}

	timeout := time.Duration(query.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 3500 * time.Millisecond
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{
		Timeout: timeout,
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, query.URL, nil)
	if err != nil {
		return PingResult{
			URL:    query.URL,
			Online: false,
			Error:  err.Error(),
		}
	}

	req.Header.Set("User-Agent", "Homebase-HealthCheck/1.0")

	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return PingResult{
			URL:        query.URL,
			Online:     false,
			LatencyMs:  latency,
			StatusCode: 0,
			Error:      fmt.Sprintf("Failed: %v", err),
		}
	}
	defer resp.Body.Close()

	online := resp.StatusCode >= 200 && resp.StatusCode < 400

	return PingResult{
		URL:        query.URL,
		Online:     online,
		StatusCode: resp.StatusCode,
		LatencyMs:  latency,
	}
}
