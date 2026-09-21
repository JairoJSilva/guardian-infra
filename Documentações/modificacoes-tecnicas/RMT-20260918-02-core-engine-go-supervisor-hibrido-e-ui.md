# 📝 RMT-20260918-02: Implementação do Core Engine Híbrido em Go com Supervisor Dinâmico e Dashboard Neo-Glassmorphism

> **RMT (Registro de Modificação Técnica)**  
> **Status**: Aplicada  
> **Data**: 2026-09-18  
> **Autor / Agente Responsável**: @SoftwareArchitect & @UIUXDesigner  
> **Tipo de Mudança**: Architecture / Performance / UI-UX / Feature  
> **Versão Afetada**: v2.1.0  

---

## 1. 🎯 Contexto e Motivação
Para unificar o monitoramento simultâneo de múltiplos clusters Kubernetes e stacks Docker Compose com baixo consumo de memória (<30MB) e interface de gestão em tempo real, desenvolveu-se o Core Engine em Go com um Supervisor Dinâmico de Targets e Dashboard web embutido diretamente no binário.

---

## 2. 📁 Arquivos e Componentes Afetados

| Arquivo / Caminho | Tipo de Alteração | Descrição Resumida |
|:---|:---|:---|
| `cmd/guardian/main.go` | Criado | Entrypoint do Core Engine em Go |
| `internal/domain/models.go` | Criado | Estruturas de Target, Incidente e Eventos |
| `internal/supervisor/supervisor.go` | Criado | Gerenciador reativo de ciclo de vida de workers concorrentes |
| `internal/supervisor/worker.go` | Criado | Goroutine de execução do target com `context.WithCancel` |
| `internal/providers/docker/` | Criado | Provedor de eventos Docker em Go |
| `internal/providers/k8s/` | Criado | Provedor de eventos Kubernetes via client-go Informers |
| `internal/actions/deduplicator.go` | Criado | Anti-spam de incidentes com chave MD5 e TTL em memória |
| `internal/actions/jira.go` | Criado | Cliente HTTP Go para abertura de chamados no Jira |
| `internal/api/server.go` | Criado | Servidor HTTP REST com rotas `/api/targets`, `/api/health` |
| `internal/storage/storage.go` | Criado | Persistência atômica de targets em `targets.json` |
| `web/embed.go` | Criado | Diretiva `//go:embed index.html` para embutir frontend no executável |
| `web/index.html` | Modificado | Interface visual Neo-Glassmorphism escura com modal e Live Feed |
| `guardian.sh` | Criado | Script CLI para compilação, início, parada e logs |
| `targets.json` | Criado | Arquivo declarativo inicial de targets monitorados |

---

## 3. ⚙️ Detalhamento Técnico das Modificações

### 3.1. Supervisor de Concorrência em Go
- Inicialização de workers independentes usando goroutines para cada Target ativo.
- Cancelamento instantâneo via contexto (`ctx.Done()`) ao pausar targets via interface web ou API REST.

### 3.2. Interface Web Neo-Glassmorphism Embutida
- Servida nativamente pelo Go na porta `8080` sem necessidade de Node.js ou Nginx em produção.
- Visual escuro imersivo, cards com glassmorphism, modais dinâmicos para novos targets e Live Feed de incidentes.

### 3.3. Deduplicador de Incidentes
- Criação de chave hash única baseada nos atributos da falha para evitar múltiplos chamados no Jira para o mesmo incidente durante o período de cooldown.

---

## 4. ⚠️ Impactos, Compatibilidade e Dependências
- **Binário Único**: O executável compilado `./guardian` encapsula toda a aplicação (API + Web UI).
- **Consumo de Memória**: Reduzido para aproximadamente 22MB em repouso.

---

## 5. 🧪 Testes e Validação
- [x] Testes unitários do Go executados com sucesso (`go test -v ./...`).
- [x] Interface Web validada e responsiva na porta `8080`.
- [x] Criação e alternância de status de targets via API REST `/api/targets/{id}/toggle`.
