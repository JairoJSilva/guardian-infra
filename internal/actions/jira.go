package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"guardian/internal/config"
	"guardian/internal/domain"
)

type JiraClient struct {
	cfg            *config.Config
	httpClient     *http.Client
	issueTypeCache map[string]string
	cacheMu        sync.RWMutex
}

func NewJiraClient(cfg *config.Config) *JiraClient {
	return &JiraClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		issueTypeCache: make(map[string]string),
	}
}

type JiraIssueResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

func (j *JiraClient) resolveIssueType(projectKey string) string {
	j.cacheMu.RLock()
	if it, ok := j.issueTypeCache[projectKey]; ok && it != "" {
		j.cacheMu.RUnlock()
		return it
	}
	j.cacheMu.RUnlock()

	configuredType := strings.TrimSpace(j.cfg.JiraIssueType)
	if configuredType == "" {
		configuredType = "Bug"
	}

	// Consulta createmeta do Jira para este projeto
	metaURL := fmt.Sprintf("%s/rest/api/2/issue/createmeta?projectKeys=%s", j.cfg.JiraBaseURL, url.QueryEscape(projectKey))
	req, err := http.NewRequest(http.MethodGet, metaURL, nil)
	if err == nil {
		req.SetBasicAuth(j.cfg.JiraUser, j.cfg.JiraPassword)
		resp, err := j.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var meta struct {
				Projects []struct {
					Key        string `json:"key"`
					IssueTypes []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"issuetypes"`
				} `json:"projects"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&meta); err == nil && len(meta.Projects) > 0 {
				types := meta.Projects[0].IssueTypes
				found := ""
				// 1. Busca correspondência exata ou case-insensitive com tipo configurado
				for _, it := range types {
					if strings.EqualFold(it.Name, configuredType) {
						found = it.Name
						break
					}
				}
				// 2. Se não encontrou, busca por palavras-chave comuns de chamados
				if found == "" {
					keywords := []string{"incidente", "bug", "falha", "serviço", "solicitação", "problema", "task"}
					for _, kw := range keywords {
						for _, it := range types {
							if strings.Contains(strings.ToLower(it.Name), kw) {
								found = it.Name
								break
							}
						}
						if found != "" {
							break
						}
					}
				}
				// 3. Fallback para o primeiro tipo de item disponível no projeto
				if found == "" && len(types) > 0 {
					found = types[0].Name
				}

				if found != "" {
					j.cacheMu.Lock()
					j.issueTypeCache[projectKey] = found
					j.cacheMu.Unlock()
					log.Printf("[Jira] [Auto-Discovery] Tipo de pendência para projeto '%s': '%s'", projectKey, found)
					return found
				}
			}
		}
	}

	return configuredType
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

	issueTypeName := j.resolveIssueType(projectKey)

	fields := map[string]interface{}{
		"project": map[string]string{
			"key": projectKey,
		},
		"summary":     summary,
		"description": description,
		"issuetype": map[string]string{
			"name": issueTypeName,
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

	sb.WriteString("h2. (!) Incidente Detectado pelo Guardian (Supervisor Híbrido)\n\n")
	sb.WriteString("|| Parâmetro || Detalhes da Ocorrência ||\n")
	sb.WriteString(fmt.Sprintf("| *Tipo de Ambiente* | %s |\n", event.Type))
	sb.WriteString(fmt.Sprintf("| *Host / Cluster* | %s |\n", event.Environment))
	sb.WriteString(fmt.Sprintf("| *Escopo (Namespace / Stack)* | %s |\n", event.Scope))
	sb.WriteString(fmt.Sprintf("| *Entidade Afetada* | %s |\n", event.EntityName))
	if event.Image != "" {
		sb.WriteString(fmt.Sprintf("| *Imagem do Container* | %s |\n", event.Image))
	}
	sb.WriteString(fmt.Sprintf("| *Motivo da Falha* | %s |\n", event.Reason))
	sb.WriteString(fmt.Sprintf("| *Código de Saída (Exit Code)* | %d |\n", event.ExitCode))
	sb.WriteString(fmt.Sprintf("| *Data / Hora da Ocorrência* | %s |\n", event.Timestamp.Format("02/01/2006 15:04:05 -0700")))

	if target != nil {
		sb.WriteString(fmt.Sprintf("| *Target Guardian* | %s (ID: %s) |\n", target.Name, target.ID))
	}
	sb.WriteString("\n")

	sb.WriteString("h3. (i) Diagnóstico & Últimos Logs do Container antes da queda:\n")
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
