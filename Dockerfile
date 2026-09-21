# Imagem base Python slim
FROM python:3.12-slim

# Metadados da imagem
LABEL maintainer="GuardianOps Open Source Team <guardian@ops.local>"
LABEL description="GuardianOps - Observador e Analisador de Falhas K8s e CI/CD"

# Evita geração de bytecode .pyc e força stdout não-bufferizado
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PYTHONIOENCODING=utf-8 \
    LANG=C.UTF-8 \
    LC_ALL=C.UTF-8

# Diretório de trabalho
WORKDIR /app

# Instala dependências do sistema necessárias para curl/saúde
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copia e instala dependências Python
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Cria usuário não-root por boas práticas de segurança K8s
RUN useradd -u 10001 -m guardian && mkdir -p /app/cache && chown -R guardian:guardian /app

# Copia código da aplicação
COPY --chown=guardian:guardian guardian_ops /app/guardian_ops
COPY --chown=guardian:guardian main.py /app/main.py
COPY --chown=guardian:guardian apresentacao-guardianops.html /app/apresentacao-guardianops.html

# Alterna para o usuário não-root
USER 10001

# Expõe porta do listener de webhooks
EXPOSE 8080

# Checagem de integridade interna
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

# Comando padrão: inicia observador de containers locais com listener de webhook integrado
CMD ["python3", "main.py", "watch", "--interval", "15", "--webhook-port", "8080"]
