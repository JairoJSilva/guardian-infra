from dataclasses import dataclass, field
from enum import Enum
from typing import Optional, Dict, Any, List
from datetime import datetime

class FailureSource(str, Enum):
    KUBERNETES_POD = "KUBERNETES_POD"
    PIPELINE_CI_CD = "PIPELINE_CI_CD"

class IncidentPriority(str, Enum):
    CRITICA = "Crítica"
    ALTA = "Alta"
    MEDIA = "Média"
    BAIXA = "Baixa"

@dataclass
class IncidentEvent:
    source: FailureSource
    identifier: str              # Nome do Pod ou ID/Nome da Pipeline
    namespace_or_project: str    # Namespace K8s ou Projeto GitLab/GitHub
    failure_type: str            # CrashLoopBackOff, OOMKilled, ImagePullBackOff, JobFailed, etc.
    environment: str = "Produção"
    container_or_stage: str = "" # Nome do container ou stage da pipeline
    error_message: str = ""
    logs_snippet: str = ""
    exit_code: Optional[int] = None
    restart_count: Optional[int] = None
    details: Dict[str, Any] = field(default_factory=dict)
    timestamp: str = field(default_factory=lambda: datetime.now().strftime("%d/%m/%Y %H:%M:%S"))

    @property
    def fingerprint(self) -> str:
        """Gera uma chave única para deduplicação de incidentes e evitar spam de chamados."""
        import hashlib
        raw = f"{self.source.value}:{self.namespace_or_project}:{self.identifier}:{self.failure_type}"
        return hashlib.md5(raw.encode("utf-8")).hexdigest()

@dataclass
class AnalysisResult:
    summary: str
    priority: IncidentPriority
    category: str
    issue_type: str
    root_cause: str
    impacted_asset_summary: str
    remediation_procedure: str
    acceptance_criteria: str
    rollback_plan: str
    jira_description: str
    labels: List[str] = field(default_factory=lambda: ["gerado-por-ia", "guardian-ops"])
    contract_field: Optional[Dict[str, str]] = None
