package analyzer

import (
	"os"
	"strings"
	"testing"

	"guardian/internal/domain"
)

func TestExtractJSONBlock(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Markdown block with json tag",
			input:    "Aqui está a análise:\n```json\n{\"category\": \"DB\", \"root_cause\": \"Error\"}\n```\nObrigado.",
			expected: `{"category": "DB", "root_cause": "Error"}`,
		},
		{
			name:     "Markdown block generic",
			input:    "```\n{\"category\": \"DB\"}\n```",
			expected: `{"category": "DB"}`,
		},
		{
			name:     "Raw JSON with spaces",
			input:    "   {\"category\": \"OOM\"}   ",
			expected: `{"category": "OOM"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := string(extractJSONBlock(tc.input))
			if got != tc.expected {
				t.Fatalf("Esperava %q, obteve %q", tc.expected, got)
			}
		})
	}
}

func TestAnalyzeIncident_DatabaseMigration_Heuristic(t *testing.T) {
	// Desabilita IA temporariamente para testar o fallback heurístico
	_ = os.Setenv("GUARDIAN_USE_AI", "false")
	defer os.Unsetenv("GUARDIAN_USE_AI")

	evt := &domain.IncidentEvent{
		Type:       domain.EnvKubernetes,
		Scope:      "issec",
		EntityName: "negociacao-dividas-api-5d5f5fd7cf-c9tm9",
		Reason:     "CrashLoopBackOff",
		ExitCode:   1,
		Logs: `Caused by: org.flywaydb.core.internal.sqlscript.FlywaySqlScriptException: Migration V1732268715__insert_idice_selic.sql failed
ERROR: relation "public.tb_componente_atualizacao_selic" does not exist`,
	}

	res := AnalyzeIncident(evt)
	if !strings.Contains(res.Category, "Flyway Migration") {
		t.Fatalf("Esperava categoria Flyway Migration, obteve: %s", res.Category)
	}
	if !strings.Contains(res.RootCause, "public.tb_componente_atualizacao_selic") && !strings.Contains(res.RootCause, "migração") {
		t.Fatalf("Esperava RootCause descrevendo a falha de migração, obteve: %s", res.RootCause)
	}
	if len(res.ActionSteps) == 0 {
		t.Fatalf("Esperava passos de ação preenchidos")
	}
}
