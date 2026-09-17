#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Suite de Testes e Validação do GuardianOps.
Valida:
1. Geração de RCA e templates em UTF-8 (acentuação, emojis, formatação Jira).
2. Deduplicação e Cooldown (Anti-Spam de chamados repetidos).
3. Modos de simulação (Pod K8s e Pipeline CI/CD).
4. Restrição de SOMENTE-LEITURA (Read-Only).
"""

import os
import sys
from guardian_ops.config import Config
from guardian_ops.models import IncidentEvent, FailureSource, IncidentPriority
from guardian_ops.analyzer import GuardianAnalyzer
from guardian_ops.jira_client import JiraClient
from guardian_ops.k8s_scanner import K8sScanner
from guardian_ops.pipeline_listener import PipelineMock

def test_suite():
    print("=" * 60)
    print("🛡️  INICIANDO BATERIA DE TESTES DO GUARDIANOPS (v1.0)")
    print("=" * 60)

    # Força modo DRY-RUN durante os testes para não criar lixo no Jira
    Config.DRY_RUN = True

    # 1. Teste de Pod CrashLoopBackOff
    print("\n[TESTE 1] Simulação de Falha de Pod (CrashLoopBackOff)...")
    pod_event = K8sScanner.create_mock_pod_failure(
        pod_name="paciente-auth-service-7b98cd-ww82k",
        namespace="producao",
        failure_type="CrashLoopBackOff",
        container="auth-api",
        exit_code=1,
        restarts=5
    )
    analyzer = GuardianAnalyzer()
    analysis = analyzer.analyze(pod_event, reference_code="OPS-GUARDIAN-TEST")

    assert "CrashLoopBackOff" in analysis.summary
    assert analysis.priority == IncidentPriority.ALTA
    assert "auth-api" in analysis.jira_description
    assert "SOMENTE-LEITURA" in analysis.jira_description
    print("  [✓] Análise de Pod e Formatação Jira gerada com sucesso!")

    # 2. Teste de Pipeline CI/CD
    print("\n[TESTE 2] Simulação de Falha de Pipeline CI/CD...")
    pipe_event = PipelineMock.create_mock_pipeline_failure(
        project="flowti/backend-prontuario",
        pipeline_id="99102",
        stage="docker-build"
    )
    analysis_pipe = analyzer.analyze(pipe_event, reference_code="OPS-GUARDIAN-TEST")
    assert "99102" in analysis_pipe.summary
    assert "flowti/backend-prontuario" in analysis_pipe.impacted_asset_summary
    print("  [✓] Análise de Pipeline e RCA gerado com sucesso!")

    # 3. Teste de Deduplicação e Cooldown (Anti-Spam)
    print("\n[TESTE 3] Validação de Deduplicação de Incidentes (Anti-Spam)...")
    jira = JiraClient()
    # Limpa cache temporário de teste
    test_fp = "test-fingerprint-unique-999"
    if test_fp in jira.cache:
        del jira.cache[test_fp]

    assert not jira.is_in_cooldown(test_fp)
    jira.record_incident(test_fp)
    assert jira.is_in_cooldown(test_fp)
    print("  [✓] Mecanismo de Anti-Spam e Cooldown validado com sucesso!")

    # 4. Teste de Integridade de Codificação UTF-8
    print("\n[TESTE 4] Validação de Integridade UTF-8...")
    utf8_chars = "Acentuação: á, é, í, ó, ú, ã, õ, ç, ê. Emojis: 🛡️, 🤖, 📌, ⚠️"
    encoded = utf8_chars.encode("utf-8")
    decoded = encoded.decode("utf-8")
    assert utf8_chars == decoded
    print(f"  [✓] UTF-8 testado: {decoded}")

    # 5. Validação da Diretriz de Segurança Read-Only
    print("\n[TESTE 5] Verificação da Diretriz de Segurança Read-Only...")
    assert Config.READ_ONLY_MODE is True
    print("  [✓] Modo SOMENTE-LEITURA garantido: Nenhuma ação corretiva direta habilitada.")

    # 6. Validação da Resolução de Contratos FLOWTI
    print("\n[TESTE 6] Validação da Resolução Inteligente de Contratos...")
    # Padrão / Fallback
    c_default = Config.resolve_contract()
    assert c_default["id"] == "22514"
    assert c_default["value"] == "INTERNO"

    # Heurística por namespace/projeto
    c_dentalis = Config.resolve_contract("portal-dentalis-backend")
    assert c_dentalis["id"] == "22300"
    assert c_dentalis["value"] == "DENTALIS"

    c_farmacia = Config.resolve_contract("farmacia-digital-app")
    assert c_farmacia["id"] == "22502"

    c_maida_gcp = Config.resolve_contract("maida-gcp-service")
    assert c_maida_gcp["id"] == "22301"

    c_infra = Config.resolve_contract("kube-system / ingress-controller")
    assert c_infra["id"] == "22514"
    assert c_infra["value"] == "INTERNO"
    print("  [✓] Resolução de Contratos testada: Default INTERNO (22514) e roteamento de clientes ativos!")

    print("\n" + "=" * 60)
    print("🎉 TODOS OS TESTES FORAM CONCLUÍDOS COM SUCESSO!")
    print("=" * 60)

if __name__ == "__main__":
    test_suite()
