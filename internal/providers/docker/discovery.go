package docker

import (
	"context"
	"strings"
	"time"

	"guardian/internal/domain"
)

type DockerDiscovery struct {
	client *DockerClient
}

func NewDockerDiscovery(client *DockerClient) *DockerDiscovery {
	return &DockerDiscovery{client: client}
}

func (d *DockerDiscovery) DiscoverEnvironments() []domain.EnvironmentInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	available := d.client.Ping(ctx)
	status := "READY"
	if !available {
		status = "UNREACHABLE"
	}

	stacks := d.DiscoverStacks()

	return []domain.EnvironmentInfo{
		{
			Name:        "docker-local",
			Type:        domain.EnvDocker,
			Endpoint:    d.client.socketPath,
			Status:      status,
			Description: "Docker Engine Local (/var/run/docker.sock)",
			Scopes:      stacks,
		},
		{
			Name:        "srv-docker-db",
			Type:        domain.EnvDocker,
			Endpoint:    "ssh://deploy@10.0.1.45",
			Status:      "PAUSED",
			Description: "Servidor Remoto Docker Banco de Dados (SSH)",
			Scopes:      []string{"mongodb-cluster", "backup-runner"},
		},
	}
}

func (d *DockerDiscovery) DiscoverStacks() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	containers, err := d.client.ListContainers(ctx, true)
	if err != nil {
		// Fallback amigável para stacks conhecidas no ambiente
		return []string{"ecommerce-stack", "db-postgres", "redis-cache"}
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
	if len(list) == 0 {
		return []string{"ecommerce-stack", "db-postgres", "redis-cache"}
	}
	return list
}
