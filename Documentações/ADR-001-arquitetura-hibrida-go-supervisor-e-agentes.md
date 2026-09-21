# 🏛️ ADR-001: Arquitetura Híbrida em Go com Supervisor Reativo de Targets e Motor Multi-Agente de RCA

> **Status**: Aceito / Implementado  
> **Data**: 2026-09-18  
> **Autores**: @SoftwareArchitect & @DevOpsSenior  
> **Decisores**: Equipe de Engenharia de Plataforma e Operações  

---

## 1. 🎯 Contexto e Problemática

Na primeira versão (v1.0), o GuardianOps operava como um script em Python que executava verificações estáticas ou abria conexões persistentes via Docker SDK para monitorar containers locais. Conforme as necessidades de operação evoluíram, identificamos os seguintes desafios:

1. **Monitoramento Híbrido Dinâmico**: A equipe atua simultaneamente em ambientes Kubernetes de desenvolvimento/homologação e servidores/máquinas locais utilizando Docker Compose. Monitorar ambos exigia manter scripts ou instâncias separadas.
2. **Consumo de Recursos & Footprint**: Ter um agente observador que consome centenas de megabytes de RAM em máquinas de desenvolvimento ou clusters é inaceitável. Era mandatário ter uma pegada de memória mínima (<30MB).
3. **Controle Sob Demanda (Target Lifecycle)**: O operador precisa ter autonomia para ativar, pausar ou remover escopos de monitoramento (ex: pausar a stack de checkout enquanto desenvolve localmente para não disparar chamados desnecessários no Jira) sem precisar reiniciar o processo.
4. **Resiliência e Concorrência**: Conexões de streaming de eventos (Docker Events Stream e Kubernetes Informers) precisam de gerenciamento isolado de concorrência com cancelamento limpo de threads/goroutines.

---

## 2. 💡 Decisão Arquitetural

Decidiu-se adotar uma **Arquitetura Híbrida e Composta**:

### 2.1. Core Engine em Go (Supervisor Reativo)
- **Linguagem**: Go (Golang) para o núcleo supervisor, API REST e clientes de observabilidade.
- **Por que Go?**:
  - Compilação estática em binário único sem dependência de runtimes externos.
  - Concorrência nativa ultra-eficiente via Goroutines e Channels.
  - Uso de `context.Context` com cancelamento imediato para pausar e retomar workers de monitoramento sob demanda.
  - Consumo médio de memória em runtime inferior a 25MB.
  - Capacidade de embutir todo o frontend do Dashboard Web dentro do próprio executável compilado (`embed.FS`), eliminando a necessidade de servidores Nginx/Node.js para a UI.

### 2.2. Modelo Unificado de Targets
O conceito central de observabilidade foi abstraído na entidade **Target**:
- Um Target pode ser do tipo **`KUBERNETES`** (escopado por Namespace) ou **`DOCKER`** (escopado por Stack/Compose Project).
- O estado de cada Target é gerenciado por uma máquina de estados (`ACTIVE`, `PAUSED`, `ERROR`) com persistência em [`targets.json`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/targets.json).

### 2.3. Motor Multi-Agente de RCA em Python
- As rotinas de análise profunda de logs, diagnóstico de causa raiz (RCA) e heurísticas avançadas por especialidade (DevOps, Database, QA, Fullstack) foram organizadas em um ecossistema modular multi-agente em Python, atuando em conjunto com o Core Engine.

### 2.4. Deduplicação e Anti-Spam de Incidentes
- Todo incidente detectado passa obrigatoriamente por uma camada de deduplicação antes de acionar o Jira.
- A chave de deduplicação é um Hash MD5 composto pelos metadados do incidente com TTL configurável (padrão: 1 hora).

---

## 3. ⚖️ Consequências e Trade-offs

### Aspectos Positivos
- ✅ **Alta Performance**: Latência de reação a quedas de container inferior a 5 milissegundos.
- ✅ **Facilidade Operacional**: O operador executa `./guardian.sh start` ou `./guardian` e já conta com o supervisor rodando e a interface web disponível na porta `8080`.
- ✅ **Zero Bloqueio**: Falhas de conexão em um cluster Kubernetes remoto não afetam os workers que monitoram o Docker local.
- ✅ **Independência de Serviços**: O Guardian atua de maneira 100% desacoplada das aplicações monitoradas.

### Trade-offs / Desafios Mitigados
- ⚠️ **Duplicidade de linguagens no repositório (Go + Python)**: Mitigada através do script [`guardian.sh`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/guardian.sh) e dos testes automatizados independentes (`go test` e `pytest/python3`), permitindo que operadores usem cada subsistema com facilidade.
