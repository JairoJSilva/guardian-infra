# 🛡️ GuardianOps (v1.0) — Documentação Executiva e Técnica

> **Status:** Versão 1.0 (Validação Local & Somente-Leitura)  
> **Data:** 17/09/2026  
> **Autor / Time:** Flowti Cloud & Platform Engineering  
> **Integração:** Jira Server / Data Center (Projeto OPS) & Kubernetes / GitLab CI  

---

## 1. 🎯 Visão Geral e Proposta de Valor

O **GuardianOps** é um agente autônomo observador projetado para residir futuramente dentro do Cluster Kubernetes, monitorando e diagnosticando continuamente qualquer falha ou anomalia originada em:
1. **Workloads de Aplicação (Kubernetes Pods)**: Quedas, reinicializações contínuas, saturação de recursos ou falhas de imagem.
2. **Esteiras de Integração Contínua (Pipelines CI/CD)**: Bloqueios em etapas de compilação, montagem Docker, testes unitários ou auditorias de segurança.

Sempre que um incidente ocorre, o GuardianOps:
* Coleta evidências e logs relevantes.
* Interpreta o problema através do seu **Motor de Análise de Causa Raiz (RCA)**.
* Gera um roteiro analítico de intervenção passo a passo para a equipe de engenharia.
* Cria automaticamente um chamado técnico no Jira dentro do projeto **OPS**, estritamente padronizado e com integridade de codificação **UTF-8**.

---

## 2. 🔒 Diretrizes Rígidas de Governança e Segurança (v1.0)

1. **Modo Estritamente Somente-Leitura (Read-Only)**:
   * **NÃO executa ações corretivas em hipótese alguma.**
   * Toda e qualquer ação de remediação é formulada como sugestão técnica para o operador humano.
   * O RBAC preparado para o Kubernetes concede apenas os verbos `["get", "list", "watch"]` em pods, logs e eventos. Verbos como `create`, `update`, `patch` e `delete` em workloads são inexistentes.
2. **Ambiente de Execução Atual (Sem Deploy no Cluster)**:
   * Nesta etapa inicial, o GuardianOps não foi aplicado em nenhum cluster Kubernetes ativo.
   * Toda a solução roda de forma leve, modular e autocontida localmente em Python (`.venv`), contando com mocks de simulação, testes automatizados e listener de webhooks.
   * Os manifestos declarativos estão organizados na pasta `k8s/` para aplicação futura.

---

## 3. 🧩 Arquitetura da Solução

```
                    ┌─────────────────────────┐
                    │      Origens de Falhas  │
                    └────────────┬────────────┘
                                 │
           ┌─────────────────────┴─────────────────────┐
           ▼                                           ▼
┌──────────────────────┐                   ┌──────────────────────┐
│    Kubernetes Pods   │                   │  Pipelines CI/CD     │
│  - CrashLoopBackOff  │                   │  - Docker Build Fail │
│  - OOMKilled (137)   │                   │  - Lint / Test Error │
│  - ImagePullBackOff  │                   │  - Deploy Timeout    │
└──────────┬───────────┘                   └──────────┬───────────┘
           │ (k8s_scanner.py)                         │ (Webhook POST /webhook/gitlab)
           └─────────────────────┬─────────────────────┘
                                 ▼
                 ┌───────────────────────────────┐
                 │    GuardianOps Core (v1.0)    │
                 │                               │
                 │ 1. Filtro & Deduplicação      │
                 │    (Cooldown Anti-Spam)       │
                 │                               │
                 │ 2. Motor Analítico RCA        │
                 │    (Diagnóstico + Solução)    │
                 │                               │
                 │ 3. Template Jira Institucional│
                 │    (UTF-8 Estrito / No-Break) │
                 └───────────────┬───────────────┘
                                 │
                                 ▼ (REST API v2 / UTF-8 JSON)
                 ┌───────────────────────────────┐
                 │    Jira Platform (Proj: OPS)  │
                 │  - Chamado Aberto Detalhado   │
                 │  - Vínculo com Issue Pai      │
                 └───────────────────────────────┘
```

