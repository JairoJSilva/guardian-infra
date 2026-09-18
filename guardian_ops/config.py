import os
from pathlib import Path
from urllib.parse import urlparse
from dotenv import load_dotenv

from typing import Optional, Dict, Any

# Carrega arquivo .env da raiz do projeto ou diretório pai se existir
env_path = Path(__file__).resolve().parent.parent / ".env"
if not env_path.exists():
    env_path = Path(__file__).resolve().parent.parent.parent / ".env"
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
    JIRA_CONTRATO_DEFAULT: str = os.getenv("JIRA_CONTRATO_DEFAULT", "INTERNO").strip()
    JIRA_CUSTOMFIELD_CONTRATO_ID: str = os.getenv("JIRA_CUSTOMFIELD_CONTRATO_ID", "customfield_30118").strip()

    # Mapeamento oficial dos Contratos FLOWTI no Jira (customfield_30118)
    CONTRATO_MAP: Dict[str, Dict[str, str]] = {
        "INTERNO": {"id": "22514", "value": "INTERNO"},
        "DENTALIS": {"id": "22300", "value": "DENTALIS"},
        "DENTALIS (SNOW FLAKE)": {"id": "22500", "value": "DENTALIS (SNOW FLAKE)"},
        "FARMACIA DIGITAL (AWS)": {"id": "22502", "value": "FARMACIA DIGITAL (AWS)"},
        "GIF": {"id": "22517", "value": "GIF"},
        "MAIDA (DIGITAL OCEAN AKAMAI)": {"id": "22303", "value": "MAIDA (DIGITAL OCEAN AKAMAI)"},
        "MAIDA (GCP)": {"id": "22301", "value": "MAIDA (GCP)"},
        "MAIDA (LEGADO AWS)": {"id": "22302", "value": "MAIDA (LEGADO AWS)"},
        "MAIDA BI CLOUDOPS (AZURE/OCI)": {"id": "22501", "value": "MAIDA BI CLOUDOPS (AZURE/OCI)"},
    }

    @classmethod
    def resolve_contract(cls, hint: Optional[str] = None) -> Dict[str, str]:
        """
        Resolve o objeto de contrato Jira {'id': ..., 'value': ...}
        baseado em um identificador, nome de namespace, projeto ou label.
        Se não encontrar correspondência específica, retorna o contrato padrão (INTERNO).
        """
        default_contrato = cls.CONTRATO_MAP.get(cls.JIRA_CONTRATO_DEFAULT, cls.CONTRATO_MAP["INTERNO"])
        if not hint:
            return default_contrato

        hint_upper = hint.strip().upper()

        # 1. Correspondência exata no mapa
        if hint_upper in cls.CONTRATO_MAP:
            return cls.CONTRATO_MAP[hint_upper]

        # 2. Correspondência por palavras-chave
        if "SNOW" in hint_upper or "FLAKE" in hint_upper:
            return cls.CONTRATO_MAP["DENTALIS (SNOW FLAKE)"]
        if "DENTALIS" in hint_upper:
            return cls.CONTRATO_MAP["DENTALIS"]
        if "FARMACIA" in hint_upper or "DROGARIA" in hint_upper:
            return cls.CONTRATO_MAP["FARMACIA DIGITAL (AWS)"]
        if "GIF" in hint_upper:
            return cls.CONTRATO_MAP["GIF"]
        if "MAIDA" in hint_upper:
            if "GCP" in hint_upper:
                return cls.CONTRATO_MAP["MAIDA (GCP)"]
            if "AKAMAI" in hint_upper or "OCEAN" in hint_upper or "DO" in hint_upper:
                return cls.CONTRATO_MAP["MAIDA (DIGITAL OCEAN AKAMAI)"]
            if "BI" in hint_upper or "AZURE" in hint_upper or "OCI" in hint_upper:
                return cls.CONTRATO_MAP["MAIDA BI CLOUDOPS (AZURE/OCI)"]
            return cls.CONTRATO_MAP["MAIDA (GCP)"]

        # 3. Infraestrutura geral, monitoramento ou termos internos
        if any(term in hint_upper for term in ["INTERNO", "INFRA", "KUBE", "MONITOR", "DEFAULT", "CORE", "SYSTEM", "DEV", "STAGING"]):
            return cls.CONTRATO_MAP["INTERNO"]

        # 4. Fallback padrão
        return default_contrato

    # GuardianOps Operational Settings
    DRY_RUN: bool = _get_bool("GUARDIAN_DRY_RUN", default=False)
    COOLDOWN_MINUTES: int = int(os.getenv("GUARDIAN_COOLDOWN_MINUTES", "60"))
    _env_cache = os.getenv("GUARDIAN_CACHE_FILE")
    CACHE_FILE: Path = Path(_env_cache) if _env_cache else Path(__file__).resolve().parent.parent / "guardian_cache.json"
    DEFAULT_ENVIRONMENT: str = os.getenv("GUARDIAN_ENVIRONMENT", "AWS Produção").strip()

    # Webhook Server (para pipelines CI/CD)
    WEBHOOK_HOST: str = os.getenv("WEBHOOK_HOST", "0.0.0.0")
    WEBHOOK_PORT: int = int(os.getenv("WEBHOOK_PORT", "8080"))

    # STRICT SAFETY ENFORCEMENT:
    # O GuardianOps opera 100% em modo somente-leitura.
    READ_ONLY_MODE: bool = True
