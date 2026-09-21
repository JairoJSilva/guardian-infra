# 📌 Issue #001: Parametrização Multi-Provedor de Chamados via Frontend (Jira, GLPI e Movidesk)

> **Tipo**: 🚀 Feature / RFC  
> **Status**: Proposta Aberta / Backlog  
> **Prioridade**: Alta  
> **Componentes**: `web` (UI), `internal/api` (REST API), `internal/actions` (Providers), `internal/storage` (Config Persistence), `internal/domain` (Models)  
> **Repositório**: `guardian-infra`  
> **Autor**: Antigravity Platform Engineering & SRE Team  

---

## 1. 🎯 Resumo da Proposta (Executive Summary)

Atualmente, o **Guardian** possui acoplamento exclusivo e estático com o **Atlassian Jira**, configurado majoritariamente via variáveis de ambiente (`.env`) e com opções limitadas na interface web (apenas ativação por target e chave do projeto).

Esta issue propõe a criação de uma **Arquitetura de Provedores de Chamados Plugáveis (Ticketing Engine Adapter)** e uma **Interface de Parametrização no Frontend**, permitindo que equipes de SRE, DevOps e Operações configurem, testem e roteiem incidentes para múltiplos sistemas de ITSM e Helpdesk:
1. **Atlassian Jira** (Server / Data Center / Cloud)
2. **GLPI** (REST API v9 / v10)
3. **Movidesk** (Web API REST v1)

A configuração completa de credenciais, tokens, URLs e mapeamento de categorias/filas passará a ser gerenciada dinamicamente via interface web do Guardian, sem necessidade de reinicialização do binário ou edição manual de arquivos de ambiente.

---

## 2. ⚠️ Problema Atual (Current Limitations)

1. **Acoplamento Forte com o Jira**: O núcleo de ações (`internal/actions/jira.go`) e o supervisor (`internal/supervisor/supervisor.go`) estão diretamente atrelados à API do Jira v2.
2. **Impossibilidade de Alternar ou Usar Outras Ferramentas**: Ambientes corporativos que utilizam **GLPI** (comum em suporte interno e infraestrutura on-premise) ou **Movidesk** (comum em atendimento a clientes externos e SaaS) não conseguem integrar o Guardian aos seus fluxos de suporte.
3. **Parametrização Estática**: Alterações de endpoint, usuário, senha ou token exigem edição de arquivos `.env` e reinício da aplicação.
4. **Falta de Roteamento Flexível por Target**: Não é possível definir que incidentes do ambiente de *Homologação* abram chamado no GLPI, enquanto incidentes de *Produção* abram chamado no Jira ou Movidesk.

---

## 3. 💡 Solução Proposta (Proposed Solution)

### 3.1. Arquitetura de Provedores Plugáveis no Backend (Go)

Criar uma interface comum de abstração em `internal/actions/provider.go`:

```go
package actions

import (
	"context"
	"guardian/internal/domain"
)

type ProviderType string

const (
	ProviderJira     ProviderType = "JIRA"
	ProviderGLPI     ProviderType = "GLPI"
	ProviderMovidesk ProviderType = "MOVIDESK"
)

type TicketResult struct {
	Provider    ProviderType `json:"provider"`
	TicketID    string       `json:"ticket_id"`
	TicketURL   string       `json:"ticket_url"`
	RawResponse string       `json:"raw_response,omitempty"`
}

type TicketingProvider interface {
	GetType() ProviderType
	TestConnection(ctx context.Context) error
	CreateIncidentTicket(ctx context.Context, event *domain.IncidentEvent, target *domain.Target) (*TicketResult, error)
}
```

#### Provedores a Implementar:
1. **JiraProvider** (`internal/actions/jira_provider.go`):
   - Refatoração do cliente Jira existente para implementar `TicketingProvider`.
   - Autenticação via Basic Auth ou Personal Access Token (Bearer).
   - Suporte a Wiki Markup e customfields (ex: contrato FLOWTI).
2. **GLPIProvider** (`internal/actions/glpi_provider.go`):
   - GLPI REST API (`/apirest.php/initSession`, `/apirest.php/Ticket`, `/apirest.php/killSession`).
   - Autenticação via `App-Token` e `User-Token` (ou credenciais de usuário).
   - Mapeamento de `itilcategories_id`, `urgency`, `impact`, `type` (Incidente).
   - Envio de descrição formatada em HTML/Markdown com logs do container.
3. **MovideskProvider** (`internal/actions/movidesk_provider.go`):
   - Movidesk Web API v1 (`https://api.movidesk.com/public/v1/tickets?token={token}`).
   - Autenticação via Query Param `token`.
   - Payload JSON com `type` (1 = Interno / 2 = Público), `subject`, `urgency`, `category`, `serviceFirstLevelId`, `actions` (com descrição e logs).
   - Coleta do número do ticket retornado.

---

### 3.2. Nova View de Configurações no Frontend (Web UI)

Adicionar uma nova seção no menu lateral (Sidebar) e view dedicada: **"Integrações ITSM & Helpdesk"**:

```
[Sidebar]
├── Dashboard
├── Targets
├── Live Feed
├── Auditoria & Eventos
└── ⚙️ Integrações ITSM (Novo)
    ├── [Aba Jira]
    ├── [Aba GLPI]
    └── [Aba Movidesk]
```

