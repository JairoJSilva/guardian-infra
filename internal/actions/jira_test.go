package actions

import (
	"testing"
	"time"

	"guardian/internal/config"
	"guardian/internal/domain"
)

func TestJiraClient_DryRun(t *testing.T) {
	cfg := &config.Config{
		JiraBaseURL:    "https://jira.example.com",
		JiraUser:       "test-user",
		JiraPassword:   "test-pass",
		JiraProjectKey: "OPS",
		JiraIssueType:  "Bug",
		DryRun:         true,
	}

	client := NewJiraClient(cfg)

	event := &domain.IncidentEvent{
		ID:          "evt-test-1",
		Type:        domain.EnvKubernetes,
		TargetID:    "target-1",
		Environment: "k8s-prod",
		Scope:       "default",
		EntityName:  "my-app-pod-123",
		Reason:      "CrashLoopBackOff",
		ExitCode:    1,
		Logs:        "FATAL: out of memory",
		Timestamp:   time.Now(),
		Severity:    "CRITICAL",
	}

	key, err := client.CreateIncidentIssue(event, nil)
	if err != nil {
		t.Fatalf("Esperava sucesso no dry-run, obteve erro: %v", err)
	}

	if key == "" {
		t.Errorf("Esperava chave simulada retornada, obteve vazio")
	}
}
