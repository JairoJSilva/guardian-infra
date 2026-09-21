# 📝 RMT-20260917-01: Criação do GuardianOps v1.0, Motor RCA e Automação de Chamados no Jira OPS

> **RMT (Registro de Modificação Técnica)**  
> **Status**: Aplicada  
> **Data**: 2026-09-17  
> **Autor / Agente Responsável**: @DevOpsSenior & @SoftwareArchitect  
> **Tipo de Mudança**: Feature / Architecture / Observabilidade  
> **Versão Afetada**: v1.0.0  

---

## 1. 🎯 Contexto e Motivação
A equipe de operações e sustentação necessitava de um agente autônomo observador para monitorar falhas em workloads e contêineres e gerar chamados diagnósticos ricos no Jira (projeto `OPS`), eliminando o preenchimento manual de tickets e fornecendo diagnósticos de causa raiz com sugestões de remediação.

---

## 2. 📁 Arquivos e Componentes Afetados

| Arquivo / Caminho | Tipo de Alteração | Descrição Resumida |
|:---|:---|:---|
| `main.py` | Criado | Script principal de execução e orquestração |
| `guardian_ops/analyzer.py` | Criado | Motor heurístico de causa raiz e classificação de falhas |
| `guardian_ops/jira_client.py` | Criado | Integração REST v2 com Jira e mapeamento de custom fields |
| `guardian_ops/k8s_scanner.py` | Criado | Scanner e coletor de logs para Pods Kubernetes |
| `guardian_ops/templates.py` | Criado | Templates estruturados de formatação de incidentes |
| `k8s/rbac-readonly.yaml` | Criado | Configuração RBAC somente-leitura (`get, list, watch`) |
| `test_jira.py` | Criado | Script de validação de conectividade com a API do Jira |

---

## 3. ⚙️ Detalhamento Técnico das Modificações

### 3.1. Motor RCA (Root Cause Analysis)
- Implementação de regras de expressão regular e análise de código de saída (ExitCode) para identificar `OOMKilled`, `CrashLoopBackOff`, `ImagePullBackOff` e exceções de aplicação.

### 3.2. Integração com Jira
- Conexão autenticada via Basic Auth / API Token.
- Roteamento obrigatório para o projeto `OPS`.
- Preservação estrita de codificação UTF-8 para evitar caracteres corrompidos nas descrições de chamados.

### 3.3. Governança Somente-Leitura
- Definição do princípio de que o Guardian nunca executa alterações destrutivas em ambientes monitorados.

---

## 4. ⚠️ Impactos, Compatibilidade e Dependências
- **Variáveis de Ambiente**: Requer criação do `.env` com `JIRA_URL`, `JIRA_EMAIL`, `JIRA_API_TOKEN` e `JIRA_PROJECT_KEY`.
- **Breaking Changes**: Nenhuma (versão inicial).

---

## 5. 🧪 Testes e Validação
- [x] Conexão com a API do Jira validada com sucesso via `python3 test_jira.py`.
- [x] Criação de chamado de teste no projeto OPS.
- [x] Validação do manifesto RBAC em cluster de homologação.
