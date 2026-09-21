package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"guardian/internal/config"
	"guardian/internal/domain"
)

type JiraClient struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewJiraClient(cfg *config.Config) *JiraClient {
	return &JiraClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type JiraIssueResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

func (j *JiraClient) CreateIncidentIssue(event *domain.IncidentEvent, target *domain.Target) (string, error) {
	summary := fmt.Sprintf("[%s] %s em %s (%s)", event.Type, event.EntityName, event.Reason, event.Scope)
	if len(summary) > 250 {
		summary = summary[:247] + "..."
	}

	description := j.buildDescription(event, target)

	projectKey := j.cfg.JiraProjectKey
	if target != nil && target.Actions.JiraProjectKey != "" {
		projectKey = target.Actions.JiraProjectKey
	}

	hint := event.Scope + " " + event.EntityName
	if target != nil && target.Actions.Contrato != "" {
		hint = target.Actions.Contrato
	}
	contract := j.cfg.ResolveContract(hint)

	priorityName := "Alta"
	if event.Severity == "WARNING" || event.Severity == "INFO" {
		priorityName = "Média"
	}

	fields := map[string]interface{}{
		"project": map[string]string{
			"key": projectKey,
		},
		"summary":     summary,
		"description": description,
		"issuetype": map[string]string{
			"name": j.cfg.JiraIssueType,
		},
		"priority": map[string]string{
			"name": priorityName,
		},
		"labels": []string{
			"guardian-ops",
			"arquitetura-hibrida",
			strings.ToLower(string(event.Type)),
		},
	}

	if j.cfg.JiraCustomfieldContratoID != "" && contract.ID != "" {
		fields[j.cfg.JiraCustomfieldContratoID] = map[string]string{
			"id":    contract.ID,
			"value": contract.Value,
		}
	}

	if j.cfg.JiraUser != "" {
		fields["reporter"] = map[string]string{
			"name": j.cfg.JiraUser,
		}
	}

	payload := map[string]interface{}{
		"fields": fields,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("falha ao serializar payload Jira: %w", err)
	}

	if j.cfg.IsDryRun() {
		mockKey := fmt.Sprintf("DRYRUN-%s-%d", projectKey, time.Now().Unix()%10000)
		log.Printf("[Jira] [DRY-RUN] Simulação de chamado Jira bem-sucedida! Chave simulada: %s", mockKey)
		log.Printf("[Jira] [DRY-RUN] Contrato: %s (ID: %s) | Resumo: %s", contract.Value, contract.ID, summary)
		return mockKey, nil
	}

	if j.cfg.JiraUser == "" || j.cfg.JiraPassword == "" {
		return "", fmt.Errorf("credenciais do Jira não configuradas (.env JIRA_USER e JIRA_PASSWORD)")
	}

	url := fmt.Sprintf("%s/rest/api/2/issue", j.cfg.JiraBaseURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("falha ao criar requisição HTTP: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.SetBasicAuth(j.cfg.JiraUser, j.cfg.JiraPassword)

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro de conexão com Jira em %s: %w", url, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("jira retornou HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var jiraResp JiraIssueResponse
	if err := json.Unmarshal(bodyBytes, &jiraResp); err != nil {
		return "", fmt.Errorf("falha ao decodificar resposta do Jira: %w", err)
	}

	log.Printf("[Jira] [✓] Chamado criado com sucesso: %s (URL: %s/browse/%s)", jiraResp.Key, j.cfg.JiraBaseURL, jiraResp.Key)
	return jiraResp.Key, nil
}

func (j *JiraClient) buildDescription(event *domain.IncidentEvent, target *domain.Target) string {
	var sb strings.Builder

	sb.WriteString("h2. 🛡️ Incidente Detectado pelo Guardian (Supervisor Híbrido)\n\n")
	sb.WriteString(fmt.Sprintf("* *Tipo de Ambiente:* %s\n", event.Type))
	sb.WriteString(fmt.Sprintf("* *Host / Cluster:* %s\n", event.Environment))
	sb.WriteString(fmt.Sprintf("* *Escopo (Namespace / Stack):* %s\n", event.Scope))
	sb.WriteString(fmt.Sprintf("* *Entidade Afetada:* %s\n", event.EntityName))
	if event.Image != "" {
		sb.WriteString(fmt.Sprintf("* *Imagem do Container:* %s\n", event.Image))
	}
	sb.WriteString(fmt.Sprintf("* *Motivo da Falha:* %s\n", event.Reason))
	sb.WriteString(fmt.Sprintf("* *Código de Saída (Exit Code):* %d\n", event.ExitCode))
	sb.WriteString(fmt.Sprintf("* *Data / Hora da Ocorrência:* %s\n\n", event.Timestamp.Format("02/01/2006 15:04:05 -0700")))

	if target != nil {
		sb.WriteString(fmt.Sprintf("* *Target Guardian:* %s (ID: %s)\n\n", target.Name, target.ID))
	}

	sb.WriteString("h3. 📄 Últimos Logs do Container antes da queda:\n")
	sb.WriteString("{code:bash}\n")
	if strings.TrimSpace(event.Logs) != "" {
		sb.WriteString(event.Logs)
	} else {
		sb.WriteString("Nenhum log capturado no buffer de saída do container.")
	}
	sb.WriteString("\n{code}\n\n")

	sb.WriteString("----\n_Chamado aberto automaticamente pelo GuardianOps v2.0 Architecture._\n")
	return sb.String()
}
