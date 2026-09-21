# 📝 RMT-YYYYMMDD-XX: [Título Resumido da Modificação Técnica]

> **RMT (Registro de Modificação Técnica)**  
> **Status**: [Proposta | Aplicada | Em Homologação | Revertida]  
> **Data**: YYYY-MM-DD  
> **Autor / Agente Responsável**: @NomeOuAgente  
> **Tipo de Mudança**: [Feature | Bugfix | Refactor | Architecture | Infra/DevOps | Security | Observabilidade]  
> **Versão Afetada**: vX.Y.Z  

---

## 1. 🎯 Contexto e Motivação
Descreva detalhadamente o motivo pelo qual esta modificação foi realizada. Qual problema operacional ou de observabilidade foi resolvido, qual funcionalidade foi adicionada ou qual melhoria técnica foi implementada?

---

## 2. 📁 Arquivos e Componentes Afetados

| Arquivo / Caminho | Tipo de Alteração | Descrição Resumida |
|:---|:---|:---|
| `caminho/do/arquivo.go` | [Criado / Modificado / Removido] | Resumo da intervenção técnica |

---

## 3. ⚙️ Detalhamento Técnico das Modificações

### 3.1. Core Engine em Go (Supervisor / Providers / API)
- Mudanças no supervisor de targets, workers concorrentes, listeners de eventos ou endpoints HTTP.

### 3.2. Motor de Agentes em Python (RCA / Heurísticas)
- Ajustes em agentes especialistas, scanners de log, expressões regulares ou regras de classificação.

### 3.3. Ações e Integrações (Jira / Anti-Spam / Notificadores)
- Alterações no cliente Jira, formato de payload, campos obrigatórios ou política de deduplicação (TTL/MD5).

### 3.4. Interface Web (Frontend Neo-Glassmorphism)
- Modificações na UI embutida, componentes visuais, live feed ou estilos CSS.

### 3.5. Infraestrutura & DevOps (Docker / K8s / Scripts)
- Mudanças em Dockerfile, compose, manifestos Kubernetes (`k8s/`) ou no script [`guardian.sh`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/guardian.sh).

---

## 4. ⚠️ Impactos, Compatibilidade e Dependências
- **Breaking Changes?** [Sim / Não] (Se sim, descrever como mitigar).
- **Variáveis de Ambiente**: Novas variáveis adicionadas ao `.env`?
- **Compatibilidade com versões anteriores**: Riscos de quebra identificados.
- **Permissões**: Exige novas permissões no socket Docker ou no RBAC do Kubernetes?

---

## 5. 🧪 Testes e Validação
Descreva o que foi testado e os resultados obtidos:
- [ ] Testes unitários do Go executados (`go test -v ./...`)
- [ ] Testes do cliente Jira executados (`python3 test_jira.py`)
- [ ] Validação visual do Dashboard Web na porta `8080`
- [ ] Validação de captura de evento de queda de container

---

## 6. 📌 Referências e Links Relacionados
- ADR relacionada: [ADR-001](../ADR-001-arquitetura-hibrida-go-supervisor-e-agentes.md)
- Documentação Técnica: [DOCUMENTACAO_DO_PROJETO.md](../DOCUMENTACAO_DO_PROJETO.md)
- Commit / Tag Git: `hash-ou-tag`
