package docker

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"guardian/internal/domain"
)

type DockerWatcher struct {
	client *DockerClient
	target *domain.Target
}

func NewDockerWatcher(client *DockerClient, target *domain.Target) *DockerWatcher {
	return &DockerWatcher{
		client: client,
		target: target,
	}
}

func (w *DockerWatcher) Watch(ctx context.Context, out chan<- *domain.IncidentEvent) error {
	interval := time.Duration(w.target.Rules.PollIntervalSeconds) * time.Second
	if interval < 5*time.Second {
		interval = 10 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[DockerWatcher] [Target: %s] Iniciando observação ativa com intervalo de %v para os escopos %v...", w.target.Name, interval, w.target.Scopes)

	// Primeira execução imediata
	w.scan(ctx, out)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[DockerWatcher] [Target: %s] Observador encerrado via cancelamento de contexto.", w.target.Name)
			return nil
		case <-ticker.C:
			w.scan(ctx, out)
		}
	}
}

func (w *DockerWatcher) scan(ctx context.Context, out chan<- *domain.IncidentEvent) {
	containers, err := w.client.ListContainers(ctx, true)
	if err != nil {
		return
	}

	for _, c := range containers {
		// Verifica se o container pertence a algum dos escopos monitorados
		matchedScope := ""
		for _, s := range w.target.Scopes {
			if proj, ok := c.Labels["com.docker.compose.project"]; ok && proj == s {
				matchedScope = s
				break
			}
			for _, name := range c.Names {
				clean := strings.TrimPrefix(name, "/")
				if clean == s || strings.HasPrefix(clean, s) {
					matchedScope = s
					break
				}
			}
			if matchedScope != "" {
				break
			}
		}

		if matchedScope == "" {
			continue
		}

		inspect, err := w.client.InspectContainer(ctx, c.ID)
		if err != nil {
			continue
		}

		name := strings.TrimPrefix(inspect.Name, "/")

		// Análise de falhas
		isOOM := inspect.State.OOMKilled
		isExitedWithError := inspect.State.Status == "exited" && inspect.State.ExitCode != 0
		isDead := inspect.State.Dead
		isRestarting := inspect.State.Restarting

		if !isOOM && !isExitedWithError && !isDead && !isRestarting {
			continue
		}

		reason := "ContainerCrash"
		severity := "WARNING"

		if isOOM {
			reason = "OOMKilled"
			severity = "CRITICAL"
		} else if isExitedWithError {
			reason = fmt.Sprintf("Die (ExitCode %d)", inspect.State.ExitCode)
			severity = "CRITICAL"
		} else if isRestarting {
			reason = "CrashLoopRestarting"
			severity = "WARNING"
		}

		// Coleta últimos 50 logs do container
		logs, _ := w.client.GetLogs(ctx, c.ID, 50)

		event := &domain.IncidentEvent{
			ID:          fmt.Sprintf("evt-docker-%d", time.Now().UnixNano()),
			Type:        domain.EnvDocker,
			TargetID:    w.target.ID,
			Environment: w.target.Endpoint,
			Scope:       matchedScope,
			EntityName:  name,
			Image:       inspect.Config.Image,
			Reason:      reason,
			ExitCode:    inspect.State.ExitCode,
			Logs:        logs,
			Timestamp:   time.Now(),
			Severity:    severity,
			Details: map[string]interface{}{
				"restart_count": inspect.RestartCount,
				"error":         inspect.State.Error,
			},
		}

		select {
		case out <- event:
		default:
		}
	}
}
