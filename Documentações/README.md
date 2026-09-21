# 📚 Central de Documentações — Guardian / GuardianOps

Bem-vindo à base oficial de conhecimento, especificações técnicas, registros de arquitetura e histórico de modificações do **Guardian** (Plataforma de Observabilidade Ativa, Supervisão de Targets Híbridos e Automação de Incidentes no Jira).

---

## 🧭 Mapa da Central de Documentações

| Documento / Diretório | Descrição |
|:---|:---|
| 📘 [**DOCUMENTACAO_DO_PROJETO.md**](DOCUMENTACAO_DO_PROJETO.md) | **Documentação Técnica Oficial Consolidada**. Detalha arquitetura híbrida (Kubernetes + Docker), Go Core Engine, Supervisor Dinâmico, Motor RCA Multi-Agente em Python, Integração Jira OPS e Governança Read-Only. |
| 📜 [**modificacoes-tecnicas/**](modificacoes-tecnicas/) | **Catálogo Oficial de Registros de Modificações Técnicas (RMT)**. Histórico cronológico de cada intervenção, refatoração e evolução técnica do projeto. |
| 📋 [**modificacoes-tecnicas/TEMPLATE-RMT.md**](modificacoes-tecnicas/TEMPLATE-RMT.md) | **Template Oficial de RMT**. Modelo obrigatório para documentar qualquer alteração técnica no repositório. |
| 🏛️ [**ADR-001-arquitetura-hibrida-go-supervisor-e-agentes.md**](ADR-001-arquitetura-hibrida-go-supervisor-e-agentes.md) | **Registro de Decisão Arquitetural (ADR)** sobre a migração para Core Engine em Go com Supervisor reativo e orquestração de Agentes Especialistas de RCA. |
| 📌 [**issues/**](issues/) | **Especificações Formais de Issues e RFCs**. Demandas, propostas de melhorias e novas funcionalidades planejadas para a plataforma. |
| 🧭 [**guias/**](guias/) | **Manuais Operacionais Práticos**. Passo a passo de execução local, configuração do Jira, gestão de targets e atuação dos agentes autônomos. |

---

## 📂 Guias Práticos Operacionais

Para operadores, desenvolvedores e engenheiros de DevOps/SRE:

1. 🐧 [**Guia de Instalação no Linux (HTML Interativo)**](guias/GUIA-INSTALACAO-LINUX.html) / [**instalacao-guardian.html**](../instalacao-guardian.html): Documentação visual oficial com passo a passo para instalar como aplicativo local nativo no Linux (Desktop Zorin/Ubuntu, pacote `.deb`, CLI `guardian-ctl` e `systemd`).
2. 💻 [**Guia de Execução Local e Docker**](guias/GUIA-EXECUCAO-LOCAL-E-DOCKER.md): Instruções para compilar o binário Go, rodar via `./guardian.sh`, executar com Docker Compose ou utilizar o ambiente virtual Python.
3. 🎫 [**Guia de Integração com o Jira**](guias/GUIA-INTEGRACAO-JIRA.md): Configuração de tokens de API, variáveis `.env`, mapeamento de campos customizados, tags, contratos e anti-duplicação de chamados.
4. 🎯 [**Guia de Supervisor e Gestão de Targets**](guias/GUIA-SUPERVISOR-E-TARGETS.md): Como cadastrar, ativar, pausar e inspecionar targets de Kubernetes (Namespaces) e Docker (Stacks Compose) pela Web UI e API REST.
5. 🤖 [**Guia do Sistema Multi-Agente de RCA**](guias/GUIA-SISTEMA-MULTI-AGENTE-RCA.md): Como funcionam os agentes especializados (DevOps, Database, QA, Fullstack e Orquestrador) na análise heurística de falhas e logs.

---

## ⚡ Regra de Ouro de Governança Contínua

> [!IMPORTANT]
> **SEMPRE que qualquer modificação técnica for efetuada no sistema** (seja no Core em Go, nos Agentes em Python, na interface Web, nos manifestos Kubernetes, no Docker ou na integração Jira):
> 1. Um novo arquivo deve ser criado em [`modificacoes-tecnicas/`](modificacoes-tecnicas/) no formato `RMT-YYYYMMDD-XX-descricao.md`.
> 2. O arquivo modelo [`TEMPLATE-RMT.md`](modificacoes-tecnicas/TEMPLATE-RMT.md) deve ser utilizado como base.
> 3. O índice na tabela de [`modificacoes-tecnicas/README.md`](modificacoes-tecnicas/README.md) deve ser atualizado.

---

## 🚀 Quick Commands de Referência

```bash
# Iniciar o Guardian Híbrido (Go Core Engine + UI na porta 8080)
./guardian.sh start

# Iniciar o Supervisor compilando o Go diretamente
go run ./cmd/guardian

# Iniciar o Observador Local via Docker Compose
docker compose up -d --build

# Executar testes unitários do Core Engine (Go)
go test -v ./...

# Executar testes do motor Python / Jira
python3 test_jira.py
```
