package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSystemStats(t *testing.T) {
	stats := GetSystemStats()
	if stats.NumCPU <= 0 {
		t.Errorf("expected NumCPU > 0, got %d", stats.NumCPU)
	}
	if stats.OS == "" {
		t.Errorf("expected OS to be non-empty")
	}
}

func TestCheckServicePing(t *testing.T) {
	// 1. Success mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	res := CheckServicePing(context.Background(), PingQuery{
		URL:     server.URL,
		Timeout: 2000,
	})

	if !res.Online {
		t.Errorf("expected server to be online, got error: %s", res.Error)
	}
	if res.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	// 2. Unreachable URL
	resFail := CheckServicePing(context.Background(), PingQuery{
		URL:     "http://127.0.0.1:54321/non-existent",
		Timeout: 200,
	})
	if resFail.Online {
		t.Errorf("expected unreachable URL to be offline")
	}
}

func TestProxmoxMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api2/json/nodes/pve/status" {
			auth := r.Header.Get("Authorization")
			if auth != "PVEAPIToken=root@pam!token=1234" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"cpu": 0.25,
					"memory": map[string]interface{}{
						"used":  4 * 1024 * 1024 * 1024,
						"total": 16 * 1024 * 1024 * 1024,
					},
					"rootfs": map[string]interface{}{
						"used":  20 * 1024 * 1024 * 1024,
						"total": 100 * 1024 * 1024 * 1024,
					},
					"uptime":     36000,
					"pveversion": "pve-manager/8.2.4",
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	res := QueryProxmox(ProxmoxQuery{
		URL:                server.URL,
		APIToken:           "root@pam!token=1234",
		Node:               "pve",
		InsecureSkipVerify: true,
	})

	if !res.Online {
		t.Fatalf("expected Proxmox mock to be online, got: %s", res.Error)
	}
	if res.CPUUsage != 25.0 {
		t.Errorf("expected 25.0%% CPU, got %f", res.CPUUsage)
	}
	if res.PveVersion != "pve-manager/8.2.4" {
		t.Errorf("expected pve-manager/8.2.4, got %s", res.PveVersion)
	}
}

func TestTrueNASMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2.0/system/info":
			resp := map[string]interface{}{
				"hostname":       "truenas-scale",
				"version":        "TrueNAS-SCALE-24.04.0",
				"uptime_seconds": 72000,
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/v2.0/pool":
			resp := []map[string]interface{}{
				{
					"name":   "tank",
					"status": "ONLINE",
					"topology": map[string]interface{}{
						"data": []map[string]interface{}{
							{
								"stats": map[string]interface{}{
									"allocated": 2000000000000,
									"size":      10000000000000,
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/v2.0/alert/list":
			json.NewEncoder(w).Encode([]interface{}{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	res := QueryTrueNAS(TrueNASQuery{
		URL:    server.URL,
		APIKey: "secret-key",
		Pool:   "tank",
	})

	if !res.Online {
		t.Fatalf("expected TrueNAS mock to be online, got error: %s", res.Error)
	}
	if res.PoolName != "tank" {
		t.Errorf("expected pool tank, got %s", res.PoolName)
	}
	if res.PoolStatus != "ONLINE" {
		t.Errorf("expected pool ONLINE, got %s", res.PoolStatus)
	}
}