#### Elementos da Interface para Cada Provedor:
* **Toggle Global de Ativação**: Habilita/desabilita o provedor globalmente.
* **Formulário de Conexão**:
  - **Jira**: URL base, Usuário, Senha/Token, Projeto padrão, Tipo de Issue padrão, Campo de contrato.
  - **GLPI**: URL da API GLPI (`https://glpi.empresa.com/apirest.php`), `App-Token`, `User-Token`, Categoria ITIL padrão, Urgência padrão.
  - **Movidesk**: Token da API Movidesk, Categoria de Serviço padrão, Tipo de Ticket padrão, Urgência padrão.
* **Botão "Testar Conexão"**: Realiza uma chamada de ping/autenticação em tempo real exibindo feedback visual (Verde: Conectado com sucesso / Vermelho: Falha de autenticação ou rede).
* **Ofuscação de Segredos**: Exibição segura de senhas/tokens (tipo password com ícone de alternância para visualização e máscara ao carregar).

---

### 3.3. Roteamento de Chamados por Target

Na criação e edição de Targets (`internal/domain/models.go` e modal `web/index.html`), expandir `ActionConfig`:

```go
type ActionConfig struct {
	// Provedores habilitados para este target específico
	EnabledProviders []ProviderType `json:"enabled_providers"` // ["JIRA", "GLPI", "MOVIDESK"]
	
	// Sobrescritas opcionais por target
	JiraProjectKey   string `json:"jira_project_key,omitempty"`
	GLPICategoryID   int    `json:"glpi_category_id,omitempty"`
	MovideskService  string `json:"movidesk_service,omitempty"`
	
	NotifySlack      bool   `json:"notify_slack"`
	SlackWebhook     string `json:"slack_webhook,omitempty"`
}
```

---

### 3.4. Novas Rotas da API REST (`internal/api`)

| Método | Endpoint | Descrição |
|:---|:---|:---|
| `GET` | `/api/integrations/helpdesk` | Retorna o status e configuração mascarada dos provedores (Jira, GLPI, Movidesk) |
| `POST` | `/api/integrations/helpdesk` | Salva as configurações parametrizadas pelo frontend (com persistência em `helpdesk_config.json`) |
| `POST` | `/api/integrations/helpdesk/test` | Executa teste de conectividade e credenciais com o provedor indicado |
| `GET` | `/api/integrations/helpdesk/meta/{provider}` | Carrega metadados (projetos do Jira, categorias do GLPI, serviços do Movidesk) |

---

## 4. 📋 Critérios de Aceite (Acceptance Criteria)

### Cenário 1: Parametrização e Teste do GLPI
- **Dado** que o operador acesse a aba "Integrações ITSM > GLPI" na interface web,
- **Quando** preencher a URL do GLPI, App-Token e User-Token e clicar em "Testar Conexão",
- **Então** o sistema deve disparar uma requisição de `/initSession` na API do GLPI e exibir mensagem de sucesso ou erro amigável.

### Cenário 2: Parametrização e Teste do Movidesk
- **Dado** que o operador acesse a aba "Integrações ITSM > Movidesk",
- **Quando** informar o API Token do Movidesk e clicar em "Testar Conexão",
- **Então** o sistema deve validar o token via consulta de status na Web API do Movidesk.

### Cenário 3: Abertura Multi-Provedor em Incidente Real
- **Dado** um target configurado para abrir chamados simultaneamente no **Jira** e no **GLPI**,
- **Quando** ocorrer uma falha de container (`CrashLoopBackOff` ou `OOMKilled`),
- **Então** o Guardian deve registrar o incidente, formatar os payloads específicos para cada ferramenta, criar ambos os chamados e exibir os links correspondentes no feed de eventos do dashboard.

### Cenário 4: Persistência e Modo Seguro
- **Dado** que as configurações sejam salvas pela UI,
- **Quando** o serviço for reiniciado,
- **Então** as configurações devem permanecer salvas e nenhum segredo sensível deve ser exposto em logs ou retornado em texto claro desnecessariamente.

---

## 5. 🛠️ Plano de Implementação Sugerido

1. **Sprint 1 (Backend Core & Interfaces)**:
   - Definição da interface `TicketingProvider` e refatoração do Jira.
   - Criação do `GLPIProvider` e `MovideskProvider`.
   - Armazenamento em `storage/helpdesk_storage.go`.
2. **Sprint 2 (API REST & Testes de Integração)**:
   - Endpoints `/api/integrations/helpdesk` e `/api/integrations/helpdesk/test`.
   - Testes unitários para serialização de payloads do GLPI e Movidesk.
3. **Sprint 3 (Frontend Web UI)**:
   - View de Configurações ITSM no arquivo `web/index.html`.
   - Componentes de input com máscara, abas para Jira/GLPI/Movidesk e botão de teste com feedback em tempo real.
   - Atualização do modal de Targets com seleção múltipla de provedores.
4. **Sprint 4 (E2E & Documentação)**:
   - Validação com instâncias de teste do Jira, GLPI e Movidesk.
   - Atualização da documentação técnica e guias de uso.
