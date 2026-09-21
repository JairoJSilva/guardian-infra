# 📝 RMT-20260918-01: Sistema Multi-Agente v2.0 e Modo Observador Docker Local

> **RMT (Registro de Modificação Técnica)**  
> **Status**: Aplicada  
> **Data**: 2026-09-18  
> **Autor / Agente Responsável**: @DevOpsSenior & @FullstackDeveloper  
> **Tipo de Mudança**: Feature / Architecture / Multi-Agent  
> **Versão Afetada**: v2.0.0  

---

## 1. 🎯 Contexto e Motivação
A necessidade de rodar o GuardianOps localmente monitorando contêineres Docker da máquina de desenvolvimento e servidores locais demandou:
1. Um observador baseado no socket local (`/var/run/docker.sock`).
2. Especialização da inteligência de diagnóstico através de múltiplos agentes atuando em paralelo (DevOps, Database, QA, Fullstack).

---

## 2. 📁 Arquivos e Componentes Afetados

| Arquivo / Caminho | Tipo de Alteração | Descrição Resumida |
|:---|:---|:---|
| `guardian_ops/docker_scanner.py` | Criado | Leitor de eventos e estados de contêineres Docker locais |
| `guardian_ops/agents/base_agent.py` | Criado | Classe abstrata para criação de agentes especialistas |
| `guardian_ops/agents/orchestrator_agent.py` | Criado | Roteador e agregador central de diagnósticos |
| `guardian_ops/agents/devops_agent.py` | Criado | Especialista em contêineres, recursos e runtime |
| `guardian_ops/agents/database_agent.py` | Criado | Especialista em pools e falhas de bancos SQL e NoSQL |
| `guardian_ops/agents/qa_agent.py` | Criado | Especialista em testes e quebras de contrato de APIs |
| `guardian_ops/agents/fullstack_agent.py` | Criado | Especialista em exceções em aplicações web |
| `docker-compose.yml` | Criado | Definição do container observador local com socket montado |
| `test_guardian.py` | Criado | Teste ponta a ponta simulando queda de container |

---

## 3. ⚙️ Detalhamento Técnico das Modificações

### 3.1. Arquitetura Multi-Agente
- Criação do pipeline de especialização onde o `OrchestratorAgent` avalia logs preliminares e despacha para agentes temáticos, gerando diagnósticos muito mais assertivos e precisos.

### 3.2. Provedor Docker Local
- Implementação da escuta contínua de eventos Docker (`container die`, `oom`, etc.) via SDK do Docker com montagem de volume somente-leitura em `/var/run/docker.sock:ro`.

---

## 4. ⚠️ Impactos, Compatibilidade e Dependências
- **Dependências**: Adicionada biblioteca `docker` ao `requirements.txt`.
- **Permissões**: O usuário que executa precisa ter acesso de leitura ao socket `/var/run/docker.sock`.

---

## 5. 🧪 Testes e Validação
- [x] Teste de parada de container de banco de dados e disparo de diagnóstico validado.
- [x] Execução de suíte de testes com `test_guardian.py`.
