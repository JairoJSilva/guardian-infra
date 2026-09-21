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
	"sync"
	"time"
)

type DockerClient struct {
	endpoint   string
	baseURL    string
	isUnix     bool
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

func NewDockerClient(endpoint string) *DockerClient {
	if endpoint == "" || endpoint == "docker-local" {
		endpoint = "/var/run/docker.sock"
	}

	var httpClient *http.Client
	var baseURL string
	var socketPath string
	isUnix := false

	clean := strings.TrimSpace(endpoint)
	if strings.HasPrefix(clean, "tcp://") || strings.HasPrefix(clean, "http://") || strings.HasPrefix(clean, "https://") || (!strings.HasPrefix(clean, "/") && strings.Contains(clean, ":")) {
		// Endpoint remoto TCP / HTTP
		rawURL := clean
		if strings.HasPrefix(rawURL, "tcp://") {
			rawURL = "http://" + strings.TrimPrefix(rawURL, "tcp://")
		} else if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
			rawURL = "http://" + rawURL
		}

		u, err := url.Parse(rawURL)
		if err != nil || u.Host == "" {
			baseURL = rawURL
		} else {
			baseURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		}

		httpClient = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   4 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:        50,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression: true,
			},
		}
	} else {
		// Unix Domain Socket local
		isUnix = true
		socketPath = strings.TrimPrefix(clean, "unix://")
		baseURL = "http://unix"
		tr := &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socketPath)
			},
		}
		httpClient = &http.Client{
			Transport: tr,
			Timeout:   10 * time.Second,
		}
	}

	return &DockerClient{
		endpoint:   endpoint,
		baseURL:    baseURL,
		isUnix:     isUnix,
		socketPath: socketPath,
		httpClient: httpClient,
	}
}

func (c *DockerClient) Endpoint() string {
	return c.endpoint
}

func (c *DockerClient) IsUnix() bool {
	return c.isUnix
}

func (c *DockerClient) Ping(ctx context.Context) bool {
	if c.isUnix {
		if _, err := os.Stat(c.socketPath); err != nil {
			return false
		}
	}
	reqURL := fmt.Sprintf("%s/_ping", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
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
	endpoint := fmt.Sprintf("%s/containers/json", c.baseURL)
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
	endpoint := fmt.Sprintf("%s/containers/%s/json", c.baseURL, url.PathEscape(idOrName))
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
	endpoint := fmt.Sprintf("%s/containers/%s/logs?stdout=1&stderr=1&tail=%d", c.baseURL, url.PathEscape(idOrName), tailLines)
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

// DockerPool gerencia instâncias de DockerClient para múltiplos endpoints (Unix sockets ou TCP remotos)
type DockerPool struct {
	mu            sync.RWMutex
	defaultSocket string
	clients       map[string]*DockerClient
}

func NewDockerPool(defaultSocket string) *DockerPool {
	if defaultSocket == "" {
		defaultSocket = "/var/run/docker.sock"
	}
	return &DockerPool{
		defaultSocket: defaultSocket,
		clients:       make(map[string]*DockerClient),
	}
}

func (p *DockerPool) DefaultSocket() string {
	return p.defaultSocket
}

func (p *DockerPool) NormalizeEndpoint(endpoint string) string {
	clean := strings.TrimSpace(endpoint)
	if clean == "" || clean == "docker-local" {
		return p.defaultSocket
	}
	return clean
}

func (p *DockerPool) GetClient(endpoint string) *DockerClient {
	normalized := p.NormalizeEndpoint(endpoint)

	p.mu.RLock()
	client, exists := p.clients[normalized]
	p.mu.RUnlock()
	if exists {
		return client
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Checagem dupla com lock exclusivo
	if client, exists := p.clients[normalized]; exists {
		return client
	}

	client = NewDockerClient(normalized)
	p.clients[normalized] = client
	return client
}

