package integrations

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TrueNASQuery contains parameters for querying TrueNAS.
type TrueNASQuery struct {
	URL                string `json:"url"`
	APIKey             string `json:"apiKey"`
	Pool               string `json:"pool"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
}

// TrueNASResult contains pool and alert statistics.
type TrueNASResult struct {
	Online     bool    `json:"online"`
	Hostname   string  `json:"hostname"`
	Version    string  `json:"version"`
	PoolName   string  `json:"poolName"`
	PoolStatus string  `json:"poolStatus"` // ONLINE, DEGRADED, FAULTED
	PoolHealth string  `json:"poolHealth"`
	TotalBytes uint64  `json:"totalBytes"`
	UsedBytes  uint64  `json:"usedBytes"`
	FreeBytes  uint64  `json:"freeBytes"`
	UsedPct    float64 `json:"usedPct"`
	AlertCount int     `json:"alertCount"`
	Uptime     string  `json:"uptime"`
	Status     string  `json:"status"`
	Error      string  `json:"error,omitempty"`
}

// QueryTrueNAS queries TrueNAS REST API v2.0 for pool and system metrics.
func QueryTrueNAS(query TrueNASQuery) TrueNASResult {
	baseURL := strings.TrimRight(strings.TrimSpace(query.URL), "/")
	if baseURL == "" {
		return TrueNASResult{
			Online: false,
			Status: "unconfigured",
			Error:  "TrueNAS URL is empty",
		}
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: query.InsecureSkipVerify},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	authHeader := query.APIKey
	if authHeader != "" && !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		authHeader = "Bearer " + authHeader
	}

	res := TrueNASResult{
		Status: "unknown",
	}

	// 1. Query System Info (/api/v2.0/system/info)
	sysReq, err := http.NewRequest(http.MethodGet, baseURL+"/api/v2.0/system/info", nil)
	if err == nil {
		if authHeader != "" {
			sysReq.Header.Set("Authorization", authHeader)
		}
		if sysResp, err := client.Do(sysReq); err == nil {
			defer sysResp.Body.Close()
			if sysResp.StatusCode == http.StatusOK {
				var info struct {
					Hostname string  `json:"hostname"`
					Version  string  `json:"version"`
					Uptime   float64 `json:"uptime_seconds"`
				}
				if json.NewDecoder(sysResp.Body).Decode(&info) == nil {
					res.Online = true
					res.Hostname = info.Hostname
					res.Version = info.Version
					res.Uptime = fmt.Sprintf("%.0f hours", info.Uptime/3600)
				}
			} else {
				body, _ := io.ReadAll(io.LimitReader(sysResp.Body, 512))
				res.Error = fmt.Sprintf("System info returned %d: %s", sysResp.StatusCode, string(body))
			}
		} else {
			res.Error = fmt.Sprintf("Failed to reach TrueNAS: %v", err)
			return res
		}
	}

	// 2. Query Storage Pools (/api/v2.0/pool)
	poolReq, err := http.NewRequest(http.MethodGet, baseURL+"/api/v2.0/pool", nil)
	if err == nil {
		if authHeader != "" {
			poolReq.Header.Set("Authorization", authHeader)
		}
		if poolResp, err := client.Do(poolReq); err == nil {
			defer poolResp.Body.Close()
			if poolResp.StatusCode == http.StatusOK {
				var pools []struct {
					Name   string `json:"name"`
					Status string `json:"status"`
					Path   string `json:"path"`
					Scan   struct {
						State string `json:"state"`
					} `json:"scan"`
					Topology struct {
						Data []struct {
							Stats struct {
								Allocated uint64 `json:"allocated"`
								Size      uint64 `json:"size"`
							} `json:"stats"`
						} `json:"data"`
					} `json:"topology"`
				}
				if json.NewDecoder(poolResp.Body).Decode(&pools) == nil && len(pools) > 0 {
					targetPool := pools[0]
					if query.Pool != "" {
						for _, p := range pools {
							if strings.EqualFold(p.Name, query.Pool) {
								targetPool = p
								break
							}
						}
					}

					res.PoolName = targetPool.Name
					res.PoolStatus = targetPool.Status
					res.PoolHealth = targetPool.Status

					var totalAlloc, totalSize uint64
					for _, d := range targetPool.Topology.Data {
						totalAlloc += d.Stats.Allocated
						totalSize += d.Stats.Size
					}
					res.TotalBytes = totalSize
					res.UsedBytes = totalAlloc
					if totalSize > totalAlloc {
						res.FreeBytes = totalSize - totalAlloc
					}
					if totalSize > 0 {
						res.UsedPct = (float64(totalAlloc) / float64(totalSize)) * 100
					}
				}
			}
		}
	}

	// 3. Query Active Alerts (/api/v2.0/alert/list)
	alertReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/v2.0/alert/list", nil)
	if authHeader != "" && alertReq != nil {
		alertReq.Header.Set("Authorization", authHeader)
		if alertResp, err := client.Do(alertReq); err == nil {
			defer alertResp.Body.Close()
			if alertResp.StatusCode == http.StatusOK {
				var alerts []struct {
					Dismissed bool `json:"dismissed"`
				}
				if json.NewDecoder(alertResp.Body).Decode(&alerts) == nil {
					activeCount := 0
					for _, a := range alerts {
						if !a.Dismissed {
							activeCount++
						}
					}
					res.AlertCount = activeCount
				}
			}
		}
	}

	if res.Online {
		res.Status = "online"
	} else if res.Error != "" {
		res.Status = "error"
	}

	return res
}
