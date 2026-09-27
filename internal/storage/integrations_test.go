package storage

import (
	"os"
	"path/filepath"
	"testing"

	"guardian/internal/config"
	"guardian/internal/domain"
)

func TestIntegrationStorage_CRUD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "guardian-test-*")
	if err != nil {
		t.Fatalf("falha ao criar temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "integrations.json")
	cfg := &config.Config{
		JiraBaseURL:    "https://jira.teste.com",
		JiraUser:       "user-test",
		JiraPassword:   "secret-pass",
		JiraProjectKey: "OPS",
		JiraIssueType:  "Bug",
	}

	store, err := NewIntegrationStorage(filePath, cfg)
	if err != nil {
		t.Fatalf("erro ao criar storage: %v", err)
	}

	// 1. Deve conter seed default do Jira
	list := store.List()
	if len(list) != 1 {
		t.Fatalf("esperava 1 integração inicial (seed), obteve %d", len(list))
	}

	safeList := store.ListSafe()
	if len(safeList) != 1 {
		t.Fatalf("esperava 1 integração safe, obteve %d", len(safeList))
	}
	if safeList[0].SecretMask != "••••••••" {
		t.Errorf("esperava máscara de senha '••••••••', obteve %s", safeList[0].SecretMask)
	}
	if !safeList[0].HasSecret {
		t.Errorf("esperava HasSecret == true")
	}

	// 2. Adicionar nova integração GLPI
	glpi := &domain.TicketingIntegration{
		ID:          "glpi-corp",
		Name:        "GLPI Suporte TI",
		Type:        domain.ProviderGLPI,
		BaseURL:     "https://glpi.empresa.com",
		Username:    "app-token-123",
		TokenSecret: "user-token-456",
		Enabled:     true,
	}

	if err := store.Save(glpi); err != nil {
		t.Fatalf("falha ao salvar GLPI: %v", err)
	}

	got, err := store.Get("glpi-corp")
	if err != nil {
		t.Fatalf("falha ao obter GLPI salvo: %v", err)
	}
	if got.Name != "GLPI Suporte TI" {
		t.Errorf("nome inesperado: %s", got.Name)
	}

	// 3. Toggle Enabled
	enabled, err := store.ToggleEnabled("glpi-corp")
	if err != nil {
		t.Fatalf("erro no toggle: %v", err)
	}
	if enabled {
		t.Errorf("esperava enabled == false após toggle")
	}

	// 4. Delete
	if err := store.Delete("glpi-corp"); err != nil {
		t.Fatalf("erro ao deletar: %v", err)
	}
	if _, err := store.Get("glpi-corp"); err == nil {
		t.Errorf("esperava erro ao buscar item deletado")
	}
}
