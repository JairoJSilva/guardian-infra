package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type DockerClient struct {
	socketPath string
	httpClient *http.Client
}

type ContainerSummary struct {
	ID      string            `json:"Id"`
	Names   []string          `json:"Names"`
	Image   string            `json:"Image"`
	State   string            `json:"State"`
	Status  string            `json:"Status"`
	Labels  map[string]string `json:"Labels"`
	Created int64             `json:"Created"`
}

type ContainerInspect struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	State struct {
		Status     string    `json:"Status"`
		Running    bool      `json:"Running"`
		Paused     bool      `json:"Paused"`
		Restarting bool      `json:"Restarting"`
		OOMKilled  bool      `json:"OOMKilled"`
		Dead       bool      `json:"Dead"`
		ExitCode   int       `json:"ExitCode"`
		Error      string    `json:"Error"`
		StartedAt  time.Time `json:"StartedAt"`
		FinishedAt time.Time `json:"FinishedAt"`
	} `json:"State"`
	Config struct {
		Image  string            `json:"Image"`
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	RestartCount int `json:"RestartCount"`
}

func NewDockerClient(socketPath string) *DockerClient {
	if socketPath == "" {
		socketPath = "/var/run/docker.sock"
	}
	tr := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
	}
	return &DockerClient{
		socketPath: socketPath,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   10 * time.Second,
		},
	}
}

func (c *DockerClient) Ping(ctx context.Context) bool {
	if _, err := os.Stat(c.socketPath); err != nil {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/_ping", nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *DockerClient) ListContainers(ctx context.Context, all bool) ([]ContainerSummary, error) {
	endpoint := "http://localhost/containers/json"
	if all {
		endpoint += "?all=1"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docker API retornou HTTP %d: %s", resp.StatusCode, string(body))
	}

	var containers []ContainerSummary
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, err
	}
	return containers, nil
}

func (c *DockerClient) InspectContainer(ctx context.Context, idOrName string) (*ContainerInspect, error) {
	endpoint := fmt.Sprintf("http://localhost/containers/%s/json", url.PathEscape(idOrName))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docker inspect retornou HTTP %d: %s", resp.StatusCode, string(body))
	}

	var inspect ContainerInspect
	if err := json.NewDecoder(resp.Body).Decode(&inspect); err != nil {
		return nil, err
	}
	return &inspect, nil
}

func (c *DockerClient) GetLogs(ctx context.Context, idOrName string, tailLines int) (string, error) {
	if tailLines <= 0 {
		tailLines = 50
	}
	endpoint := fmt.Sprintf("http://localhost/containers/%s/logs?stdout=1&stderr=1&tail=%d", url.PathEscape(idOrName), tailLines)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// O Docker multiplexa stdout/stderr com cabeçalho de 8 bytes por frame
	clean := stripDockerLogHeaders(raw)
	return clean, nil
}

// stripDockerLogHeaders remove os 8 bytes de cabeçalho do stream do Docker se presentes
func stripDockerLogHeaders(input []byte) string {
	if len(input) == 0 {
		return ""
	}
	var sb strings.Builder
	idx := 0
	for idx < len(input) {
		if idx+8 <= len(input) && (input[idx] == 1 || input[idx] == 2) {
			// Frame format: [stream_type(1), 0, 0, 0, size(4 bytes big-endian)]
			size := int(input[idx+4])<<24 | int(input[idx+5])<<16 | int(input[idx+6])<<8 | int(input[idx+7])
			idx += 8
			if idx+size <= len(input) {
				sb.Write(input[idx : idx+size])
				idx += size
				continue
			}
		}
		sb.Write(input[idx:])
		break
	}
	return sb.String()
}