---

## 4. ⚙️ Pilares de Engenharia Implementados

### 4.1. Integridade de Codificação UTF-8
Para evitar corrupção de caracteres no Jira Server/Data Center:
* Serialização de dados via `json.dumps(payload, ensure_ascii=False)`.
* Envio de bytes literais em UTF-8: `.encode('utf-8')`.
* Header HTTP explícito: `Content-Type: application/json; charset=utf-8`.
* Suporte nativo a acentuação (`á`, `ç`, `õ`, `ê`) e símbolos/emojis visuais (`🛡️`, `🤖`, `📌`, `⚠️`).

### 4.2. Deduplicação e Cooldown (Anti-Spam)
Quando uma aplicação entra em `CrashLoopBackOff`, ela reinicia repetidamente em intervalos de segundos ou minutos. Sem controle, geraria centenas de chamados no Jira.
* O GuardianOps gera um hash `MD5` único combinando: `origem + namespace + identificador + tipo_de_falha`.
* Aplica uma janela de **Cooldown** (configurável, padrão de 60 minutos).
* Novas detecções da mesma falha dentro da janela são registradas em log mas descartadas para abertura de chamados.

### 4.3. Motor de Análise de Causa Raiz (RCA)
Classificação inteligente por padrões comuns de produção:

| Cenário de Falha | Diagnóstico Identificado | Ação Sugerida no Chamado |
| :--- | :--- | :--- |
| **OOMKilled** | Exit Code 137. Container estourou `limits.memory` ou vazamento de memória. | Inspecionar `kubectl top pod`, aumentar limites de memória em 25-50% via GitOps e auditar logs de consumo. |
| **CrashLoopBackOff** | Exit Code 1/2/127. Erro logo após o start. Falha de banco, secret ausente. | Inspecionar `kubectl logs --previous`, checar ConfigMaps/Secrets e conectividade com bancos de dados. |
| **ImagePullBackOff** | Tag inexistente no Container Registry ou erro no `imagePullSecrets`. | Verificar tag no repositório Docker/GitLab, validar segredo de pull e aplicar revisão correta. |
| **Failed Probes** | HTTP 500 ou Timeout na verificação de Liveness/Readiness. | Ajustar `initialDelaySeconds`, testar rota interna de saúde e auditar performance do processo. |
| **Pipeline Failure** | Erro de dependência (`npm ci`, `pip`, `maven`), conflito Docker ou quebra de testes. | Informar branch/commit, reproduzir erro no ambiente local e aplicar `git revert` se necessário. |

---

## 5. 📋 Padrão do Chamado Gerado no Jira

Abaixo está o modelo exato gerado e validado em UTF-8:

