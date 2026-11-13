package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

// DockerQuery contains parameters for Docker integration.
type DockerQuery struct {
	SocketPath string `json:"socketPath"` // e.g. /var/run/docker.sock or http://127.0.0.1:2375
}

// ContainerSummary contains basic info for a running or stopped container.
type ContainerSummary struct {
	ID      string `json:"id"`
	Names   string `json:"name"`
	Image   string `json:"image"`
	State   string `json:"state"`  // running, exited, paused
	Status  string `json:"status"` // "Up 4 hours", "Exited (0) 2 days ago"
	Created int64  `json:"created"`
}

// DockerResult contains aggregate Docker engine status.
type DockerResult struct {
	Online     bool               `json:"online"`
	Containers []ContainerSummary `json:"containers"`
	Total      int                `json:"total"`
	Running    int                `json:"running"`
	Paused     int                `json:"paused"`
	Stopped    int                `json:"stopped"`
	Error      string             `json:"error,omitempty"`
}

// QueryDocker queries the Docker socket or TCP endpoint for container information.
func QueryDocker(ctx context.Context, q DockerQuery) DockerResult {
	socket := q.SocketPath
	if socket == "" {
		if runtime.GOOS == "windows" {
			socket = "//./pipe/docker_engine"
		} else {
			socket = "/var/run/docker.sock"
		}
	}

	var client *http.Client

	if strings.HasPrefix(socket, "http://") || strings.HasPrefix(socket, "https://") {
		client = &http.Client{Timeout: 5 * time.Second}
	} else if runtime.GOOS != "windows" && strings.HasPrefix(socket, "/") {
		if _, err := os.Stat(socket); os.IsNotExist(err) {
			return DockerResult{
				Online: false,
				Error:  fmt.Sprintf("Docker socket %s does not exist", socket),
			}
		}
		client = &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", socket)
				},
			},
		}
	} else {
		// Attempt local TCP fallback if pipe or unknown
		client = &http.Client{Timeout: 3 * time.Second}
	}

	apiURL := "http://localhost/containers/json?all=1"
	if strings.HasPrefix(socket, "http://") || strings.HasPrefix(socket, "https://") {
		apiURL = strings.TrimRight(socket, "/") + "/containers/json?all=1"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return DockerResult{Online: false, Error: err.Error()}
	}

	resp, err := client.Do(req)
	if err != nil {
		return DockerResult{Online: false, Error: fmt.Sprintf("Cannot connect to Docker: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return DockerResult{Online: false, Error: fmt.Sprintf("Docker API returned %d", resp.StatusCode)}
	}

	var rawList []struct {
		ID      string   `json:"Id"`
		Names   []string `json:"Names"`
		Image   string   `json:"Image"`
		State   string   `json:"State"`
		Status  string   `json:"Status"`
		Created int64    `json:"Created"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawList); err != nil {
		return DockerResult{Online: false, Error: err.Error()}
	}

	res := DockerResult{
		Online:     true,
		Total:      len(rawList),
		Containers: make([]ContainerSummary, 0, len(rawList)),
	}

	for _, c := range rawList {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		shortID := c.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		switch strings.ToLower(c.State) {
		case "running":
			res.Running++
		case "paused":
			res.Paused++
		default:
			res.Stopped++
		}

		res.Containers = append(res.Containers, ContainerSummary{
			ID:      shortID,
			Names:   name,
			Image:   c.Image,
			State:   c.State,
			Status:  c.Status,
			Created: c.Created,
		})
	}

	return res
}
