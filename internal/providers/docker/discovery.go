package docker

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"guardian/internal/domain"
	"guardian/internal/storage"
)

type DockerHostInspectResult struct {
	Endpoint        string             `json:"endpoint"`
	Status          string             `json:"status"`
	Connected       bool               `json:"connected"`
	ErrorMessage    string             `json:"error_message,omitempty"`
	ContainersCount int                `json:"containers_count"`
	Scopes          []string           `json:"scopes"`
	Containers      []ContainerSummary `json:"containers"`
}

type DockerDiscovery struct {
	pool    *DockerPool
	storage *storage.Storage
}

func NewDockerDiscovery(pool *DockerPool, store *storage.Storage) *DockerDiscovery {
	return &DockerDiscovery{
		pool:    pool,
		storage: store,
	}
}

func (d *DockerDiscovery) DiscoverEnvironments() []domain.EnvironmentInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var envs []domain.EnvironmentInfo
	seen := make(map[string]bool)

	// 1. Host Docker Local padrão
	localSocket := d.pool.DefaultSocket()
	seen[localSocket] = true
	seen["docker-local"] = true

	localClient := d.pool.GetClient(localSocket)
	localStatus := "READY"
	if !localClient.Ping(ctx) {
		localStatus = "UNREACHABLE"
	}
	localStacks := d.discoverStacksForClient(ctx, localClient)

	envs = append(envs, domain.EnvironmentInfo{
		Name:        "docker-local",
		Type:        domain.EnvDocker,
		Endpoint:    localSocket,
		Status:      localStatus,
		Description: fmt.Sprintf("Docker Engine Local (%s)", localSocket),
		Scopes:      localStacks,
	})

	// 2. Hosts Docker registrados em targets configurados
	if d.storage != nil {
		targets := d.storage.List()
		for _, t := range targets {
			if t.Type != domain.EnvDocker || t.Endpoint == "" {
				continue
			}
			norm := d.pool.NormalizeEndpoint(t.Endpoint)
			if seen[norm] {
				continue
			}
			seen[norm] = true

			client := d.pool.GetClient(norm)
			status := "READY"
			if !client.Ping(ctx) {
				status = "UNREACHABLE"
			}
			stacks := d.discoverStacksForClient(ctx, client)
			if len(stacks) == 0 {
				stacks = t.Scopes
			}

			desc := fmt.Sprintf("Docker Host (%s)", t.Endpoint)
			envs = append(envs, domain.EnvironmentInfo{
				Name:        t.Name,
				Type:        domain.EnvDocker,
				Endpoint:    t.Endpoint,
				Status:      status,
				Description: desc,
				Scopes:      stacks,
			})
		}
	}

	return envs
}

func (d *DockerDiscovery) discoverStacksForClient(ctx context.Context, client *DockerClient) []string {
	containers, err := client.ListContainers(ctx, true)
	if err != nil {
		return []string{}
	}

	set := make(map[string]struct{})
	for _, c := range containers {
		if proj, ok := c.Labels["com.docker.compose.project"]; ok && proj != "" {
			set[proj] = struct{}{}
		}
		for _, name := range c.Names {
			cleanName := strings.TrimPrefix(name, "/")
			set[cleanName] = struct{}{}
		}
	}

	var list []string
	for s := range set {
		list = append(list, s)
	}
	sort.Strings(list)
	return list
}

func (d *DockerDiscovery) InspectEndpoint(ctx context.Context, endpoint string) (*DockerHostInspectResult, error) {
	norm := d.pool.NormalizeEndpoint(endpoint)
	client := d.pool.GetClient(norm)

	pingCtx, cancelPing := context.WithTimeout(ctx, 3*time.Second)
	defer cancelPing()

	if !client.Ping(pingCtx) {
		return &DockerHostInspectResult{
			Endpoint:     norm,
			Status:       "UNREACHABLE",
			Connected:    false,
			ErrorMessage: fmt.Sprintf("Não foi possível conectar ao Docker Host em '%s'. Verifique se o daemon está acessível e se a porta/socket está correta.", norm),
			Scopes:       []string{},
			Containers:   []ContainerSummary{},
		}, nil
	}

	listCtx, cancelList := context.WithTimeout(ctx, 4*time.Second)
	defer cancelList()

	containers, err := client.ListContainers(listCtx, true)
	if err != nil {
		return &DockerHostInspectResult{
			Endpoint:     norm,
			Status:       "ERROR",
			Connected:    true,
			ErrorMessage: fmt.Sprintf("Conectou, mas falhou ao listar containers: %v", err),
			Scopes:       []string{},
			Containers:   []ContainerSummary{},
		}, nil
	}

	set := make(map[string]struct{})
	for _, c := range containers {
		if proj, ok := c.Labels["com.docker.compose.project"]; ok && proj != "" {
			set[proj] = struct{}{}
		}
		for _, name := range c.Names {
			cleanName := strings.TrimPrefix(name, "/")
			set[cleanName] = struct{}{}
		}
	}

	var scopes []string
	for s := range set {
		scopes = append(scopes, s)
	}
	sort.Strings(scopes)

	return &DockerHostInspectResult{
		Endpoint:        norm,
		Status:          "READY",
		Connected:       true,
		ContainersCount: len(containers),
		Scopes:          scopes,
		Containers:      containers,
	}, nil
}

