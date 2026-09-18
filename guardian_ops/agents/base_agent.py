# -*- coding: utf-8 -*-
"""
BaseAgent — Classe abstrata base para todos os agentes especializados do GuardianOps.

Todos os agentes herdam desta classe e devem implementar o método `analyze()`.
A saída padronizada em AgentDiagnosis garante que o OrchestratorBotAgent
possa consolidar os diagnósticos de múltiplos especialistas em um único chamado.
"""
from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from enum import Enum
from typing import Dict, Any, List, Optional


class Severity(str, Enum):
    """Níveis de severidade padronizados conforme agents.md."""
    LOW      = "LOW"
    MEDIUM   = "MEDIUM"
    HIGH     = "HIGH"
    CRITICAL = "CRITICAL"


class DefectLayer(str, Enum):
    """Camada onde o defeito foi identificado."""
    INFRASTRUCTURE = "Infrastructure"
    CONTAINER      = "Container"
    ORCHESTRATION  = "Orchestration"
    DATABASE       = "Database"
    BACKEND        = "Backend"
    FRONTEND       = "Frontend"
    API            = "API"
    CI_CD          = "CI/CD"
    AUTOMATION     = "Automation"
    UNKNOWN        = "Unknown"


@dataclass
class AgentDiagnosis:
    """
    Resultado padronizado produzido por qualquer agente especialista.
    Usado pelo OrchestratorBotAgent para montar o chamado final.
    """
    agent_id: str                          # Ex: "agent-devops-cloudnative"
    agent_name: str                        # Ex: "DevOps & Cloud Native Agent"
    severity: Severity                     # LOW | MEDIUM | HIGH | CRITICAL
    defect_layer: DefectLayer              # Camada onde o defeito ocorreu
    root_cause: str                        # Diagnóstico de Causa Raiz (RCA)
    affected_component: str               # Serviço, pod, query, endpoint afetado
    fix_code: str                          # Patch/diff/comando de correção
    fix_description: str                  # Descrição técnica da mudança
    repro_steps: List[str] = field(default_factory=list)     # Passos para reproduzir
    validation_steps: List[str] = field(default_factory=list) # Passos de validação (QA)
    extra: Dict[str, Any] = field(default_factory=dict)      # Dados extras livres

    def to_jira_section(self) -> str:
        """Serializa o diagnóstico como seção de Wiki Markup do Jira."""
        lines = [
            f"h4. Agente: {self.agent_name} | Severidade: {self.severity.value}",
            f"*Camada:* {self.defect_layer.value}",
            f"*Componente Afetado:* {self.affected_component}",
            "",
            f"*Causa Raiz:*",
            self.root_cause,
            "",
        ]
        if self.repro_steps:
            lines.append("*Passos para Reproduzir:*")
            for i, step in enumerate(self.repro_steps, 1):
                lines.append(f"{i}. {step}")
            lines.append("")

        if self.fix_code:
            lines.append("*Patch / Correcao Proposta:*")
            lines.append(f"{{code}}")
            lines.append(self.fix_code)
            lines.append(f"{{code}}")
            lines.append("")

        if self.fix_description:
            lines.append(f"*Justificativa Tecnica:* {self.fix_description}")
            lines.append("")

        if self.validation_steps:
            lines.append("*Passos de Validacao (QA):*")
            for i, step in enumerate(self.validation_steps, 1):
                lines.append(f"{i}. {step}")

        return "\n".join(lines)

    def to_dict(self) -> Dict[str, Any]:
        """Serializa para dicionário (JSON-friendly)."""
        return {
            "agent_id": self.agent_id,
            "agent_name": self.agent_name,
            "severity": self.severity.value,
            "defect_layer": self.defect_layer.value,
            "root_cause": self.root_cause,
            "affected_component": self.affected_component,
            "fix_code": self.fix_code,
            "fix_description": self.fix_description,
            "repro_steps": self.repro_steps,
            "validation_steps": self.validation_steps,
            "extra": self.extra,
        }


class BaseAgent(ABC):
    """
    Classe abstrata base para todos os agentes especializados.

    Cada agente especializado deve:
      1. Herdar de BaseAgent
      2. Definir AGENT_ID e AGENT_NAME
      3. Implementar o método analyze()
    """
    AGENT_ID: str = "agent-base"
    AGENT_NAME: str = "Base Agent"

    def can_handle(self, payload: Dict[str, Any]) -> bool:
        """
        Verifica se este agente é capaz de processar o payload recebido.
        Subclasses podem sobrescrever para filtrar por tipo de evento.
        Por padrão, aceita qualquer payload.
        """
        return True

    @abstractmethod
    def analyze(self, payload: Dict[str, Any]) -> AgentDiagnosis:
        """
        Analisa o payload de evento/incidente e retorna um diagnóstico estruturado.

        Args:
            payload: Dicionário com dados do incidente. Campos comuns:
                - source (str): origem (kubernetes, docker, database, pipeline, app)
                - error_message (str): mensagem de erro principal
                - logs (str): trecho de logs
                - component (str): serviço/pod/container afetado
                - namespace (str): namespace K8s (se aplicável)
                - failure_type (str): tipo da falha
                - details (dict): dados extras específicos por fonte

        Returns:
            AgentDiagnosis com diagnóstico completo e patch de correção.
        """
        ...

    def _detect_severity(self, failure_type: str, restarts: int = 0) -> Severity:
        """Helper: detecta severidade baseado no tipo de falha e reinicializações."""
        ft = failure_type.upper()
        if any(k in ft for k in ["OOMKILLED", "CRASHLOOP", "CRITICAL", "DOWN", "FATAL", "DEADLOCK"]):
            return Severity.CRITICAL if restarts >= 5 else Severity.HIGH
        if any(k in ft for k in ["ERROR", "FAIL", "TIMEOUT", "SLOW", "HIGH"]):
            return Severity.MEDIUM
        return Severity.LOW

    def __repr__(self) -> str:
        return f"<{self.__class__.__name__} id={self.AGENT_ID}>"
