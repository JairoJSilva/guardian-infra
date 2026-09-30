package actions

import (
	"strings"
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

func TestJiraClient_BuildDescriptionWithAnalysis(t *testing.T) {
	client := &JiraClient{}
	event := &domain.IncidentEvent{
		Type:              domain.EnvKubernetes,
		Scope:             "pagamentos",
		EntityName:        "checkout-pod-99",
		Reason:            "OOMKilled",
		ExitCode:          137,
		Logs:              "fatal: out of memory",
		Timestamp:         time.Now(),
		AnalysisCategory:  "MEMÓRIA (OOMKilled)",
		AnalysisSummary:   "Container abortado por estouro de RAM.",
		RootCause:         "Processo excedeu o limits.memory de 512Mi.",
		SuggestedFix:      "Aumente limits.memory para 1Gi.",
		ActionSteps:       []string{"1. Verifique consumo: kubectl top pod", "2. Edite deployment"},
		SuggestedCommands: []string{"kubectl describe pod checkout-pod-99 -n pagamentos"},
	}

	desc := client.buildDescription(event, nil)
	if !strings.Contains(desc, "Análise Prévia da Causa Raiz") {
		t.Fatalf("Esperava painel de Análise Prévia na descrição")
	}
	if !strings.Contains(desc, "Sugestão de Correção & Ajuste") {
		t.Fatalf("Esperava painel de Sugestões na descrição")
	}
	if !strings.Contains(desc, "Modo SRE Informativo") {
		t.Fatalf("Esperava aviso de modo informativo na descrição")
	}
}
