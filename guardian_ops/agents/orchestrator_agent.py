# -*- coding: utf-8 -*-
"""
05. Orchestrator Bot Agent (Agente Orquestrador)
=================================================
Identificador : agent-orchestrator-bot
Responsável   : Captura de eventos, Roteamento para especialistas,
                Consolidação de diagnósticos e Abertura Autônoma de Chamados.
Escopo        : Event-Driven, Docker/K8s, REST APIs, Jira, GitHub, GitLab,
                Consolidação de respostas de múltiplos agentes especialistas.

Fluxo de trabalho:
  1. Recebe evento de erro (webhook, scan, CLI)
  2. Aciona os agentes especialistas relevantes via router
  3. Consolida os diagnósticos em um chamado unificado
  4. Gera o payload formatado para o Jira (Wiki Markup)
  5. Abre o chamado automaticamente
"""
from typing import Dict, Any, List, Optional
from datetime import datetime

from guardian_ops.agents.base_agent import BaseAgent, AgentDiagnosis, Severity, DefectLayer
from guardian_ops.agents.devops_agent import DevOpsCloudNativeAgent
from guardian_ops.agents.database_agent import DatabaseSpecialistAgent
from guardian_ops.agents.qa_agent import SeniorQAAgent
from guardian_ops.agents.fullstack_agent import FullStackDeveloperAgent


