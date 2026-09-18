# 🛡️ GuardianOps (v2.0) — Sistema Multi-Agente de Observabilidade & RCA

O **GuardianOps v2.0** é uma plataforma inteligente e autônoma de observabilidade, diagnóstico de causa-raiz (RCA) e abertura automatizada de chamados técnicos no **Jira**.

Executando localmente como um container Docker de baixo consumo ou como daemon no Kubernetes, o GuardianOps monitora containers locais, pods de clusters remotos e esteiras de CI/CD. Quando uma falha é detectada, o **OrchestratorBot** aciona um esquadrão de **agentes especialistas** para diagnosticar o incidente, sintetizar o impacto e abrir um chamado com procedimentos detalhados de correção.

---

## 🔒 Princípio de Segurança Mandatório (Read-Only)

> [!IMPORTANT]
> **Modo Estritamente Somente-Leitura (Read-Only):**  
> O GuardianOps opera 100% em modo somente-leitura. Ele **NÃO executa ações destrutivas, correções cegas ou reinicializações não supervisionadas** nos ambientes monitorados. Sua função é diagnosticar com precisão técnica cirúrgica, classificar a severidade e fornecer o procedimento passo a passo pronto para o operador humano validar e aplicar.

---

## 🤖 Esquadrão de Agentes Especialistas (v2.0)

O GuardianOps adota uma arquitetura multi-agente determinística (sem custos de API ou dependência de serviços externos). Cada agente possui domínio aprofundado sobre sua camada:

| Agente | ID | Especialidade & Cobertura |
| :--- | :--- | :--- |
| **🌐 DevOps & Cloud Native** | `agent-devops-cloudnative` | **Kubernetes** (OOMKilled, CrashLoopBackOff, ImagePull, Probes, PVC Pending), **Docker** (saúde, parada de containers, socket), **IaC** (Terraform, OpenTofu, Ansible) e **Pipelines CI/CD** (GitLab CI, GitHub Actions). |
| **💾 Database & Data Systems** | `agent-database-specialist` | **Bancos Relacionais e NoSQL** (MySQL, PostgreSQL, MongoDB, Redis, Elasticsearch). Identifica exaustão de conexões, deadlocks, transações longas, queries lentas e falhas em migrações (Flyway, Alembic, Prisma). |
| **🧪 QA Senior & Automation** | `agent-qa-senior` | **Garantia de Qualidade & Testes**: Regressões críticas, quebra de contratos de API (REST/GraphQL), testes intermitentes (*flaky tests*), cobertura de testes e templates de cenários BDD/Gherkin prontos para automação. |
| **⚡ FullStack Developer** | `agent-fullstack-developer` | **Camada de Aplicação & Código**: Exceções não tratadas (NullPointer, TypeError), vazamento de memória (*heap leak*), problemas de CORS, falhas assíncronas (unhandled promises), segurança OWASP (SQLi/XSS) e erros de autenticação (401/403). |
| **🎯 Orchestrator Bot** | `agent-orchestrator` | **Orquestrador Central**: Analisa os sintomas do incidente, roteia para os especialistas relevantes (acionando diagnósticos combinados quando necessário), consolida o relatório unificado e envia para o Jira. |

---

## 🏗️ Arquitetura de Execução Local (Container Docker)

A arquitetura recomendada para desenvolvimento e sustentação operacional consiste em rodar o GuardianOps localmente na sua máquina via **Docker Compose**:

```
 ┌─────────────────────────────────────────────────────────┐
 │                   MÁQUINA LOCAL (HOST)                  │
 │                                                         │
 │   ┌────────────────┐  ┌────────────────┐                │
 │   │   flowti-app   │  │  flowti-mysql  │  ...containers │
 │   └────────┬───────┘  └────────┬───────┘                │
 │            │                   │                        │
 │            └─────────┬─────────┘                        │
 │                      ▼                                  │
 │             /var/run/docker.sock (:ro)                  │
 │                      │                                  │
 │   ┌──────────────────┴──────────────────────────────┐   │
 │   │  Container: guardianops_watch (v2.0)            │   │
 │   │                                                 │   │
 │   │  ┌───────────────────────────────────────────┐  │   │
 │   │  │ DockerScanner (Varredura Contínua)        │  │   │
 │   │  └─────────────────────┬─────────────────────┘  │   │
 │   │                        ▼                        │   │
 │   │  ┌───────────────────────────────────────────┐  │   │
 │   │  │ OrchestratorBotAgent                     │  │   │
 │   │  │   ├── DevOpsAgent                         │  │   │
 │   │  │   ├── DatabaseAgent                       │  │   │
 │   │  │   ├── FullStackAgent                      │  │   │
 │   │  │   └── QAAgent                             │  │   │
 │   │  └─────────────────────┬─────────────────────┘  │   │
 │   │                        ▼                        │   │
 │   │  ┌───────────────────────────────────────────┐  │   │
 │   │  │ JiraClient (UTF-8 + Deduplicação/Cooldown)│  │   │
 │   │  └─────────────────────┬─────────────────────┘  │   │
 │   │                        │                        │   │
 │   │  ~/.kube (:ro)         │                        │   │
 │   └────────┬───────────────┼────────────────────────┘   │
 └────────────┼───────────────┼────────────────────────────┘
              │               │ HTTPS
              ▼               ▼
     Clusters K8s Remotos    Jira Data Center (Projeto OPS)
```

