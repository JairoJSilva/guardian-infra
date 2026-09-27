package actions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"guardian/internal/domain"
)

func TestTestIntegration_JiraMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/2/myself" {
			user, pass, ok := r.BasicAuth()
			if ok && user == "valid-user" && pass == "valid-pass" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"displayName": "Engenheiro SRE", "emailAddress": "sre@empresa.com"}`))
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// 1. Sucesso
	integSuccess := &domain.TicketingIntegration{
		Type:        domain.ProviderJira,
		BaseURL:     server.URL,
		AuthType:    "BASIC",
		Username:    "valid-user",
		TokenSecret: "valid-pass",
	}

	res := TestIntegration(context.Background(), integSuccess)
	if !res.Success {
		t.Fatalf("esperava sucesso no teste do Jira, obteve erro: %s", res.Message)
	}

	// 2. Falha de autenticação
	integFail := &domain.TicketingIntegration{
		Type:        domain.ProviderJira,
		BaseURL:     server.URL,
		AuthType:    "BASIC",
		Username:    "invalid-user",
		TokenSecret: "wrong-pass",
	}

	resFail := TestIntegration(context.Background(), integFail)
	if resFail.Success {
		t.Fatalf("esperava falha de autenticação no teste do Jira")
	}
	if resFail.StatusCode != http.StatusUnauthorized {
		t.Errorf("esperava status 401, obteve %d", resFail.StatusCode)
	}
}
