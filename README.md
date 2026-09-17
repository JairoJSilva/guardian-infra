# 🛡️ GuardianOps (v1.0) - Observador & Analisador de Falhas de Aplicações e Pipelines

O **GuardianOps** é um componente inteligente desenhado para monitorar e capturar falhas em aplicações (Pods do Kubernetes) e esteiras de CI/CD (Pipelines GitLab/GitHub), analisando a causa-raiz (RCA) e abrindo chamados técnicos detalhados e padronizados no Jira.

---

## 🔒 Princípio de Segurança Mandatório (v1.0)

> [!IMPORTANT]
> **Modo Estritamente Somente-Leitura (Read-Only):**  
> Nesta primeira versão, o GuardianOps **NÃO executa ações corretivas em hipótese alguma**. Ele atua exclusivamente como diagnosticador, sintetizador de causa-raiz e emissor de chamados com sugestão de procedimento passo a passo para o operador humano.

---

## ✨ Principais Capacidades

1. **Diagnóstico Automatizado de Workloads K8s**:
   - `CrashLoopBackOff` (Identificação de erros de inicialização, conexão de banco, secrets faltantes).
   - `OOMKilled` (Detecção de estouro de limites de memória `limits.memory` e exit code 137).
   - `ImagePullBackOff` / `ErrImagePull` (Detecção de tags incorretas, registros inacessíveis ou segredos de pull).
   - `FailedLivenessProbe` / `FailedReadinessProbe` (Falhas de probes de saúde HTTP/TCP).
2. **Diagnóstico de Falhas em Pipelines CI/CD**:
   - Recebe webhooks do GitLab CI / GitHub Actions com status `failed`.
   - Analisa o estágio que falhou (`build`, `test`, `docker-build`, `deploy`).
   - Extrai trechos de logs do runner e sintetiza a causa do bloqueio.
3. **Padrão Institucional de Chamados no Jira**:
   - Resumo claro e objetivo do incidente.
   - Diagnóstico & Análise de Causa Raiz (RCA).
   - Tabela de ativos impactados (Namespace, Pod, Container, Status).
   - Procedimento de correção passo a passo com comandos recomendados (`kubectl describe`, `kubectl logs`, etc.).
   - Critérios de aceite e plano de rollback.
   - Vínculo automático a chamados pais/referência (ex: `OPS-236`).
4. **Integridade de Codificação UTF-8**:
   - Todo tráfego HTTP é transmitido com `charset=utf-8` e `ensure_ascii=False`, preservando 100% da acentuação em português (`á`, `é`, `ç`, `ã`) e emojis (`🛡️`, `🤖`, `📌`, `⚠️`) sem quebrar a formatação no Jira Server/Data Center.
5. **Deduplicação & Anti-Spam de Chamados (Cooldown)**:
   - Se um pod estiver em loop de reinicialização (`CrashLoopBackOff`), o GuardianOps gera uma assinatura (_fingerprint_) única e aplica um cooldown configurável (padrão: 60 minutos), evitando a criação de dezenas de chamados repetidos para o mesmo incidente.

---

## 🚀 Como Executar e Testar Localmente (Sem subir no cluster)

### 1. Pré-requisitos e Ativação do Ambiente
```bash
# Entrar no diretório
cd /home/jairosjunior/Documentos/Jairo/automação-jira

# O ambiente virtual já está configurado em .venv
source .venv/bin/activate
```

### 2. Testar Conexão com o Jira
Valida suas credenciais e endpoint do Jira configurados no `.env`:
```bash
python3 main.py test-jira
```

### 3. Simulação de Falha de Pod K8s (Dry-Run ou Real)
Gera a análise e formatação completa sem criar chamado no Jira:
```bash
# Simula CrashLoopBackOff (Modo Dry-Run)
python3 main.py simulate-pod --dry-run

# Simula OOMKilled (Modo Dry-Run)
python3 main.py simulate-pod --type OOMKilled --exit-code 137 --dry-run
```

Para enviar o chamado de verdade ao Jira vinculado ao pai:
```bash
python3 main.py simulate-pod --parent OPS-236
```

### 4. Simulação de Falha de Pipeline CI/CD
```bash
# Simula falha no estágio docker-build (Modo Dry-Run)
python3 main.py simulate-pipeline --dry-run

# Envia chamado real para o Jira
python3 main.py simulate-pipeline --project "flowti/portal-paciente" --parent OPS-236
```

### 5. Iniciar o Webhook Listener para Pipelines
Inicia o servidor HTTP para receber webhooks reais do GitLab CI:
```bash
python3 main.py webhook --port 8080 --dry-run
```
Endpoint configurável no GitLab: `http://<IP_DO_GUARDIAN>:8080/webhook/gitlab`

### 6. Executar Bateria de Testes Automatizada
```bash
python3 test_guardian.py
```

---

## 📦 Estrutura do Projeto

```
automação-jira/
├── guardian_ops/
│   ├── __init__.py           # Versão e metadados
│   ├── config.py             # Variáveis de ambiente e segurança
│   ├── models.py             # Modelos de dados (IncidentEvent, AnalysisResult)
│   ├── analyzer.py           # Motor de RCA e sugestões analíticas
│   ├── jira_client.py        # Cliente Jira com UTF-8 estrito e Anti-Spam
│   ├── templates.py          # Gerador de marcação Jira padrão
│   ├── k8s_scanner.py        # Scanner e gerador de mocks de Pods K8s
│   └── pipeline_listener.py  # Webhook server e mocks de pipelines
├── k8s/                      # Manifestos para implantação futura no Cluster
│   ├── rbac-readonly.yaml    # ServiceAccount e ClusterRole estritamente Read-Only
│   ├── configmap-secret.yaml # ConfigMap e Secret com credenciais
│   └── deployment.yaml       # Deployment (2 réplicas) + Service ClusterIP
├── Dockerfile                # Imagem container não-root otimizada
├── main.py                   # CLI principal de operação e simulação
├── test_guardian.py          # Bateria de testes unitários e de integração
└── requirements.txt          # Dependências mínimas (requests, python-dotenv)
```

---

## ☸️ Implantação Futura no Cluster Kubernetes

Quando a equipe decidir implantar o GuardianOps dentro do cluster Kubernetes, os manifestos em `k8s/` já estão 100% prontos seguindo as regras de governança:
- **Deployment Declarativo** com 2 réplicas e estratégia `RollingUpdate` (Zero Downtime).
- **RBAC Estritamente Somente-Leitura**: Verbos apenas `["get", "list", "watch"]` em pods, logs e eventos. Nenhuma permissão de modificação (`create`, `update`, `delete`, `patch`).
- **Segurança Não-Root**: Execução sob UID `10001` (`guardian`) com `runAsNonRoot: true`.
- **Health Probes**: `livenessProbe` e `readinessProbe` em `/healthz`.