### Por que esta é a melhor opção?
1. **Acesso Nativo e Seguro ao Docker**: O container acessa o `/var/run/docker.sock` em modo somente-leitura. Qualquer falha ou parada inesperada de container é detectada em segundos.
2. **Ponte com Clusters K8s Remotos**: Montando o volume do `${HOME}/.kube`, o GuardianOps inspeciona pods de clusters remotos usando as mesmas permissões do seu `kubectl` local.
3. **Persistência Segura de Cache**: O cache de deduplicação é persistido em volume Docker (`guardian_cache`), evitando chamados repetidos para o mesmo incidente.
4. **Sem Conflito de Portas**: O webhook listener roda na porta interna `8080` e é exposto na porta `8088` do host, deixando a porta `8080` livre para suas aplicações locais (`flowti-app`).

---

## 🚀 Como Executar

### 1. Configurar Variáveis de Ambiente
Crie ou verifique o arquivo `.env` na raiz do projeto:
```env
JIRA_BASE_URL=https://jira.mv.com.br
JIRA_USER=seu.usuario
JIRA_PASSWORD=sua_senha_ou_token
```

### 2. Iniciar o Observador Contínuo (Modo Recomendado)
```bash
# Construir a imagem com o esquadrão de agentes
docker build -t guardianops:v2.0 .

# Iniciar em segundo plano
docker compose up -d guardian

# Visualizar logs em tempo real
docker compose logs -f guardian
```

### 3. Testar a Detecção Automática
Em outro terminal, pare qualquer container monitorado:
```bash
# Exemplo: simulando queda de container
docker stop flowti-phpmyadmin
```
Em até 30 segundos, o GuardianOps identificará o evento `ContainerStopped`, convocará os agentes especialistas, gerará a análise e abrirá o chamado automaticamente no Jira!

Para restabelecer o container de teste:
```bash
docker start flowti-phpmyadmin
```

---

## 🛠️ Modos de Uso e Comandos CLI

O GuardianOps oferece ferramentas para simulação, testes e chamadas pontuais sob demanda:

### Listar os Agentes Ativos e Suas Capacidades
```bash
# Via container CLI:
docker compose run --rm guardian-cli list-agents

# Ou diretamente no Python local:
python3 main.py list-agents
```

### Orquestrar Diagnóstico Manual Multi-Agente
```bash
# Diagnóstico de banco com agentes DevOps + Database
docker compose run --rm guardian-cli orchestrate \
  --source DOCKER_CONTAINER \
  --failure-type ConnectionRefused \
  --component flowti-mysql \
  --container mysql \
  --logs "ERROR 2002 (HY000): Can't connect to local MySQL server through socket" \
  --dry-run
```

### Testar Conexão com o Jira
```bash
docker compose run --rm guardian-cli test-jira
```

### Script Utilitário Interativo (`guardian.sh`)
Para facilitar a operação no dia a dia, use o script interativo com menu:
```bash
./guardian.sh
```

---

## 📦 Estrutura do Projeto

```
guardian-infra/
├── guardian_ops/
│   ├── agents/                   # 🤖 Esquadrão Multi-Agente (v2.0)
│   │   ├── __init__.py
│   │   ├── base_agent.py         # Definições base, enums e formatador Jira Wiki
│   │   ├── devops_agent.py       # Agente DevOps, K8s, Docker e IaC
│   │   ├── database_agent.py     # Agente Especialista em Bancos de Dados
│   │   ├── qa_agent.py           # Agente QA Sênior e Automação
│   │   ├── fullstack_agent.py    # Agente Desenvolvedor FullStack
│   │   └── orchestrator_agent.py # Orquestrador Multi-Agente
│   ├── config.py                 # Configurações, segurança e resolução de contrato
│   ├── models.py                 # Modelos de dados de eventos e diagnósticos
│   ├── jira_client.py            # Cliente Jira com UTF-8 estrito e Anti-Spam
│   ├── templates.py              # Templates de formatação de tickets Jira
│   ├── docker_scanner.py         # Scanner de containers Docker via unix socket
│   ├── k8s_scanner.py            # Scanner de Pods Kubernetes
│   └── pipeline_listener.py      # Servidor HTTP para webhooks de CI/CD
├── k8s/                          # Manifestos declarativos para deploy em cluster
│   ├── rbac-readonly.yaml        # ClusterRole e ServiceAccount estritamente Read-Only
│   ├── configmap-secret.yaml     # ConfigMap e Secret
│   └── deployment.yaml           # Deployment com health probes
├── Dockerfile                    # Imagem Python 3.12-slim com usuário guardian
├── docker-compose.yml            # Orquestração local (serviços watch e cli)
├── guardian.sh                   # Script interativo de atalhos operacionais
├── requirements.txt              # Dependências (requests, python-dotenv)
└── README.md                     # Documentação oficial
```

---

## 🛡️ Regras de Formatação do Jira e Resolução de Contratos

- **Contrato Inteligente**: O campo `customfield_30118` é resolvido automaticamente. Serviços com termos como `flowti`, `infra`, `kube`, `interno`, `sistema` são mapeados diretamente para o contrato corporativo **`INTERNO`** (ID: 22514).
- **Codificação UTF-8 Estrita**: Todas as requisições para a API do Jira usam codificação UTF-8 limpa, preservando a acentuação em português e ícones sem gerar quebras de caracteres (`¿`).
- **Janela de Cooldown Anti-Spam**: Se o mesmo serviço apresentar anomalias repetidas no mesmo ciclo, um cooldown de 60 minutos (configurável) impede a abertura duplicada de tickets.
