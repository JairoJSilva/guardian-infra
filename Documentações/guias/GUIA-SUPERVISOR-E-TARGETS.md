# 🎯 Guia do Supervisor e Gestão de Targets — Guardian

O **Supervisor de Targets** é o coração reativo do Guardian. Ele permite gerenciar dinamicamente quais ambientes, clusters e projetos estão sob observação em tempo real, sem necessidade de reiniciar o sistema.

---

## 1. 🧩 O Conceito de Target

Um **Target** representa um alvo de monitoramento contínuo com escopo delimitado:

| Tipo de Target | Context / Host | Scope (Escopo Monitorado) | Unidade de Alerta |
|:---|:---|:---|:---|
| **`KUBERNETES`** | Nome do Contexto (ex: `context-PORTAL-FLOWTI-DEV-01`, `minikube`) | **Namespace** (ex: `producao`, `homolog`, `default`) | Pods com erro (`CrashLoopBackOff`, `OOMKilled`, `Error`) |
| **`DOCKER`** | Host do Docker (ex: `/var/run/docker.sock` ou IP remoto) | **Stack / Projeto Compose** (ex: `flowti-hub`, `dentalis`) | Containers com evento `die` (exitCode != 0) ou `oom` |

---

## 2. 🖥️ Gerenciamento Visual pela Interface Web

A interface Web do Guardian (**`http://localhost:8080`**) oferece controle visual e imediato sobre todos os targets:

### Adicionando um Novo Target:
1. Clique no botão **`+ Novo Target`** no cabeçalho do painel.
2. Selecione o **Tipo de Ambiente**:
   - `☸️ Kubernetes Cluster`
   - `🐳 Docker Stack / Compose`
3. Preencha os campos obrigatórios:
   - **Nome Amigável**: Identificador do target (ex: `Portal Dev Cluster`, `Hub Local Stack`).
   - **Contexto**: Nome do contexto do Kubeconfig ou caminho do socket Docker.
   - **Escopo**: Nome do Namespace do K8s ou do projeto do Docker Compose.
   - **Tags / Labels**: Palavras-chave separadas por vírgula (ex: `core, critical, backend`).
4. Clique em **`Salvar & Iniciar Target`**. O supervisor iniciará a goroutine de monitoramento imediatamente.

### Pausar e Retomar Targets:
- Cada card de target possui um botão de controle:
  - **Pausar (`⏸️`)**: Interrompe imediatamente a leitura de eventos e fecha a conexão com a API do cluster/socket.
  - **Retomar (`▶️`)**: Aloca um novo worker e restabelece a escuta de incidentes.
  - **Excluir (`🗑️`)**: Remove o target permanentemente da lista e do disco.

---

## 3. 🌐 Gerenciamento via API REST

A API do Guardian expõe endpoints completos em formato JSON para automação via CI/CD, scripts bash ou chamadas cURL:

### Listar Todos os Targets:
```bash
curl -s http://localhost:8080/api/targets | jq .
```

### Criar Novo Target via POST:
```bash
curl -X POST http://localhost:8080/api/targets \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Microsserviços de Pagamento",
    "type": "KUBERNETES",
    "context": "context-PORTAL-FLOWTI-DEV-01",
    "scope": "payments",
    "auto_remedy": false,
    "labels": ["financeiro", "checkout", "p1"]
  }'
```

### Pausar ou Ativar um Target (Toggle):
```bash
curl -X POST http://localhost:8080/api/targets/target-docker-local/toggle
```

### Excluir um Target:
```bash
curl -X DELETE http://localhost:8080/api/targets/target-k8s-dev
```

---

## 4. 💾 Estrutura de Persistência (`targets.json`)

Todos os alvos cadastrados são persistidos no arquivo [`targets.json`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/targets.json). Caso o Guardian seja reiniciado ou o container seja recriado, os targets e seus respectivos status anteriores (`ACTIVE` ou `PAUSED`) são restaurados com fidelidade.

```json
[
  {
    "id": "target-k8s-dev-01",
    "name": "Portal Flowti Dev (Kubernetes)",
    "type": "KUBERNETES",
    "context": "context-PORTAL-FLOWTI-DEV-01-ckpsycc2zeq",
    "scope": "flowti-dev",
    "status": "ACTIVE",
    "auto_remedy": false,
    "labels": ["k8s", "dev", "portal"]
  },
  {
    "id": "target-docker-hub",
    "name": "Flowti Hub Local (Docker)",
    "type": "DOCKER",
    "context": "/var/run/docker.sock",
    "scope": "flowti-hub",
    "status": "ACTIVE",
    "auto_remedy": false,
    "labels": ["docker", "local", "hub"]
  }
]
```
