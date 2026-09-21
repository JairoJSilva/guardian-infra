package domain

import (
	"crypto/sha256"
	"fmt"
	"time"
)

type EnvironmentType string

const (
	EnvKubernetes EnvironmentType = "KUBERNETES"
	EnvDocker     EnvironmentType = "DOCKER"
)

type TargetStatus string

const (
	StatusActive TargetStatus = "ACTIVE"
	StatusPaused TargetStatus = "PAUSED"
	StatusError  TargetStatus = "ERROR"
)

type MonitoringRules struct {
	WatchCrashLoop      bool `json:"watch_crash_loop"`
	WatchOOM            bool `json:"watch_oom"`
	WatchHealthCheck    bool `json:"watch_healthcheck"`
	PollIntervalSeconds int  `json:"poll_interval_seconds"`
}

type ActionConfig struct {
	CreateJiraIssue bool   `json:"create_jira_issue"`
	JiraProjectKey  string `json:"jira_project_key"`
	Contrato        string `json:"contrato"`
	NotifySlack     bool   `json:"notify_slack"`
	SlackWebhook    string `json:"slack_webhook"`
}

type Target struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`     // ex: "Pagamentos Produção"
	Type      EnvironmentType `json:"type"`     // KUBERNETES ou DOCKER
	Endpoint  string          `json:"endpoint"` // Contexto K8s ou Host Docker
	Scopes    []string        `json:"scopes"`   // K8s Namespaces OU Docker Compose Stacks
	Rules     MonitoringRules `json:"rules"`
	Actions   ActionConfig    `json:"actions"`
	Status    TargetStatus    `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type IncidentEvent struct {
	ID          string                 `json:"id"`
	Type        EnvironmentType        `json:"type"`        // KUBERNETES ou DOCKER
	TargetID    string                 `json:"target_id"`
	Environment string                 `json:"environment"` // ex: "aks-prod-brazil" ou "docker-local"
	Scope       string                 `json:"scope"`       // Namespace ou Compose Project
	EntityName  string                 `json:"entity_name"` // Nome do Pod ou Container
	Image       string                 `json:"image"`       // Imagem do container
	Reason      string                 `json:"reason"`      // OOMKilled, CrashLoopBackOff, Die (ExitCode != 0)
	ExitCode    int                    `json:"exit_code"`
	Logs        string                 `json:"logs"`        // Últimos 50 logs coletados antes do crash
	Details     map[string]interface{} `json:"details,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	JiraIssue   string                 `json:"jira_issue,omitempty"`
	JiraError   string                 `json:"jira_error,omitempty"`
	Severity    string                 `json:"severity,omitempty"` // CRITICAL, WARNING, INFO
}

// Fingerprint gera um hash único para deduplicação anti-spam de incidentes
func (e *IncidentEvent) Fingerprint() string {
	raw := fmt.Sprintf("%s:%s:%s:%s:%s", e.Type, e.Environment, e.Scope, e.EntityName, e.Reason)
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h[:8])
}

// EnvironmentInfo representa um host ou cluster detectado
type EnvironmentInfo struct {
	Name        string          `json:"name"`
	Type        EnvironmentType `json:"type"`
	Endpoint    string          `json:"endpoint"`
	Status      string          `json:"status"` // READY, ERROR, UNREACHABLE
	Description string          `json:"description"`
	Scopes      []string        `json:"scopes"` // Namespaces ou Compose Stacks disponíveis
}