```jira
*Referência:* OPS-GUARDIAN
*Data:* 17/09/2026 11:27:07
*Ambiente:* AWS Produção (EKS)
*Origem:* Gerado por I.A. 🤖 (GuardianOps 🛡️)
*Tag de Rastreabilidade:* {{gerado-por-ia}}, {{guardian-ops}}

----

h2. 📌 Incidente: Pod 'portal-flowti-backend-69b7bf656b-x82d9' em CrashLoopBackOff (Namespace: producao)

* *Tipo:* Solicitação de serviço
* *Prioridade:* Alta
* *Categoria:* Kubernetes Workload / Estabilidade de Aplicação
* *Tag / Rótulo:* {{gerado-por-ia}}, {{guardian-ops}}

h3. 1. Diagnóstico & Análise de Causa Raiz (RCA)
O processo principal do container *flowti-api* está falhando imediatamente após a inicialização (Exit Code: 1). O Kubelet tentou reiniciar o container repetidamente (8 reinicializações), ativando a política de espera exponencial (_back-off delay_). Causas frequentes: Variáveis de ambiente/Secrets ausentes, falha de conexão com banco de dados/cache ou exceção fatal não tratada.

*Evidências Coletadas:*
{code:text}
[2026-09-17 11:20:04] ERROR [DatabasePool] Connection to PostgreSQL at 'db-prod.internal:5432' failed: Connection timed out.
org.postgresql.util.PSQLException: The connection attempt timed out after 5000ms.
[2026-09-17 11:20:05] SYSTEM [Process] Exited with status code 1.
{code}

h3. 2. Ativos Impactados
|| Namespace || Nome do Pod || Container || Motivo da Falha || Reinicializações ||
| producao | {{ portal-flowti-backend-69b7bf656b-x82d9 }} | {{ flowti-api }} | *CrashLoopBackOff* | 8 |

h3. 3. Procedimento de Correção Sugerido (Step-by-Step)
> *AVISO DE GOVERNANÇA:* O *GuardianOps* atua exclusivamente em modo de *SOMENTE-LEITURA* (Read-Only). Nenhuma modificação automática foi realizada no cluster. A equipe responsável deve executar o roteiro abaixo manualmente:

1. Visualizar os logs da última execução do container antes de cair:
{code:bash}
kubectl logs portal-flowti-backend-69b7bf656b-x82d9 -n producao -c flowti-api --previous --tail=100
{code}
2. Inspecionar os eventos recentes do ciclo de vida do Pod:
{code:bash}
kubectl describe pod portal-flowti-backend-69b7bf656b-x82d9 -n producao
{code}
3. Validar se os Secrets e ConfigMaps referenciados pelo Deployment existem no namespace:
{code:bash}
kubectl get configmaps,secrets -n producao
{code}
4. Corrigir a causa-raiz identificada nos logs via repositório de código/infraestrutura.

h3. 4. Critérios de Aceite
* Pod *portal-flowti-backend-69b7bf656b-x82d9* atinge status {Running} e passa em todas as checagens de liveness/readiness.
* Contador de reinicializações cessa o incremento.

h3. 5. Plano de Rollback
Reverter a última alteração no repositório de configuração ou aplicar rollback do Deployment:
{code:bash}kubectl rollout undo deployment/<deployment-name> -n producao{code}
```

---

## 6. 🧪 Roteiro de Testes e Simulações Locais

Todo o projeto é executável através do CLI central `main.py` com o ambiente virtual `.venv`:

```bash
# 1. Executar bateria de testes automatizada (UTF-8, RCA, Cooldown, Segurança)
.venv/bin/python3 test_guardian.py

# 2. Testar autenticação com o Jira
.venv/bin/python3 main.py test-jira

# 3. Simular falha de Pod em modo Dry-Run (não abre chamado, apenas exibe formatação)
.venv/bin/python3 main.py simulate-pod --dry-run
.venv/bin/python3 main.py simulate-pod --type OOMKilled --exit-code 137 --dry-run

# 4. Simular falha de Pipeline em modo Dry-Run
.venv/bin/python3 main.py simulate-pipeline --dry-run

# 5. Criar chamado real no Jira vinculado ao pai (ex: OPS-236)
.venv/bin/python3 main.py simulate-pod --parent OPS-236

# 6. Iniciar listener HTTP para Webhooks de CI/CD (porta 8080)
.venv/bin/python3 main.py webhook --port 8080 --dry-run
```

---

## 7. ☸️ Preparação para Futura Implantação no Kubernetes

Quando a equipe atingir a maturidade desejada para subir o GuardianOps dentro do cluster, os manifestos em `k8s/` já cumprem 100% dos padrões da engenharia de plataformas:
* **Deployment Declarativo** (`k8s/deployment.yaml`): 2 réplicas ativas, estratégia de `RollingUpdate` com zero downtime.
* **RBAC de Mínimo Privilégio** (`k8s/rbac-readonly.yaml`): Permissões restritas a `get`, `list` e `watch`.
* **Segurança de Container** (`Dockerfile`): Imagem sem privilégios de root (UID `10001`), com healthchecks nativos.
* **Abstração por Service** (`ClusterIP`): Ponto estável de rede interno para receber webhooks das esteiras.
