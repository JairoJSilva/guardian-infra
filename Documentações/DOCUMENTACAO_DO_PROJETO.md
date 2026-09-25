# 🛡️ Guardian / GuardianOps — Documentação Técnica Oficial Completa

> **Projeto**: Guardian (GuardianOps)  
> **Versão**: 2.1 (Supervisor Híbrido Go + Motor Multi-Agente Python)  
> **Classificação**: Observabilidade Ativa, Supervisão de Targets e Automação de Incidentes  
> **Status**: Em Produção / Validação Híbrida  
> **Integrações**: Jira Server / Data Center (Projeto OPS), Docker Engine API e Kubernetes Clusters  

<p align="center">
  <img src="imagens/guardian-dashboard.jpg" alt="Guardian SRE Command Center Dashboard" width="100%" />
</p>

---

## 1. 🎯 Visão Executiva e Proposta de Valor

O **Guardian** é uma plataforma de observabilidade ativa e resposta diagnóstica a incidentes de infraestrutura e esteiras de desenvolvimento. Diferente de ferramentas tradicionais de telemetria que apenas acumulam dashboards passivos, o Guardian:

1. **Supervisiona Alvos Dinamicamente (Targets)**: Em vez de varrer continuamente toda a máquina ou cluster, o operador define quais *Namespaces* (Kubernetes) ou *Projetos/Stacks* (Docker Compose) devem estar sob vigilância.
2. **Diagnostica com Agentes Especialistas (Motor RCA)**: Quando um Pod entra em `CrashLoopBackOff`/`OOMKilled` ou um container morre com `exitCode != 0`, agentes autônomos especializados em DevOps, Banco de Dados, QA e Fullstack analisam os logs e o estado do runtime para determinar a causa raiz.
3. **Automatiza Chamados Técnicos no Jira (Projeto OPS)**: Produz um relatório técnico mastigado no Jira da equipe, com logs contextuais, causa raiz identificada e passo a passo claro para correção humana.
4. **Anti-Spam & Deduplicação Robusta**: Impede abertura de chamados repetidos para a mesma falha durante o período de cooldown (cache persistente e TTL em memória).
5. **Governança Estritamente Somente-Leitura (Read-Only)**: Zero risco à infraestrutura. O Guardian nunca deleta, escala ou edita recursos em produção; toda a intervenção é entregue como sugestão técnica ao operador.

---

## 2. 🧩 Diagrama Geral da Arquitetura

```
 ┌────────────────────────────────────────────────────────────────────────┐
 │                   INTERFACE WEB (Neo-Glassmorphism UI)                 │
 │            Dashboard de Targets, Live Stream & Ações Rápidas           │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     │ HTTP REST / SSE (Porta 8080)
                                     ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │                     GUARDIAN CORE ENGINE (Go)                          │
 │                                                                        │
 │   ┌──────────────────────┐         ┌───────────────────────────────┐   │
 │   │  API Server (Mux)    │         │  Dynamic Target Supervisor    │   │
 │   │  - /api/targets      │ ──────> │  - Worker Lifecycle (Context) │   │
 │   │  - /api/health       │         │  - targets.json Persistence   │   │
 │   └──────────────────────┘         └───────────────┬───────────────┘   │
 │                                                    │                   │
 │                     ┌──────────────────────────────┴───────────────┐   │
 │                     ▼                                              ▼   │
 │        ┌────────────────────────┐                    ┌─────────────┴──────────┐
 │        │  K8s Provider          │                    │  Docker Provider       │
 │        │  - client-go Informers │                    │  - Docker SDK Stream   │
 │        │  - Watch Namespaces    │                    │  - Events por Stack    │
 │        └────────────┬───────────┘                    └─────────────┬──────────┘
 │                     │                                              │   │
 │                     └──────────────────────┬───────────────────────┘   │
 │                                            ▼                           │
 │                           ┌─────────────────────────────────┐          │
 │                           │  Incident Deduplicator          │          │
 │                           │  - Hash MD5 / Cache TTL         │          │
 │                           └────────────────┬────────────────┘          │
 └────────────────────────────────────────────┼───────────────────────────┘
                                              │
                      ┌───────────────────────┴───────────────────────┐
                      ▼                                               ▼
       ┌───────────────────────────────┐             ┌────────────────────────────────┐
       │   Jira Action (Client REST)   │             │  Python Multi-Agent Engine     │
       │   - Projeto OPS               │             │  - Orchestrator Agent          │
       │   - Contrato & Componente     │ <---------> │  - DevOps / SRE Agent          │
       │   - Formatação UTF-8          │             │  - Database Agent              │
       │   - Payload Estruturado Jira  │             │  - QA / Tester & Fullstack     │
       └───────────────────────────────┘             └────────────────────────────────┘
```

