import os
from pathlib import Path
from urllib.parse import urlparse
from dotenv import load_dotenv

# Carrega arquivo .env da raiz do projeto se existir
env_path = Path(__file__).resolve().parent.parent / ".env"
load_dotenv(dotenv_path=env_path)

def _get_bool(key: str, default: bool = False) -> bool:
    val = os.getenv(key, "").strip().lower()
    if val in ("true", "1", "yes", "sim"):
        return True
    if val in ("false", "0", "no", "nao"):
        return False
    return default

def _clean_url(url: str) -> str:
    parsed = urlparse(url)
    return f"{parsed.scheme}://{parsed.netloc}" if parsed.netloc else url.rstrip("/")

class Config:
    # Jira Settings
    JIRA_BASE_URL: str = _clean_url(os.getenv("JIRA_BASE_URL", "https://jira.mv.com.br"))
    JIRA_USER: str = os.getenv("JIRA_USER", "").strip()
    JIRA_PASSWORD: str = os.getenv("JIRA_PASSWORD", "").strip()
    JIRA_PROJECT_KEY: str = os.getenv("JIRA_PROJECT_KEY", "OPS").strip()
    JIRA_ISSUE_TYPE: str = os.getenv("JIRA_ISSUE_TYPE", "Solicitação de serviço").strip()
    JIRA_CONTRATO_DEFAULT: str = os.getenv("JIRA_CONTRATO_DEFAULT", "DENTALIS").strip()
    JIRA_CUSTOMFIELD_CONTRATO_ID: str = os.getenv("JIRA_CUSTOMFIELD_CONTRATO_ID", "customfield_30118").strip()

    # Mapeamento de contratos Jira
    CONTRATO_MAP = {
        "DENTALIS": {"id": "22300", "value": "DENTALIS"}
    }

    # GuardianOps Operational Settings
    DRY_RUN: bool = _get_bool("GUARDIAN_DRY_RUN", default=False)
    COOLDOWN_MINUTES: int = int(os.getenv("GUARDIAN_COOLDOWN_MINUTES", "60"))
    CACHE_FILE: Path = Path(__file__).resolve().parent.parent / os.getenv("GUARDIAN_CACHE_FILE", "guardian_cache.json")
    DEFAULT_ENVIRONMENT: str = os.getenv("GUARDIAN_ENVIRONMENT", "AWS Produção").strip()

    # Webhook Server (para pipelines CI/CD)
    WEBHOOK_HOST: str = os.getenv("WEBHOOK_HOST", "0.0.0.0")
    WEBHOOK_PORT: int = int(os.getenv("WEBHOOK_PORT", "8080"))

    # STRICT SAFETY ENFORCEMENT:
    # O GuardianOps opera 100% em modo somente-leitura.
    READ_ONLY_MODE: bool = True
