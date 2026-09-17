#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
GuardianOps - Agente Observador e Analisador de Falhas de Aplicações e Pipelines.
Versão 1.0 (Modo Somente-Leitura / Read-Only).
"""

import sys
import argparse
from guardian_ops.config import Config
from guardian_ops.models import IncidentEvent
from guardian_ops.analyzer import GuardianAnalyzer
from guardian_ops.jira_client import JiraClient
from guardian_ops.k8s_scanner import K8sScanner
from guardian_ops.docker_scanner import DockerScanner
from guardian_ops.pipeline_listener import PipelineMock, run_webhook_server

def handle_incident(event: IncidentEvent, parent_issue: str = None, dry_run: bool = False):
    """Processa um incidente: analisa a falha e abre o chamado no Jira."""
    print("\n" + "═" * 65)
    print(f"🛡️  [GuardianOps] INCIDENTE DETECTADO: {event.failure_type}")
    print(f"   Origem: {event.source.value} | Alvo: {event.identifier}")
    print(f"   Escopo: {event.namespace_or_project} | Container/Stage: {event.container_or_stage}")
    print("═" * 65)

    if dry_run:
        Config.DRY_RUN = True

    analyzer = GuardianAnalyzer()
    analysis = analyzer.analyze(event)

    jira = JiraClient()
    result = jira.create_incident_issue(event, analysis, parent_issue_key=parent_issue)
    return result

def cmd_simulate_pod(args):
    print("[*] Gerando evento simulado de falha em Pod do Kubernetes...")
    event = K8sScanner.create_mock_pod_failure(
        pod_name=args.pod,
        namespace=args.namespace,
        failure_type=args.type,
        exit_code=args.exit_code,
        restarts=args.restarts
    )
    if getattr(args, "contrato", None):
        event.details["contrato"] = args.contrato
    handle_incident(event, parent_issue=args.parent, dry_run=args.dry_run)

def cmd_simulate_pipeline(args):
    print("[*] Gerando evento simulado de falha de Pipeline CI/CD...")
    event = PipelineMock.create_mock_pipeline_failure(
        project=args.project,
        pipeline_id=args.pipeline_id,
        stage=args.stage,
        error_type=args.error_type
    )
    if getattr(args, "contrato", None):
        event.details["contrato"] = args.contrato
    handle_incident(event, parent_issue=args.parent, dry_run=args.dry_run)

def cmd_scan_k8s(args):
    print(f"[*] Iniciando varredura SOMENTE-LEITURA nos namespaces: {args.namespaces}...")
    scanner = K8sScanner(namespaces=args.namespaces)
    incidents = scanner.scan_cluster_pods()
    if not incidents:
        print(" [✓] Nenhum pod em estado anômalo detectado nos namespaces inspecionados.")
        return
    print(f" [!] Encontrados {len(incidents)} pods com falhas/alertas:")
    for inc in incidents:
        handle_incident(inc, parent_issue=args.parent, dry_run=args.dry_run)

def cmd_webhook(args):
    print("[*] Iniciando servidor Webhook para recepção de alertas de CI/CD...")
    def on_incident(event: IncidentEvent):
        handle_incident(event, parent_issue=args.parent, dry_run=args.dry_run)
    run_webhook_server(host=args.host, port=args.port, callback=on_incident)

def cmd_simulate_docker(args):
    print("[*] Gerando evento simulado de falha em Container Docker local...")
    event = DockerScanner.create_mock_docker_failure(
        container_name=args.container,
        failure_type=args.type,
        exit_code=args.exit_code,
        restarts=args.restarts
    )
    if getattr(args, "contrato", None):
        event.details["contrato"] = args.contrato
    handle_incident(event, parent_issue=args.parent, dry_run=args.dry_run)

def cmd_watch(args):
    print("=" * 65)
    print("🛡️  [GuardianOps] MODO OBSERVADOR LOCAL ATIVO (SOMENTE-LEITURA)")
    print(f"   Socket Docker : {args.socket}")
    print(f"   Intervalo     : {args.interval}s")
    print(f"   Filtro Alvos  : {args.targets or 'Todos os containers locais'}")
    print(f"   Modo Dry-Run  : {'ATIVADO (Somente análise RCA, sem envio ao Jira)' if args.dry_run else 'DESATIVADO (Envia chamados para o Jira)'}")
    print(f"   Destino Jira  : Projeto {Config.JIRA_PROJECT_KEY} ({Config.JIRA_BASE_URL})")
    print("=" * 65)

    targets = [t.strip() for t in args.targets.split(",")] if args.targets else None
    scanner = DockerScanner(socket_path=args.socket, target_names=targets)

    # Inicia webhook server em background se solicitado
    if args.webhook_port > 0:
        import threading
        def on_webhook_incident(event: IncidentEvent):
            handle_incident(event, parent_issue=args.parent, dry_run=args.dry_run)
        t = threading.Thread(
            target=run_webhook_server,
            kwargs={"host": "0.0.0.0", "port": args.webhook_port, "callback": on_webhook_incident},
            daemon=True
        )
        t.start()
        print(f"[*] Listener de Webhooks ativo em segundo plano na porta {args.webhook_port}.")

    print("\n[*] Iniciando ciclo contínuo de observabilidade local...")
    import time
    loop_count = 0
    while True:
        loop_count += 1
        try:
            if not scanner.is_available():
                print(f"[{time.strftime('%H:%M:%S')}] [!] Socket Docker '{args.socket}' não encontrado ou inacessível. Aguardando...")
            else:
                containers = scanner.list_containers()
                c_names = [c['name'] for c in containers]
                incidents = scanner.scan_containers()

                if incidents:
                    print(f"\n[{time.strftime('%H:%M:%S')}] ⚠️  ANOMALIA DETECTADA! {len(incidents)} container(s) com falhas:")
                    for inc in incidents:
                        handle_incident(inc, parent_issue=args.parent, dry_run=args.dry_run)
                else:
                    if loop_count % 4 == 1 or loop_count == 1:
                        status_summary = f"{len(containers)} container(s) monitorados: {', '.join(c_names) if c_names else 'nenhum ativo'}"
                        print(f"[{time.strftime('%H:%M:%S')}] [✓] {status_summary} - Ambiente operacional estável.")

        except Exception as e:
            print(f"[{time.strftime('%H:%M:%S')}] [!] Erro durante ciclo de varredura: {e}")

        time.sleep(args.interval)

def cmd_test_jira(args):
    print(f"[*] Testando conectividade com Jira: {Config.JIRA_BASE_URL}")
    import requests
    url = f"{Config.JIRA_BASE_URL}/rest/api/2/myself"
    try:
        res = requests.get(url, auth=(Config.JIRA_USER, Config.JIRA_PASSWORD), timeout=15)
        if res.status_code == 200:
            data = res.json()
            print("\n" + "═" * 50)
            print(">>> AUTENTICAÇÃO COM O JIRA BEM-SUCEDIDA! <<<")
            print("═" * 50)
            print(f"Usuário Conectado : {data.get('displayName')} ({data.get('name')})")
            print(f"E-mail             : {data.get('emailAddress')}")
            print(f"Fuso Horário       : {data.get('timeZone')}")
            print(f"Projeto de Destino : {Config.JIRA_PROJECT_KEY}")
            print("═" * 50)
        else:
            print(f"[!] Falha de autenticação: HTTP {res.status_code}")
            print(res.text)
    except Exception as e:
        print(f"[!] Erro ao conectar no Jira: {e}")

def main():
    parser = argparse.ArgumentParser(
        description="GuardianOps - Monitor e Analisador de Falhas de Aplicações e Pipelines",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""Exemplos de uso:
  python3 main.py test-jira
  python3 main.py simulate-pod --dry-run
  python3 main.py simulate-pod --type OOMKilled --dry-run
  python3 main.py simulate-pipeline --dry-run
  python3 main.py webhook --port 8080 --dry-run
"""
    )

    subparsers = parser.add_subparsers(dest="command", help="Comando a executar")

    # test-jira
    subparsers.add_parser("test-jira", help="Testa conectividade e credenciais do Jira")

    # simulate-pod
    p_pod = subparsers.add_parser("simulate-pod", help="Simula falha de Pod do Kubernetes e gera chamado")
    p_pod.add_argument("--pod", default="portal-flowti-backend-69b7bf656b-x82d9", help="Nome do pod")
    p_pod.add_argument("--namespace", default="producao", help="Namespace do Kubernetes")
    p_pod.add_argument("--type", default="CrashLoopBackOff", choices=["CrashLoopBackOff", "OOMKilled", "ImagePullBackOff", "FailedLivenessProbe"], help="Tipo da falha")
    p_pod.add_argument("--exit-code", type=int, default=1, help="Exit Code do container")
    p_pod.add_argument("--restarts", type=int, default=8, help="Número de reinicializações")
    p_pod.add_argument("--parent", default="OPS-236", help="Issue de referência/pai para vincular")
    p_pod.add_argument("--contrato", default=None, help="Força contrato FLOWTI (ex: INTERNO, DENTALIS, MAIDA (GCP))")
    p_pod.add_argument("--dry-run", action="store_true", help="Gera análise e template sem postar no Jira")

    # simulate-pipeline
    p_pipe = subparsers.add_parser("simulate-pipeline", help="Simula falha de Pipeline CI/CD e gera chamado")
    p_pipe.add_argument("--project", default="flowti/portal-paciente", help="Nome do projeto/repositório")
    p_pipe.add_argument("--pipeline-id", default="48921", help="ID da pipeline")
    p_pipe.add_argument("--stage", default="docker-build", help="Estágio da pipeline com falha")
    p_pipe.add_argument("--error-type", default="DockerBuildConflict", help="Tipo do erro")
    p_pipe.add_argument("--parent", default="OPS-236", help="Issue de referência/pai para vincular")
    p_pipe.add_argument("--contrato", default=None, help="Força contrato FLOWTI (ex: INTERNO, DENTALIS, MAIDA (GCP))")
    p_pipe.add_argument("--dry-run", action="store_true", help="Gera análise e template sem postar no Jira")

    # scan-k8s
    p_scan = subparsers.add_parser("scan-k8s", help="Executa varredura somente-leitura em cluster K8s via kubectl")
    p_scan.add_argument("-n", "--namespaces", nargs="+", default=["default", "producao"], help="Namespaces a escanear")
    p_scan.add_argument("--parent", default="OPS-236", help="Issue de referência/pai para vincular")
    p_scan.add_argument("--dry-run", action="store_true", help="Simula sem postar no Jira")

    # webhook
    p_wh = subparsers.add_parser("webhook", help="Inicia listener HTTP para receber webhooks de CI/CD")
    p_wh.add_argument("--host", default="0.0.0.0", help="Host de escuta")
    p_wh.add_argument("--port", type=int, default=8080, help="Porta de escuta")
    p_wh.add_argument("--parent", default="OPS-236", help="Issue de referência/pai para vincular")
    p_wh.add_argument("--dry-run", action="store_true", help="Simula sem postar no Jira")

    # simulate-docker
    p_doc = subparsers.add_parser("simulate-docker", help="Simula falha em container Docker local")
    p_doc.add_argument("--container", default="flowti-app", help="Nome do container local")
    p_doc.add_argument("--type", default="CrashLoopBackOff", choices=["CrashLoopBackOff", "OOMKilled", "FailedHealthCheck"], help="Tipo da falha")
    p_doc.add_argument("--exit-code", type=int, default=1, help="Exit Code do container")
    p_doc.add_argument("--restarts", type=int, default=4, help="Número de reinicializações")
    p_doc.add_argument("--parent", default="OPS-236", help="Issue de referência/pai para vincular")
    p_doc.add_argument("--contrato", default=None, help="Força contrato FLOWTI (ex: INTERNO, DENTALIS, MAIDA (GCP))")
    p_doc.add_argument("--dry-run", action="store_true", help="Gera análise e template sem postar no Jira")

    # watch (Modo Observador Contínuo Local)
    p_wt = subparsers.add_parser("watch", help="Observador contínuo em background de containers locais (Docker)")
    p_wt.add_argument("--socket", default="/var/run/docker.sock", help="Caminho do socket Docker (somente-leitura)")
    p_wt.add_argument("--interval", type=int, default=15, help="Intervalo de varredura em segundos")
    p_wt.add_argument("--targets", default="flowti-app,flowti-mysql,flowti-phpmyadmin", help="Containers alvo separados por vírgula")
    p_wt.add_argument("--webhook-port", type=int, default=8080, help="Porta para subir o listener de webhooks em paralelo (0 desativa)")
    p_wt.add_argument("--parent", default="OPS-236", help="Issue de referência/pai para vincular")
    p_wt.add_argument("--dry-run", action="store_true", help="Modo simulação sem postar no Jira")

    args = parser.parse_args()

    if not args.command:
        parser.print_help()
        sys.exit(0)

    if args.command == "test-jira":
        cmd_test_jira(args)
    elif args.command == "simulate-pod":
        cmd_simulate_pod(args)
    elif args.command == "simulate-pipeline":
        cmd_simulate_pipeline(args)
    elif args.command == "simulate-docker":
        cmd_simulate_docker(args)
    elif args.command == "scan-k8s":
        cmd_scan_k8s(args)
    elif args.command == "webhook":
        cmd_webhook(args)
    elif args.command == "watch":
        cmd_watch(args)

if __name__ == "__main__":
    main()
