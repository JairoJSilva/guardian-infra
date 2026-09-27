package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"guardian/internal/domain"
)

type TestResult struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message"`
	LatencyMs  int64  `json:"latency_ms"`
}

func TestIntegration(ctx context.Context, integ *domain.TicketingIntegration) *TestResult {
	if integ == nil {
		return &TestResult{Success: false, Message: "Dados da integração não informados"}
	}

	start := time.Now()
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	rawBase := strings.TrimRight(strings.TrimSpace(integ.BaseURL), "/")
	if rawBase == "" {
		return &TestResult{Success: false, Message: "URL Base do portal é obrigatória"}
	}

	switch integ.Type {
	case domain.ProviderJira:
		return testJira(ctx, client, rawBase, integ, start)
	case domain.ProviderGLPI:
		return testGLPI(ctx, client, rawBase, integ, start)
	case domain.ProviderMovidesk:
		return testMovidesk(ctx, client, rawBase, integ, start)
	default:
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Provedor não suportado para teste: %s", integ.Type),
		}
	}
}

func testJira(ctx context.Context, client *http.Client, baseURL string, integ *domain.TicketingIntegration, start time.Time) *TestResult {
	testURL := fmt.Sprintf("%s/rest/api/2/myself", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
	if err != nil {
		return &TestResult{Success: false, Message: fmt.Sprintf("URL inválida: %v", err)}
	}

	if integ.AuthType == "BEARER" || integ.Username == "" {
		req.Header.Set("Authorization", "Bearer "+integ.TokenSecret)
	} else {
		req.SetBasicAuth(integ.Username, integ.TokenSecret)
	}

	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &TestResult{
			Success:   false,
			Message:   fmt.Sprintf("Falha de conexão com Jira (%s): %v", baseURL, err),
			LatencyMs: latency,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var myself struct {
			DisplayName  string `json:"displayName"`
			EmailAddress string `json:"emailAddress"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&myself)
		msg := "Conectado com sucesso ao Jira!"
		if myself.DisplayName != "" {
			msg = fmt.Sprintf("Conectado com sucesso ao Jira! (Autenticado como: %s)", myself.DisplayName)
		}
		return &TestResult{
			Success:    true,
			StatusCode: resp.StatusCode,
			Message:    msg,
			LatencyMs:  latency,
		}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return &TestResult{
			Success:    false,
			StatusCode: resp.StatusCode,
			Message:    "Falha de autenticação no Jira (HTTP 401). Verifique o usuário e senha/API Token.",
			LatencyMs:  latency,
		}
	}

	if resp.StatusCode == http.StatusForbidden {
		return &TestResult{
			Success:    false,
			StatusCode: resp.StatusCode,
			Message:    "Acesso negado no Jira (HTTP 403). Verifique as permissões de usuário ou regras de CAPTCHA.",
			LatencyMs:  latency,
		}
	}

	return &TestResult{
		Success:    false,
		StatusCode: resp.StatusCode,
		Message:    fmt.Sprintf("Jira retornou status HTTP %d", resp.StatusCode),
		LatencyMs:  latency,
	}
}

func testGLPI(ctx context.Context, client *http.Client, baseURL string, integ *domain.TicketingIntegration, start time.Time) *TestResult {
	apiBase := baseURL
	if !strings.HasSuffix(apiBase, "/apirest.php") {
		apiBase = apiBase + "/apirest.php"
	}
	testURL := fmt.Sprintf("%s/initSession", apiBase)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
	if err != nil {
		return &TestResult{Success: false, Message: fmt.Sprintf("URL inválida: %v", err)}
	}

	if integ.Username != "" {
		req.Header.Set("App-Token", integ.Username)
	}
	if integ.TokenSecret != "" {
		req.Header.Set("Authorization", "user_token "+integ.TokenSecret)
	}

	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &TestResult{
			Success:   false,
			Message:   fmt.Sprintf("Falha de conexão com GLPI (%s): %v", apiBase, err),
			LatencyMs: latency,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var session struct {
			SessionToken string `json:"session_token"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&session)
		if session.SessionToken != "" {
			killReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/killSession", apiBase), nil)
			if killReq != nil {
				killReq.Header.Set("Session-Token", session.SessionToken)
				if integ.Username != "" {
					killReq.Header.Set("App-Token", integ.Username)
				}
				_, _ = client.Do(killReq)
			}
		}
		return &TestResult{
			Success:    true,
			StatusCode: resp.StatusCode,
			Message:    "Sessão autenticada e validada com sucesso na API REST do GLPI!",
			LatencyMs:  latency,
		}
	}

	return &TestResult{
		Success:    false,
		StatusCode: resp.StatusCode,
		Message:    fmt.Sprintf("GLPI retornou status HTTP %d. Verifique User-Token e App-Token.", resp.StatusCode),
		LatencyMs:  latency,
	}
}

func testMovidesk(ctx context.Context, client *http.Client, baseURL string, integ *domain.TicketingIntegration, start time.Time) *TestResult {
	apiBase := baseURL
	if !strings.Contains(apiBase, "api.movidesk.com") {
		apiBase = "https://api.movidesk.com/public/v1"
	}
	testURL := fmt.Sprintf("%s/services?token=%s", apiBase, url.QueryEscape(integ.TokenSecret))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
	if err != nil {
		return &TestResult{Success: false, Message: fmt.Sprintf("URL inválida: %v", err)}
	}

	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &TestResult{
			Success:   false,
			Message:   fmt.Sprintf("Falha de conexão com Movidesk: %v", err),
			LatencyMs: latency,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return &TestResult{
			Success:    true,
			StatusCode: resp.StatusCode,
			Message:    "Conexão e token validados com sucesso no Movidesk Web API!",
			LatencyMs:  latency,
		}
	}

	return &TestResult{
		Success:    false,
		StatusCode: resp.StatusCode,
		Message:    fmt.Sprintf("Movidesk retornou status HTTP %d. Verifique o Token da API.", resp.StatusCode),
		LatencyMs:  latency,
	}
}