<p align="center">
  <img src="imagens/guardian-topology.jpg" alt="Mapa de Topologia e Arquitetura Híbrida do Guardian" width="100%" />
</p>

---

## 3. ⚙️ Supervisor Dinâmico de Targets (Core Engine em Go)

O núcleo do Guardian foi projetado em Go para garantir consumo mínimo de memória (<30MB) e latência de processamento inferior a 5ms.

<p align="center">
  <img src="imagens/guardian-targets.jpg" alt="Painel de Gestão de Targets Multicluster e Docker" width="100%" />
</p>

### 3.1. Abstração de Domínio Unificada
O modelo `domain.Target` unifica clusters e servidores Docker:

```go
type EnvironmentType string

const (
    EnvKubernetes EnvironmentType = "KUBERNETES"
    EnvDocker     EnvironmentType = "DOCKER"
)

type Target struct {
    ID          string          `json:"id"`
    Name        string          `json:"name"`
    Type        EnvironmentType `json:"type"`        // KUBERNETES ou DOCKER
    Context     string          `json:"context"`     // Contexto K8s ou Host Docker
    Scope       string          `json:"scope"`       // Namespace K8s ou Stack Docker
    Status      TargetStatus    `json:"status"`      // ACTIVE, PAUSED ou ERROR
    AutoRemedy  bool            `json:"auto_remedy"` // Governança de sugestões
    Labels      []string        `json:"labels"`
    CreatedAt   time.Time       `json:"created_at"`
    LastCheck   time.Time       `json:"last_check"`
}
```

### 3.2. Ciclo de Vida dos Workers
- **Não-bloqueante**: Cada target ativo possui sua própria goroutine controlada por um `context.WithCancel`.
- **Pausa / Retomada Imediata**: Ao pausar um target na interface web, a chamada HTTP dispara o cancelamento do context do worker correspondente, liberando os watchers de rede e conexões com o Docker Socket ou API do Kubernetes.
- **Persistência Declarativa**: Os targets são armazenados em [`targets.json`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/targets.json), garantindo reinicialização sem perda de configuração.

---

## 4. 🤖 Sistema Multi-Agente em Python (Motor RCA v2.0)

A análise aprofundada de causa raiz (Root Cause Analysis) é conduzida por agentes especializados estruturados em [`guardian_ops/agents/`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/guardian_ops/agents/):

### 4.1. Catálogo de Agentes
1. **Orchestrator Agent (`orchestrator_agent.py`)**:
   - Analisa o evento recebido e classifica a tipologia da falha (Infra, Banco, Aplicação, Pipeline).
   - Delega a investigação para os agentes especialistas e sintetiza o diagnóstico final.
2. **DevOps & SRE Agent (`devops_agent.py`)**:
   - Especializado em contêineres Docker, pods Kubernetes, limites de CPU/Memória, `OOMKilled`, `CrashLoopBackOff`, variáveis de ambiente ausentes e portas conflitantes.
3. **Database Agent (`database_agent.py`)**:
   - Identifica falhas em bancos PostgreSQL, MySQL, Redis, MongoDB e Oracle (ex: conexões esgotadas, deadlocks, erros de autenticação, corrupção de tablespace).
4. **QA & Test Analyst Senior (`qa_agent.py`)**:
   - Detecta falhas de integração, testes quebrando em esteiras de CI/CD, divergência de endpoints REST e respostas HTTP 5xx/4xx.
5. **Fullstack Developer Agent (`fullstack_agent.py`)**:
   - Analisa stack traces em Node.js, Python, PHP, Java, Go e Rust, localizando exceções não tratadas, syntax errors e dependências faltantes.

### 4.2. Heurística de RCA e Estrutura do Chamado
O motor compõe um laudo estruturado:
- **Causa Raiz Provável**: Análise técnica precisa do que desencadeou o erro.
- **Evidências Coletadas**: Últimas 50 linhas de log relevante, código de saída e metadados do container/pod.
- **Plano de Correção Passo a Passo**: Comandos exatos (`docker`, `kubectl`, SQL ou shell) para o time de suporte executar.

---

## 5. 🎫 Automação de Chamados no Jira (Projeto OPS)