class OrchestratorBotAgent(BaseAgent):
    """
    Agente Orquestrador Central — Coordena todos os agentes especialistas.

    Responsável por:
    - Receber o payload de evento bruto
    - Detectar qual(is) agente(s) devem ser acionados
    - Acionar os agentes em paralelo lógico
    - Consolidar os diagnósticos em um único chamado Jira
    - Gerar o payload final formatado para abertura do ticket
    """
    AGENT_ID   = "agent-orchestrator-bot"
    AGENT_NAME = "Bot Automation & Orchestrator Agent"

    # Todos os agentes especialistas registrados
    SPECIALIST_AGENTS: List[BaseAgent] = [
        DevOpsCloudNativeAgent(),
        DatabaseSpecialistAgent(),
        SeniorQAAgent(),
        FullStackDeveloperAgent(),
    ]

    def can_handle(self, payload: Dict[str, Any]) -> bool:
        """O Orquestrador aceita qualquer payload."""
        return True

    def analyze(self, payload: Dict[str, Any]) -> AgentDiagnosis:
        """
        Roteamento inteligente: aciona os agentes compatíveis com o evento
        e retorna um diagnóstico consolidado.
        """
        print(f"\n[{self.AGENT_NAME}] Iniciando triagem do evento...")
        diagnoses = self._route_and_analyze(payload)
        return self._consolidate(payload, diagnoses)

    def _route_and_analyze(self, payload: Dict[str, Any]) -> List[AgentDiagnosis]:
        """
        Detecta qual(is) agente(s) podem tratar o payload e os aciona.
        Retorna lista de diagnósticos de todos os especialistas acionados.
        """
        triggered: List[BaseAgent] = []
        for agent in self.SPECIALIST_AGENTS:
            if agent.can_handle(payload):
                triggered.append(agent)
                print(f"  [{self.AGENT_NAME}] -> Acionando: {agent.AGENT_NAME}")

        if not triggered:
            # Fallback: acionar o DevOps agent como default
            print(f"  [{self.AGENT_NAME}] -> Nenhum agente específico detectado. Usando fallback: DevOps Agent")
            triggered = [DevOpsCloudNativeAgent()]

        diagnoses = []
        for agent in triggered:
            try:
                diag = agent.analyze(payload)
                diagnoses.append(diag)
                print(f"  [{self.AGENT_NAME}] -> {agent.AGENT_NAME}: diagnóstico concluído [{diag.severity.value}]")
            except Exception as e:
                print(f"  [{self.AGENT_NAME}] -> ERRO em {agent.AGENT_NAME}: {e}")

        return diagnoses

    def _consolidate(self, payload: Dict[str, Any], diagnoses: List[AgentDiagnosis]) -> AgentDiagnosis:
        """
        Consolida múltiplos diagnósticos em um único AgentDiagnosis
        com a descrição completa pronta para o Jira.
        """
        if not diagnoses:
            return self._empty_diagnosis(payload)

        # Severidade final = máxima entre todos os agentes
        severity_order = [Severity.LOW, Severity.MEDIUM, Severity.HIGH, Severity.CRITICAL]
        max_severity = max(diagnoses, key=lambda d: severity_order.index(d.severity)).severity

        # Componente principal = primeiro diagnóstico
        primary = diagnoses[0]
        all_components = list({d.affected_component for d in diagnoses})

        # Montar corpo consolidado
        component = payload.get("component", payload.get("identifier", "Unknown Component"))
        environment = payload.get("environment", "Produção")
        timestamp = datetime.now().strftime("%d/%m/%Y %H:%M:%S")
        failure_type = payload.get("failure_type", "Falha Operacional")

        # Gerar título conforme agents.md
        title = f"[AUTOMACAO] Erro Detectado no Ambiente - {component} ({failure_type})"

        # Montar seções dos especialistas
        specialist_sections = "\n\n----\n\n".join(
            d.to_jira_section() for d in diagnoses
        )

        # Consolidar repro_steps e validation_steps
        all_repro = []
        all_validation = []
        for d in diagnoses:
            all_repro.extend(d.repro_steps)
            all_validation.extend(d.validation_steps)
        # Deduplicar mantendo ordem
        seen = set()
        repro_unique = []
        for s in all_repro:
            if s not in seen:
                seen.add(s)
                repro_unique.append(s)
        seen = set()
        validation_unique = []
        for s in all_validation:
            if s not in seen:
                seen.add(s)
                validation_unique.append(s)

        # Montar fix_code consolidado
        fix_parts = []
        for d in diagnoses:
            if d.fix_code:
                fix_parts.append(f"# === {d.agent_name.upper()} ===\n{d.fix_code}")
        consolidated_fix = "\n\n".join(fix_parts)

        # Montar jira_description completo (Wiki Markup)
        jira_description = self._build_jira_description(
            title=title,
            severity=max_severity,
            component=component,
            environment=environment,
            timestamp=timestamp,
            failure_type=failure_type,
            diagnoses=diagnoses,
            specialist_sections=specialist_sections,
            repro_steps=repro_unique,
            validation_steps=validation_unique,
            payload=payload,
        )

        print(f"\n[{self.AGENT_NAME}] Consolidação concluída:")
        print(f"  Agentes acionados : {len(diagnoses)}")
        print(f"  Severidade Final  : {max_severity.value}")
        print(f"  Componentes       : {', '.join(all_components)}")

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=max_severity,
            defect_layer=primary.defect_layer,
            root_cause="\n\n".join(d.root_cause for d in diagnoses),
            affected_component=" | ".join(all_components),
            fix_code=consolidated_fix,
            fix_description=f"Diagnóstico consolidado de {len(diagnoses)} agente(s) especialista(s).",
            repro_steps=repro_unique,
            validation_steps=validation_unique,
            extra={
                "jira_description": jira_description,
                "title": title,
                "agents_triggered": [d.agent_id for d in diagnoses],
                "severity": max_severity.value,
                "environment": environment,
                "timestamp": timestamp,
                "component": component,
                "failure_type": failure_type,
            },
        )

    def _build_jira_description(
        self,
        title: str,
        severity: Severity,
        component: str,
        environment: str,
        timestamp: str,
        failure_type: str,
        diagnoses: List[AgentDiagnosis],
        specialist_sections: str,
        repro_steps: List[str],
        validation_steps: List[str],
        payload: Dict[str, Any],
    ) -> str:
        """
        Gera a descrição completa no formato Wiki Markup do Jira,
        seguindo o template definido em agents.md (formato padrão do chamado).
        """
        logs_snippet = payload.get("logs", payload.get("logs_snippet", ""))
        agents_list = ", ".join(d.agent_name for d in diagnoses)

        # Seção de Evidências
        evidence_block = ""
        if logs_snippet and logs_snippet.strip():
            snippet = logs_snippet.strip()
            if len(snippet) > 2000:
                snippet = snippet[:2000] + "\n... [TRECHO TRUNCADO PELO GUARDIANOPS]"
            evidence_block = f"""
*Evidencias Coletadas (Log / Stack Trace):*
{{code:text}}
{snippet}
{{code}}"""

        # Passos de reprodução
        repro_block = "\n".join(f"{i+1}. {step}" for i, step in enumerate(repro_steps)) if repro_steps else "Ver logs da aplicacao."
        # Passos de validação
        validation_block = "\n".join(f"* {step}" for step in validation_steps) if validation_steps else "* Componente operando normalmente."

        # Severidade colorida (Jira Wiki Markup)
        severity_color = {
            Severity.LOW: "green",
            Severity.MEDIUM: "yellow",
            Severity.HIGH: "red",
            Severity.CRITICAL: "red",
        }.get(severity, "red")

        description = f"""*Referencia:* OPS-GUARDIAN
*Data:* {timestamp}
*Ambiente:* {environment}
*Origem:* Gerado por I.A. (GuardianOps - Sistema de Agentes Autonomos)
*Agentes Acionados:* {agents_list}
*Tag de Rastreabilidade:* {{{{gerado-por-ia}}}}, {{{{guardian-ops}}}}

----

h2. {title}

* *Servico/Pod:* {component}
* *Severidade:* {{color:{severity_color}}}{severity.value}{{color}}
* *Tipo de Falha:* {failure_type}
* *Ambiente:* {environment}
* *Timestamp:* {timestamp}

----

h3. 1. Resumo do Incidente

Falha detectada no componente *{component}* no ambiente *{environment}*.
O sistema de agentes autonomos GuardianOps identificou a anomalia e acionou {len(diagnoses)} agente(s) especialista(s)
para realizar o diagnostico de causa raiz e propor a correcao.

> *AVISO DE GOVERNANCA:* O *GuardianOps* atua exclusivamente em modo de *SOMENTE-LEITURA* (Read-Only).
Nenhuma modificacao automatica foi realizada no ambiente. A equipe responsavel deve executar
o roteiro de correcao abaixo manualmente ou via PR/MR.

----

h3. 2. Diagnostico por Agente Especialista

{specialist_sections}

----

h3. 3. Evidencia do Erro (Log / Stack Trace)
{evidence_block if evidence_block else '_Nenhum trecho de log fornecido. Verificar logs do servico._'}

----

h3. 4. Passos de Reproducao

{repro_block}

----

h3. 5. Passos de Validacao (QA)

{validation_block}

----

h3. 6. Referencias

* Agentes Acionados: {agents_list}
* Sistema: GuardianOps v2.0 - Multi-Agent Autonomous Incident Response
* Modo: Somente-Leitura (Read-Only) - Nenhuma acao automatica realizada"""

        return description.strip()

    def _empty_diagnosis(self, payload: Dict[str, Any]) -> AgentDiagnosis:
        """Retorna diagnóstico genérico quando nenhum agente pôde ser acionado."""
        component = payload.get("component", payload.get("identifier", "Unknown"))
        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.MEDIUM,
            defect_layer=DefectLayer.UNKNOWN,
            root_cause=f"Evento recebido para o componente *{component}*, mas nenhum agente especialista pôde ser identificado para o tipo de falha informado.",
            affected_component=component,
            fix_code="# Inspecionar manualmente os logs e eventos do componente.",
            fix_description="Analisar manualmente o incidente.",
            repro_steps=["Verificar logs do componente afetado"],
            validation_steps=["Componente restaurado para operação normal"],
        )

    # -------------------------------------------------------------------------
    # Métodos utilitários para integração com o sistema existente (JiraClient)
    # -------------------------------------------------------------------------

    def build_jira_payload(self, payload: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Executa o fluxo completo de análise e retorna o payload pronto para
        ser enviado ao JiraClient.create_incident_issue().

        Retorna dict com:
          - summary: str (título do chamado)
          - description: str (Wiki Markup completo)
          - severity: str
          - labels: list
          - agents_triggered: list
        """
        diagnosis = self.analyze(payload)
        jira_desc = diagnosis.extra.get("jira_description", diagnosis.to_jira_section())

        severity_to_priority = {
            Severity.LOW:      "Baixa",
            Severity.MEDIUM:   "Média",
            Severity.HIGH:     "Alta",
            Severity.CRITICAL: "Alta",
        }

        return {
            "summary":         diagnosis.extra.get("title", f"[AUTOMACAO] Falha em {diagnosis.affected_component}"),
            "description":     jira_desc,
            "priority":        severity_to_priority.get(diagnosis.severity, "Média"),
            "severity":        diagnosis.severity.value,
            "labels":          ["gerado-por-ia", "guardian-ops", "multi-agent"],
            "agents_triggered": diagnosis.extra.get("agents_triggered", []),
            "environment":     diagnosis.extra.get("environment", "Produção"),
            "component":       diagnosis.extra.get("component", ""),
            "failure_type":    diagnosis.extra.get("failure_type", ""),
            "root_cause":      diagnosis.root_cause,
        }

    @staticmethod
    def list_agents() -> List[str]:
        """Retorna lista de agentes registrados no orquestrador."""
        return [
            f"[{a.AGENT_ID}] {a.AGENT_NAME}"
            for a in OrchestratorBotAgent.SPECIALIST_AGENTS
        ]
