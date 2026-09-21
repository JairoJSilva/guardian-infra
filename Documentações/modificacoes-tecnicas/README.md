# 📜 Catálogo de Registros de Modificações Técnicas (RMT)

Este diretório contém o histórico oficial de todas as alterações, refatorações e evoluções técnicas implementadas no **Guardian**.

---

## 🎯 Regra Obrigatória de Governança

> [!IMPORTANT]
> **SEMPRE que for realizada qualquer modificação técnica no projeto** (seja no Core Engine em Go, no motor Multi-Agente em Python, na interface Web Neo-Glassmorphism, no Docker, nos manifestos Kubernetes ou na integração Jira):
> **Um novo arquivo de registro deve ser criado neste diretório**.

### Convenção de Nomenclatura:
Os arquivos devem seguir rigorosamente o padrão:
```
RMT-YYYYMMDD-XX-descricao-curta.md
```
- **`YYYYMMDD`**: Ano, mês e dia da alteração.
- **`XX`**: Sequencial numérico do dia (`01`, `02`, etc.).
- **`descricao-curta`**: Palavras-chave em minúsculas separadas por hífen indicando o objetivo da alteração.

### Como registrar uma nova alteração:
1. Copie o arquivo modelo [`TEMPLATE-RMT.md`](TEMPLATE-RMT.md).
2. Salve com o novo nome seguindo a convenção acima.
3. Preencha detalhadamente as seções de contexto, arquivos alterados, aspectos técnicos, impactos e validação/testes.
4. Adicione a nova entrada na tabela cronológica abaixo.

---

## 📋 Tabela Cronológica de Registros

| ID | Data | Título / Descrição | Tipo | Autor / Agente | Status |
|:---|:---|:---|:---|:---|:---|
| [RMT-20260917-01](RMT-20260917-01-criacao-guardian-ops-motor-rca-e-jira.md) | 2026-09-17 | Criação do GuardianOps v1.0, Motor RCA e Automação de Chamados no Jira OPS | Feature / Architecture | @DevOpsSenior & @SoftwareArchitect | ✅ Aplicada |
| [RMT-20260918-01](RMT-20260918-01-sistema-multi-agente-e-modo-observador-docker.md) | 2026-09-18 | Sistema Multi-Agente v2.0 (Orquestrador, DevOps, DB, QA, Fullstack) e Observador Docker Local | Feature / Multi-Agent / Observabilidade | @DevOpsSenior & @FullstackDeveloper | ✅ Aplicada |
| [RMT-20260918-02](RMT-20260918-02-core-engine-go-supervisor-hibrido-e-ui.md) | 2026-09-18 | Implementação do Core Engine Híbrido em Go com Supervisor Dinâmico e Dashboard Neo-Glassmorphism | Architecture / Performance / UI-UX | @SoftwareArchitect & @UIUXDesigner | ✅ Aplicada |
| [RMT-20260918-03](RMT-20260918-03-central-oficial-de-documentacoes-e-governanca.md) | 2026-09-18 | Estruturação da Central Oficial de Documentações (`Documentações/`), Guias Operacionais e Padrão RMT | Governance / Docs | Antigravity AI / @PlatformEngineer | ✅ Aplicada |
