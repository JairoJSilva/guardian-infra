# 🎫 Guia de Integração com o Jira — Guardian

Este guia orienta a configuração completa, homologação e manutenção da integração automatizada entre o **Guardian** e o **Jira Server / Data Center / Cloud** da organização.

---

## 1. ⚙️ Variáveis de Ambiente Obrigatórias (`.env`)

O arquivo [`.env`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/.env) na raiz do repositório armazena os parâmetros de autenticação e roteamento do Jira. 

> [!CAUTION]
> **Nunca commite o arquivo `.env` no Git.** Mantenha-o sempre no `.gitignore`.

### Exemplo de Configuração:
```env
# URL base da sua instância Jira (sem barra final)
JIRA_URL=https://jira.suaempresa.com.br

# E-mail ou usuário de serviço com permissão de criação de issues no projeto OPS
JIRA_EMAIL=servico.guardian@suaempresa.com.br
JIRA_USER=servico.guardian@suaempresa.com.br

# API Token (Jira Cloud) ou Personal Access Token (Jira Server/Data Center)
JIRA_API_TOKEN=seu_token_aqui_gerado_no_jira

# Chave do Projeto Jira onde os incidentes são registrados
JIRA_PROJECT_KEY=OPS

# Tipo de Item no Jira (ex: Incident, Bug, Task ou Incidente)
JIRA_ISSUE_TYPE=Task

# Prioridade padrão para incidentes críticos
JIRA_DEFAULT_PRIORITY=High

# Tempo de resfriamento para deduplicação (segundos)
GUARDIAN_DEDUP_TTL=3600
```

---

## 2. 🛡️ Motor de Deduplicação e Prevenção de Spam

Quando um serviço entra em ciclo contínuo de reinicialização (`CrashLoopBackOff`), o Guardian impede o disparo massivo de chamados repetidos através de uma assinatura única gerada por hash MD5:

$$\text{Hash} = \text{MD5}(\text{Ambiente} + \text{TargetScope} + \text{Recurso} + \text{TipoDeFalha})$$

### Como o Cache Funciona:
1. **Em Memória (Go Engine)**: A struct `IncidentDeduplicator` armazena as chaves ativas e seus timestamps de expiração (`TTL`).
2. **Em Disco (Python Motor)**: O arquivo [`guardian_cache.json`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/guardian_cache.json) persiste os incidentes reportados recentemente.
3. Se o mesmo container falhar novamente dentro da janela de TTL (padrão de 1 hora), o chamado no Jira **NÃO** é reaberto; o evento é apenas contabilizado e registrado no Live Feed da interface web.

---

## 3. 🧪 Como Validar a Conexão com o Jira

O repositório inclui utilitários prontos para testar a comunicação sem depender de uma falha real de container.

### Teste Rápido de Autenticação e Criação:
```bash
python3 test_jira.py
```
O script realizará as seguintes validações:
1. Conexão HTTP com a API REST do Jira (`/rest/api/2/myself`).
2. Validação de permissão no projeto `OPS` (`/rest/api/2/project/OPS`).
3. Criação de um chamado de teste com o resumo: `[TESTE] GuardianOps - Validação de Conexão`.
4. Exibição da chave gerada (ex: `OPS-482`) e link direto para o chamado no Jira.

---

## 4. 📄 Estrutura Padrão do Chamado Gerado

Cada chamado gerado pelo Guardian no Jira segue um formato visual rigoroso, garantindo que a equipe de suporte e plantão tenha todas as respostas imediatas:

```jira
h2. 🛡️ Alerta Operacional: Falha Detectada pelo Guardian

*Ambiente:* {ambiente} ({tipo})
*Alvo / Target:* {target_name}
*Recurso Acometido:* {container_ou_pod}
*Status Detectado:* {CrashLoopBackOff | OOMKilled | Die ExitCode 1}
*Data/Hora do Evento:* {timestamp}

----

h3. 🔍 Diagnóstico e Causa Raiz Provável (RCA)
{diagnostico_do_agente_especialista}

h3. 📋 Evidências Técnicas & Logs
{noformat}
{ultimas_linhas_de_log}
{noformat}

h3. 🛠️ Plano de Intervenção Recomendado
# Verificar se as variáveis de conexão estão corretas: `kubectl describe pod ...`
# Checar o limite de memória alocado para evitar OOM: `docker stats ...`
# Aplicar reinício controlado se necessário.
```

---

## 5. ⚠️ Resolução de Erros na Integração

| Código HTTP | Erro no Log | Solução |
|:---|:---|:---|
| **401 Unauthorized** | `Invalid credentials / API Token expired` | Regenerar o token de API no perfil do usuário no Jira e atualizar a variável `JIRA_API_TOKEN` no `.env`. |
| **400 Bad Request** | `Field 'customfield_XXXXX' is required` | O projeto Jira OPS possui campos customizados obrigatórios na tela de criação. Mapeie-os no payload do cliente Jira em [`internal/actions/jira.go`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/internal/actions/jira.go) ou [`guardian_ops/jira_client.py`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/guardian_ops/jira_client.py). |
| **403 Forbidden** | `User does not have permission to create issues` | Conceder ao usuário de serviço a permissão de "Create Issues" no Permission Scheme do projeto `OPS`. |
| **404 Not Found** | `Project OPS not found` | Verificar se a chave do projeto no `.env` (`JIRA_PROJECT_KEY`) está digitada exatamente em letras maiúsculas. |
