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

type ProviderType string

const (
	ProviderJira     ProviderType = "JIRA"
	ProviderGLPI     ProviderType = "GLPI"
	ProviderMovidesk ProviderType = "MOVIDESK"
)

type TicketingIntegration struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        ProviderType `json:"type"` // JIRA, GLPI, MOVIDESK
	BaseURL     string       `json:"base_url"`
	AuthType    string       `json:"auth_type"` // BASIC, BEARER, API_KEY
	Username    string       `json:"username,omitempty"`
	TokenSecret string       `json:"token_secret,omitempty"`
	ProjectKey  string       `json:"project_key,omitempty"`  // Jira Project / Categoria / Fila
	DefaultType string       `json:"default_type,omitempty"` // Bug, Incident, etc.
	Enabled     bool         `json:"enabled"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type TicketingIntegrationSafe struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        ProviderType `json:"type"`
	BaseURL     string       `json:"base_url"`
	AuthType    string       `json:"auth_type"`
	Username    string       `json:"username,omitempty"`
	HasSecret   bool         `json:"has_secret"`
	SecretMask  string       `json:"secret_mask"`
	ProjectKey  string       `json:"project_key,omitempty"`
	DefaultType string       `json:"default_type,omitempty"`
	Enabled     bool         `json:"enabled"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func (t *TicketingIntegration) ToSafe() *TicketingIntegrationSafe {
	mask := ""
	hasSecret := false
	if t.TokenSecret != "" {
		hasSecret = true
		mask = "••••••••"
	}
	return &TicketingIntegrationSafe{
		ID:          t.ID,
		Name:        t.Name,
		Type:        t.Type,
		BaseURL:     t.BaseURL,
		AuthType:    t.AuthType,
		Username:    t.Username,
		HasSecret:   hasSecret,
		SecretMask:  mask,
		ProjectKey:  t.ProjectKey,
		DefaultType: t.DefaultType,
		Enabled:     t.Enabled,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

type ActionConfig struct {
	IntegrationID   string `json:"integration_id,omitempty"` // ID do portal de chamados selecionado
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

	// Análise Prévia e Sugestões SRE para Ação Humana
	AnalysisCategory  string   `json:"analysis_category,omitempty"`
	AnalysisSummary   string   `json:"analysis_summary,omitempty"`
	RootCause         string   `json:"root_cause,omitempty"`
	SuggestedFix      string   `json:"suggested_fix,omitempty"`
	ActionSteps       []string `json:"action_steps,omitempty"`
	SuggestedCommands []string `json:"suggested_commands,omitempty"`
}

// Fingerprint gera um hash único para deduplicação anti-spam de incidentes
func (e *IncidentEvent) Fingerprint() string {
	return FingerprintRaw(string(e.Type), e.Environment, e.Scope, e.EntityName, e.Reason)
}

// FingerprintRaw gera o mesmo hash sem precisar de um IncidentEvent completo.
// Usado pelo modo seeding do watcher para pré-registrar pods já existentes.
func FingerprintRaw(envType, environment, scope, entityName, reason string) string {
	raw := fmt.Sprintf("%s:%s:%s:%s:%s", envType, environment, scope, entityName, reason)
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
