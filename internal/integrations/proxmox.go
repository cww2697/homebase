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

// ProxmoxQuery contains options for querying Proxmox VE.
type ProxmoxQuery struct {
	URL                string `json:"url"`
	APIToken           string `json:"apiToken"` // format: USER@REALM!TOKENID=UUID
	Node               string `json:"node"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
}

// ProxmoxResult contains normalized node and cluster metrics.
type ProxmoxResult struct {
	Online     bool    `json:"online"`
	Node       string  `json:"node"`
	CPUUsage   float64 `json:"cpuUsage"` // Percentage 0 - 100
	MemUsed    uint64  `json:"memUsed"`  // Bytes
	MemTotal   uint64  `json:"memTotal"` // Bytes
	MemPercent float64 `json:"memPercent"`
	DiskUsed   uint64  `json:"diskUsed"`
	DiskTotal  uint64  `json:"diskTotal"`
	Uptime     uint64  `json:"uptime"`   // Seconds
	VMCount    int     `json:"vmCount"`  // Count of QEMU VMs
	LXCCount   int     `json:"lxcCount"` // Count of LXC containers
	PveVersion string  `json:"pveVersion"`
	Status     string  `json:"status"`
	Error      string  `json:"error,omitempty"`
}

// QueryProxmox connects to the Proxmox VE API and returns node metrics.
func QueryProxmox(query ProxmoxQuery) ProxmoxResult {
	baseURL := strings.TrimRight(strings.TrimSpace(query.URL), "/")
	if baseURL == "" {
		return ProxmoxResult{
			Online: false,
			Status: "unconfigured",
			Error:  "Proxmox URL is empty",
		}
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: query.InsecureSkipVerify},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	authHeader := query.APIToken
	if authHeader != "" && !strings.HasPrefix(strings.ToLower(authHeader), "pveapitoken=") {
		authHeader = "PVEAPIToken=" + authHeader
	}

	node := query.Node
	if node == "" {
		node = "pve"
	}

	// 1. Query Node status
	nodeURL := fmt.Sprintf("%s/api2/json/nodes/%s/status", baseURL, node)
	req, err := http.NewRequest(http.MethodGet, nodeURL, nil)
	if err != nil {
		return ProxmoxResult{Online: false, Error: err.Error()}
	}

	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := client.Do(req)
	if err != nil {
		return ProxmoxResult{
			Online: false,
			Node:   node,
			Status: "unreachable",
			Error:  fmt.Sprintf("Failed to connect to Proxmox: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return ProxmoxResult{
			Online: false,
			Node:   node,
			Status: fmt.Sprintf("HTTP %d", resp.StatusCode),
			Error:  fmt.Sprintf("Proxmox API returned %d: %s", resp.StatusCode, string(body)),
		}
	}

	var statusResp struct {
		Data struct {
			CPU    float64 `json:"cpu"`
			Memory struct {
				Used  uint64 `json:"used"`
				Total uint64 `json:"total"`
			} `json:"memory"`
			RootFS struct {
				Used  uint64 `json:"used"`
				Total uint64 `json:"total"`
			} `json:"rootfs"`
			Uptime     uint64 `json:"uptime"`
			PveVersion string `json:"pveversion"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return ProxmoxResult{
			Online: false,
			Node:   node,
			Status: "parse_error",
			Error:  fmt.Sprintf("Failed to parse Proxmox status: %v", err),
		}
	}

	res := ProxmoxResult{
		Online:     true,
		Node:       node,
		CPUUsage:   statusResp.Data.CPU * 100,
		MemUsed:    statusResp.Data.Memory.Used,
		MemTotal:   statusResp.Data.Memory.Total,
		DiskUsed:   statusResp.Data.RootFS.Used,
		DiskTotal:  statusResp.Data.RootFS.Total,
		Uptime:     statusResp.Data.Uptime,
		PveVersion: statusResp.Data.PveVersion,
		Status:     "online",
	}

	if res.MemTotal > 0 {
		res.MemPercent = (float64(res.MemUsed) / float64(res.MemTotal)) * 100
	}

	// 2. Query Qemu & LXC count (optional, ignore errors if unauthorized for cluster)
	qemuReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api2/json/nodes/%s/qemu", baseURL, node), nil)
	if authHeader != "" && qemuReq != nil {
		qemuReq.Header.Set("Authorization", authHeader)
		if qemuResp, err := client.Do(qemuReq); err == nil {
			defer qemuResp.Body.Close()
			var qemuData struct {
				Data []interface{} `json:"data"`
			}
			if json.NewDecoder(qemuResp.Body).Decode(&qemuData) == nil {
				res.VMCount = len(qemuData.Data)
			}
		}
	}

	lxcReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api2/json/nodes/%s/lxc", baseURL, node), nil)
	if authHeader != "" && lxcReq != nil {
		lxcReq.Header.Set("Authorization", authHeader)
		if lxcResp, err := client.Do(lxcReq); err == nil {
			defer lxcResp.Body.Close()
			var lxcData struct {
				Data []interface{} `json:"data"`
			}
			if json.NewDecoder(lxcResp.Body).Decode(&lxcData) == nil {
				res.LXCCount = len(lxcData.Data)
			}
		}
	}

	return res
}