A integração oficial do Guardian com o Jira está implementada em Go ([`internal/actions/jira.go`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/internal/actions/jira.go)) e em Python ([`guardian_ops/jira_client.py`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/guardian_ops/jira_client.py)).

<p align="center">
  <img src="imagens/guardian-incidents.jpg" alt="Stream de Incidentes e Chamado Automatizado no Jira" width="100%" />
</p>

### 5.1. Anti-Spam e Deduplicação
Para evitar a criação descontrolada de chamados quando um container reinicia em loop:
1. Um **Hash de Assinatura MD5** é gerado com base em `[Ambiente + Scope + Recurso + TipoDeErro]`.
2. O hash é consultado no cache de memória e no arquivo [`guardian_cache.json`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/guardian_cache.json).
3. Se a falha já foi reportada e o intervalo de TTL (padrão: 3600 segundos) não expirou, a abertura é suprimida e apenas logada como duplicada.

### 5.2. Mapeamento de Campos
- **Projeto**: `OPS`
- **Issue Type**: `Incidente` / `Problema` (ou `Task`)
- **Campos Customizados**: Contrato, Componente e SLA mapeados de acordo com a stack observada.
- **Codificação**: 100% UTF-8 nativo em todos os payloads JSON para evitar caracteres corrompidos.

---

## 6. 🖥️ Interface Web Neo-Glassmorphism

O Guardian embarca uma interface web completa através do mecanismo nativo de compilação do Go (`embed.FS` em [`web/embed.go`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/web/embed.go)):
- **Zero Dependências Externas**: O binário único do Guardian serve a API REST e a interface web na mesma porta (`8080`).
- **Design Neo-Glassmorphism**: Visual escuro ultra-moderno com efeitos de vidro fosco (`backdrop-filter`), gradientes ciberpunk sutis, tipografia Inter e feedback háptico visual.
- **Live Feed de Incidentes**: Monitoramento visual dos eventos de infraestrutura conforme eles ocorrem.
- **Controle Total dos Targets**: Criação de novos targets com seletor de tipo, pausa/retomada em 1 clique e exclusão.

---

## 7. 🔒 Segurança e Governança

1. **Princípio de Somente-Leitura (Read-Only)**:
   - No Kubernetes, o RBAC ([`k8s/rbac-readonly.yaml`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/k8s/rbac-readonly.yaml)) concede apenas `["get", "list", "watch"]` em pods, eventos e logs. Verbos mutáveis (`create`, `delete`, `patch`, `update`) são estritamente bloqueados.
   - No Docker, o socket é montado em modo somente-leitura (`/var/run/docker.sock:ro`).
2. **Segurança de Credenciais**:
   - Tokens de API e senhas do Jira residem no arquivo local [`.env`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/.env) (protegido no `.gitignore`) ou em Kubernetes Secrets.

---

## 8. 📁 Mapa Estrutural do Repositório

```
guardian-infra/
├── cmd/guardian/main.go            # Entrypoint do Core Engine em Go
├── internal/
│   ├── actions/                   # Deduplicação, Jira Client e Notificações
│   ├── api/                       # Rotas REST HTTP e handlers do Dashboard
│   ├── config/                    # Parsing de variáveis de ambiente (.env)
│   ├── domain/                    # Modelos de Target, Incidente e Eventos
│   ├── providers/
│   │   ├── docker/                # Watcher do Docker Events Stream
│   │   └── k8s/                   # Watcher do Kubernetes via client-go Informers
│   ├── storage/                   # Persistência de targets em targets.json
│   └── supervisor/                # Gerenciador de ciclo de vida dos Workers
├── guardian_ops/                  # Motor Multi-Agente em Python (RCA v2.0)
│   ├── agents/                    # Orchestrator, DevOps, Database, QA, Fullstack
│   ├── analyzer.py                # Regras de heurística e classificação
│   ├── docker_scanner.py          # Scanner de containers Docker
│   ├── jira_client.py             # Client Jira REST v2 com custom fields
│   └── k8s_scanner.py             # Scanner do Kubernetes
├── web/
│   ├── embed.go                   # Go embed do frontend compilado no binário
│   └── index.html                 # UI Neo-Glassmorphism do Dashboard
├── k8s/                           # Manifestos Kubernetes (RBAC, Deployment, Secret)
├── Documentações/                 # Base Oficial de Conhecimento e RMTs
├── guardian.sh                    # CLI Shell unificado de inicialização e controle
├── Dockerfile                     # Imagem de produção multi-stage
└── docker-compose.yml             # Execução via Docker Compose local
```
