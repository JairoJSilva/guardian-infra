package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"guardian/internal/actions"
	"guardian/internal/config"
	"guardian/internal/domain"
	"guardian/internal/storage"
)

func TestAPI_Integrations_CRUD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "guardian-api-test-*")
	if err != nil {
		t.Fatalf("falha ao criar temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		HTTPPort:        8092,
		JiraBaseURL:     "https://jira.teste.com",
		JiraUser:        "user-test",
		JiraPassword:    "secret-pass",
		JiraProjectKey:  "OPS",
		CooldownMinutes: 60,
	}

	targetsPath := filepath.Join(tmpDir, "targets.json")
	integsPath := filepath.Join(tmpDir, "integrations.json")

	store, _ := storage.NewStorage(targetsPath)
	integStore, _ := storage.NewIntegrationStorage(integsPath, cfg)
	notif := actions.NewNotifier(10)

	server := NewServer(cfg, store, integStore, nil, notif, nil, nil)
	handler := server.Handler()

	// 1. GET /api/integrations (deve retornar seed default seguro com senha mascarada)
	req := httptest.NewRequest(http.MethodGet, "/api/integrations", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava status 200 no GET /api/integrations, obteve %d", rec.Code)
	}

	var list []*domain.TicketingIntegrationSafe
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("falha ao decodificar JSON: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("esperava 1 integração inicial, obteve %d", len(list))
	}
	if list[0].SecretMask != "••••••••" {
		t.Errorf("esperava máscara de segredo, obteve %s", list[0].SecretMask)
	}

	// 2. POST /api/integrations (criar novo portal GLPI)
	newPortal := domain.TicketingIntegration{
		Name:        "GLPI Produção",
		Type:        domain.ProviderGLPI,
		BaseURL:     "https://glpi.empresa.com",
		Username:    "app-token-x",
		TokenSecret: "user-secret-y",
		ProjectKey:  "12",
		Enabled:     true,
	}
	body, _ := json.Marshal(newPortal)
	reqPost := httptest.NewRequest(http.MethodPost, "/api/integrations", bytes.NewBuffer(body))
	reqPost.Header.Set("Content-Type", "application/json")
	recPost := httptest.NewRecorder()
	handler.ServeHTTP(recPost, reqPost)

	if recPost.Code != http.StatusCreated {
		t.Fatalf("esperava status 201 no POST, obteve %d: %s", recPost.Code, recPost.Body.String())
	}

	var created domain.TicketingIntegrationSafe
	_ = json.NewDecoder(recPost.Body).Decode(&created)
	if created.ID == "" || created.Name != "GLPI Produção" {
		t.Errorf("dados criados inconsistentes: %+v", created)
	}

	// 3. PUT /api/integrations/{id}
	updatePayload := map[string]interface{}{
		"name":         "GLPI Produção Atualizado",
		"token_secret": "••••••••", // mantem senha
		"enabled":      true,
	}
	upBody, _ := json.Marshal(updatePayload)
	reqPut := httptest.NewRequest(http.MethodPut, "/api/integrations/"+created.ID, bytes.NewBuffer(upBody))
	reqPut.Header.Set("Content-Type", "application/json")
	recPut := httptest.NewRecorder()
	handler.ServeHTTP(recPut, reqPut)

	if recPut.Code != http.StatusOK {
		t.Fatalf("esperava status 200 no PUT, obteve %d: %s", recPut.Code, recPut.Body.String())
	}

	// Verifica se a senha original foi mantida no storage
	savedInteg, _ := integStore.Get(created.ID)
	if savedInteg.TokenSecret != "user-secret-y" {
		t.Errorf("senha original deveria ser preservada quando enviado bullet mask, obteve %s", savedInteg.TokenSecret)
	}

	// 4. DELETE /api/integrations/{id}
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/integrations/"+created.ID, nil)
	recDel := httptest.NewRecorder()
	handler.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusOK {
		t.Fatalf("esperava status 200 no DELETE, obteve %d", recDel.Code)
	}
}
