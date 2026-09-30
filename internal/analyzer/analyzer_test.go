package analyzer

import (
	"strings"
	"testing"

	"guardian/internal/domain"
)

func TestAnalyzeIncident_OOM(t *testing.T) {
	evt := &domain.IncidentEvent{
		Type:       domain.EnvKubernetes,
		Scope:      "pagamentos",
		EntityName: "api-checkout-774f7678c-xk8pm",
		Reason:     "OOMKilled",
		ExitCode:   137,
		Logs:       "Killed\nprocess out of memory",
	}

	res := AnalyzeIncident(evt)
	if !strings.Contains(res.Category, "MEMÓRIA") {
		t.Fatalf("Esperava categoria MEMÓRIA, obteve: %s", res.Category)
	}
	if len(res.ActionSteps) == 0 {
		t.Fatalf("Esperava passos de ação preenchidos")
	}
	if len(res.SuggestedCommands) == 0 {
		t.Fatalf("Esperava comandos sugeridos preenchidos")
	}
}

func TestAnalyzeIncident_Database(t *testing.T) {
	evt := &domain.IncidentEvent{
		Type:       domain.EnvKubernetes,
		Scope:      "financeiro",
		EntityName: "faturamento-backend-884bc-2m1lp",
		Reason:     "CrashLoopBackOff",
		ExitCode:   1,
		Logs:       "2026-09-30 dial tcp 10.0.1.50:5432: connect: connection refused\nfatal: could not connect to postgres",
	}

	res := AnalyzeIncident(evt)
	if !strings.Contains(res.Category, "PostgreSQL") {
		t.Fatalf("Esperava categoria PostgreSQL, obteve: %s", res.Category)
	}
	if !strings.Contains(res.RootCause, "PostgreSQL") {
		t.Fatalf("Esperava RootCause mencionando PostgreSQL")
	}
}

func TestAnalyzeIncident_Evicted(t *testing.T) {
	evt := &domain.IncidentEvent{
		Type:       domain.EnvKubernetes,
		Scope:      "default",
		EntityName: "worker-batch-56cd-498x",
		Reason:     "Evicted",
		ExitCode:   137,
		Logs:       "Pod The node was low on resource: ephemeral-storage. Container was using 1234KiB",
	}

	res := AnalyzeIncident(evt)
	if !strings.Contains(res.Category, "Evicted") {
		t.Fatalf("Esperava categoria Evicted, obteve: %s", res.Category)
	}
}
