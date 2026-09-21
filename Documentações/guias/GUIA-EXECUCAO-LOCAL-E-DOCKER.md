# 💻 Guia de Execução Local e Docker — Guardian

Este guia detalha todos os métodos homologados para iniciar, operar e testar o **Guardian** em máquinas de desenvolvimento e servidores locais.

---

## 1. 📋 Pré-requisitos do Ambiente

| Recurso | Versão Recomendada | Obrigatório? | Observação |
|:---|:---|:---|:---|
| **Docker Engine** | 24.0+ | Sim | Necessário acesso a `/var/run/docker.sock` para o Docker Provider |
| **Docker Compose** | v2.20+ | Opcional | Recomendado para execução em container isolado |
| **Go (Golang)** | 1.21+ | Sim (p/ build local) | Necessário para compilar o Core Engine |
| **Python** | 3.10+ | Opcional | Necessário para execução avulsa do motor de agentes (`main.py`) |
| **Kubectl / Kubeconfig** | 1.28+ | Opcional | Necessário se for monitorar targets Kubernetes |

---

## 2. 🚀 Métodos de Execução

### Método 1: Utilizando o Script Oficial `guardian.sh` (Recomendado)
O script [`guardian.sh`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/guardian.sh) automatiza a compilação, checagem de porta e execução em background do binário compilado.

```bash
# Iniciar o Guardian em segundo plano (porta 8080)
./guardian.sh start

# Verificar status da execução e PID do processo
./guardian.sh status

# Visualizar logs em tempo real
./guardian.sh logs

# Reiniciar o serviço
./guardian.sh restart

# Parar o serviço com encerramento gracioso
./guardian.sh stop
```

Após iniciar, abra o navegador em: **`http://localhost:8080`**.

---

### Método 2: Execução Direta em Go (Ambiente Nativo)

Você pode compilar e executar o código Go diretamente do terminal:

```bash
# Executar em modo desenvolvimento (compilação sob demanda)
go run ./cmd/guardian

# Ou compilar o binário estático otimizado
go build -ldflags="-s -w" -o guardian ./cmd/guardian
./guardian
```

---

### Método 3: Execução via Docker Compose (Container Isolado)

Se preferir manter o Guardian rodando dentro de um container Docker isolado:

```bash
# Subir a stack com build automático
docker compose up -d --build

# Acompanhar os logs do container
docker compose logs -f guardian

# Parar a stack
docker compose down
```

> [!NOTE]
> O arquivo [`docker-compose.yml`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/docker-compose.yml) já monta o socket do Docker em modo somente-leitura (`/var/run/docker.sock:ro`) e sincroniza o arquivo [`targets.json`](file:///home/jairosjunior/Documentos/Jairo/automação-jira/guardian-infra/targets.json).

---

### Método 4: Execução do Motor de Agentes Python (v2.0)

Caso deseje executar o motor heurístico de agentes diretamente pelo Python:

```bash
# Criar e ativar o ambiente virtual (se ainda não existir)
python3 -m venv .venv
source .venv/bin/activate

# Instalar as dependências necessárias
pip install -r requirements.txt

# Executar o orquestrador de observabilidade
python3 main.py
```

---

## 3. 🔍 Verificação de Saúde e Diagnóstico

### Checagem da API REST
```bash
curl -s http://localhost:8080/api/health | jq .
```
Resposta esperada:
```json
{
  "status": "healthy",
  "version": "2.1.0",
  "active_workers": 2,
  "uptime": "15m32s"
}
```

### Checagem da Lista de Targets Ativos
```bash
curl -s http://localhost:8080/api/targets | jq .
```

---

## 4. 🛠️ Solução de Problemas Comuns

### 1. `permission denied while trying to connect to the Docker daemon socket`
**Causa**: Seu usuário não possui permissão para ler o socket `/var/run/docker.sock`.  
**Solução**:
```bash
sudo usermod -aG docker $USER
newgrp docker
# Ou ajuste pontualmente as permissões de leitura
sudo chmod 666 /var/run/docker.sock
```

### 2. `address already in use: 8080`
**Causa**: Outro serviço (ex: Tomcat, Jenkins, dev-server) está usando a porta 8080.  
**Solução**:
```bash
# Identificar qual processo está ocupando a porta
sudo lsof -i :8080
# Ou altere a variável PORT no arquivo .env
echo "PORT=8090" >> .env
```

### 3. Falha de Leitura do arquivo `targets.json`
**Causa**: Sintaxe JSON corrompida durante edição manual.  
**Solução**:
O Guardian recria o `targets.json` automaticamente com os targets padrão caso o arquivo esteja vazio ou corrompido. Para resetar:
```bash
echo "[]" > targets.json
./guardian.sh restart
```
