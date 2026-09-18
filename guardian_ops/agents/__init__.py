# -*- coding: utf-8 -*-
"""
GuardianOps — Sistema de Agentes Especializados Autônomos
=========================================================
5 agentes especializados para diagnóstico e resolução de incidentes:

  01. DevOpsCloudNativeAgent   — IaC, Docker, Kubernetes, CI/CD
  02. DatabaseSpecialistAgent  — PostgreSQL, MySQL, MongoDB, Redis, SQL Server
  03. SeniorQAAgent            — Cypress, Playwright, Jest, PyTest, JUnit
  04. FullStackDeveloperAgent  — Node.js, Python, Java, React, Vue, Angular
  05. OrchestratorBotAgent     — Roteamento, consolidação e abertura de chamados
"""
from guardian_ops.agents.base_agent import BaseAgent, AgentDiagnosis
from guardian_ops.agents.devops_agent import DevOpsCloudNativeAgent
from guardian_ops.agents.database_agent import DatabaseSpecialistAgent
from guardian_ops.agents.qa_agent import SeniorQAAgent
from guardian_ops.agents.fullstack_agent import FullStackDeveloperAgent
from guardian_ops.agents.orchestrator_agent import OrchestratorBotAgent

__all__ = [
    "BaseAgent",
    "AgentDiagnosis",
    "DevOpsCloudNativeAgent",
    "DatabaseSpecialistAgent",
    "SeniorQAAgent",
    "FullStackDeveloperAgent",
    "OrchestratorBotAgent",
]
