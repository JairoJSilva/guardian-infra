package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"guardian/internal/domain"
)

type agyEnvelope struct {
	Status   string `json:"status"`
	Response string `json:"response"`
}

// IsAIAnalysisEnabled verifica se a análise por IA via Antigravity local está habilitada.
// Por padrão, está habilitada caso a variável GUARDIAN_USE_AI não seja false e o binário agy/antigravity exista.
func IsAIAnalysisEnabled() bool {
	if val := strings.ToLower(os.Getenv("GUARDIAN_USE_AI")); val == "false" || val == "0" || val == "no" {
		return false
	}
	_, err := resolveAntigravityBinary()
	return err == nil
}

func resolveAntigravityBinary() (string, error) {
	if custom := os.Getenv("ANTIGRAVITY_BIN"); custom != "" {
		if path, err := exec.LookPath(custom); err == nil {
			return path, nil
		}
	}
	if path, err := exec.LookPath("agy"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("antigravity"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("binário agy/antigravity não encontrado no PATH")
}

// AnalyzeWithAntigravity invoca a CLI local do Antigravity em modo estritamente LEITURA (read-only)
// para realizar uma análise prévia e cirúrgica de causa raiz a partir dos logs e metadados do incidente.
func AnalyzeWithAntigravity(ctx context.Context, event *domain.IncidentEvent) (*AnalysisResult, error) {
	binPath, err := resolveAntigravityBinary()
	if err != nil {
		return nil, err
	}

	timeout := 15 * time.Second
	if tStr := os.Getenv("GUARDIAN_AI_TIMEOUT"); tStr != "" {
		if d, err := time.ParseDuration(tStr); err == nil && d > 0 {
			timeout = d
		}
	}

	subCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	logsSnippet := strings.TrimSpace(event.Logs)
	if len(logsSnippet) > 4000 {
		// Preserva o início e o final dos logs onde geralmente ficam os stack traces
		logsSnippet = logsSnippet[len(logsSnippet)-4000:]
	}
	if logsSnippet == "" {
		logsSnippet = "(Nenhum log adicional retornado pelo container anterior)"
	}

	prompt := fmt.Sprintf(`ATENÇÃO CRÍTICA DE SEGURANÇA E COMPLIANCE:
Você é um agente de análise SRE atuando em modo ESTRITAMENTE LEITURA (READ-ONLY).
É TERMINANTEMENTE PROIBIDO executar comandos de modificação, exclusão, restart ou escrita no ambiente ou nos bancos de dados.
Sua função é APENAS analisar tecnicamente os logs e o erro fornecido e retornar o diagnóstico estruturado em formato JSON para que um operador humano decida e atue manualmente.

Dados do Incidente:
- Ambiente / Cluster: %s
- Escopo / Namespace: %s
- Entidade / Pod: %s
- Imagem: %s
- Motivo: %s
- Código de Saída (ExitCode): %d
- Logs coletados:
%s

Instruções para o JSON de saída:
Responda EXCLUSIVAMENTE com um objeto JSON válido, sem texto explicativo antes ou depois, com o seguinte schema:
{
  "category": "Categoria concisa do problema (ex: Database Migration / Flyway, Falha de Conexão, etc)",
  "summary": "Resumo objetivo do que causou o crash em 1 frase",
  "root_cause": "Explicação técnica detalhada da causa raiz encontrada nos logs",
  "suggested_fix": "Sugestão clara de correção para o operador humano",
  "action_steps": ["Passo 1 a ser feito manualmente", "Passo 2..."],
  "suggested_commands": ["Comandos de troubleshooting / leitura sugeridos para o operador rodar manualmente"]
}
`, event.Environment, event.Scope, event.EntityName, event.Image, event.Reason, event.ExitCode, logsSnippet)

	cmd := exec.CommandContext(subCtx, binPath, "--print", prompt, "--output-format", "json", "--dangerously-skip-permissions")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("falha ao executar %s: %w", binPath, err)
	}

	var envelope agyEnvelope
	var rawJSON []byte

	if err := json.Unmarshal(output, &envelope); err == nil && envelope.Response != "" {
		rawJSON = extractJSONBlock(envelope.Response)
	} else {
		rawJSON = extractJSONBlock(string(output))
	}

	var result AnalysisResult
	if err := json.Unmarshal(rawJSON, &result); err != nil {
		return nil, fmt.Errorf("falha ao decodificar JSON retornado pela IA: %w (conteúdo bruto: %s)", err, string(rawJSON))
	}

	if result.Category == "" || result.RootCause == "" {
		return nil, fmt.Errorf("resposta da IA incompleta: campos obrigatórios ausentes")
	}

	log.Printf("[Analyzer/AI] ✅ Análise de causa raiz gerada com sucesso via Antigravity: [%s] %s", result.Category, result.Summary)
	return &result, nil
}

func extractJSONBlock(text string) []byte {
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "```json"); idx != -1 {
		rest := text[idx+len("```json"):]
		if endIdx := strings.Index(rest, "```"); endIdx != -1 {
			return []byte(strings.TrimSpace(rest[:endIdx]))
		}
	} else if idx := strings.Index(text, "```"); idx != -1 {
		rest := text[idx+len("```"):]
		if endIdx := strings.Index(rest, "```"); endIdx != -1 {
			return []byte(strings.TrimSpace(rest[:endIdx]))
		}
	}

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start != -1 && end != -1 && end > start {
		return []byte(strings.TrimSpace(text[start : end+1]))
	}

	return []byte(text)
}
