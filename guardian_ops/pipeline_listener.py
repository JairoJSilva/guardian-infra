import json
from http.server import HTTPServer, BaseHTTPRequestHandler
from typing import Callable, Optional
from guardian_ops.models import IncidentEvent, FailureSource

class PipelineMock:
    @staticmethod
    def create_mock_pipeline_failure(
        project: str = "acme/portal-app",
        pipeline_id: str = "48921",
        stage: str = "docker-build",
        error_type: str = "JobFailed"
    ) -> IncidentEvent:
        """Cria evento simulado de falha de Pipeline CI/CD para validação."""
        logs = """[00:01:23] $ docker build -t registry.example.com/portal-app:v2.4.1 -f Dockerfile .
[00:01:25] Step 1/12 : FROM node:20-alpine AS builder
[00:01:28] Step 2/12 : WORKDIR /app
[00:01:29] Step 3/12 : COPY package.json package-lock.json ./
[00:01:31] Step 4/12 : RUN npm ci
[00:01:45] npm ERR! code ERESOLVE
[00:01:45] npm ERR! ERESOLVE could not resolve dependency:
[00:01:45] npm ERR! peer @types/react@"^18.0.0" from @acme/ui-components@3.1.0
[00:01:45] npm ERR! Conflicting dependency: react@19.0.0
[00:01:46] ERROR: Service 'builder' failed to build: The command '/bin/sh -c npm ci' returned a non-zero code: 1
[00:01:46] Cleaning up project directory and file based variables
[00:01:47] ERROR: Job failed: exit status 1"""

        return IncidentEvent(
            source=FailureSource.PIPELINE_CI_CD,
            identifier=pipeline_id,
            namespace_or_project=project,
            failure_type="DockerBuildConflict (ERESOLVE)",
            container_or_stage=stage,
            error_message="Job 'docker-build' falhou com exit code 1 durante a execução do npm ci no Dockerfile",
            logs_snippet=logs,
            exit_code=1,
            environment="CI/CD Runner / Staging",
            details={
                "commit_ref": "main",
                "author": "developer@example.com",
                "pipeline_url": f"https://gitlab.com/{project}/-/pipelines/{pipeline_id}"
            }
        )

def make_webhook_handler(on_incident_callback: Callable[[IncidentEvent], None]):
    class GuardianWebhookHandler(BaseHTTPRequestHandler):
        def _send_json_response(self, code: int, data: dict):
            body = json.dumps(data, ensure_ascii=False).encode("utf-8")
            self.send_response(code)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def do_GET(self):
            if self.path == "/healthz":
                self._send_json_response(200, {"status": "ok", "service": "GuardianOps Webhook Listener", "mode": "read-only"})
            else:
                self._send_json_response(404, {"error": "Not Found"})

        def do_POST(self):
            content_length = int(self.headers.get("Content-Length", 0))
            raw_body = self.rfile.read(content_length)

            try:
                payload = json.loads(raw_body.decode("utf-8"))
            except Exception as e:
                self._send_json_response(400, {"error": f"JSON inválido: {str(e)}"})
                return

            # Suporte a webhook do GitLab CI ou webhook genérico
            event = self._parse_payload(payload)
            if not event:
                self._send_json_response(200, {"status": "ignored", "message": "Evento recebido não é uma falha observável"})
                return

            # Dispara análise e criação de chamado
            try:
                on_incident_callback(event)
                self._send_json_response(200, {
                    "status": "processed",
                    "incident": event.identifier,
                    "target": event.namespace_or_project
                })
            except Exception as e:
                self._send_json_response(500, {"error": f"Erro no processamento do incidente: {str(e)}"})

        def _parse_payload(self, payload: dict) -> Optional[IncidentEvent]:
            object_kind = payload.get("object_kind")

            # Caso 1: GitLab Pipeline Hook
            if object_kind == "pipeline":
                obj_attrs = payload.get("object_attributes", {})
                status = obj_attrs.get("status")
                if status in ["failed", "canceled"]:
                    project_name = payload.get("project", {}).get("path_with_namespace", "unknown-project")
                    pipeline_id = str(obj_attrs.get("id", "0"))
                    builds = payload.get("builds", [])
                    failed_stages = [b.get("stage", "build") for b in builds if b.get("status") == "failed"]
                    failed_stage = failed_stages[0] if failed_stages else "unknown-stage"

                    return IncidentEvent(
                        source=FailureSource.PIPELINE_CI_CD,
                        identifier=pipeline_id,
                        namespace_or_project=project_name,
                        failure_type=f"Pipeline_{status.upper()}",
                        container_or_stage=failed_stage,
                        error_message=f"Pipeline {pipeline_id} finalizada com status '{status}' no estágio '{failed_stage}'",
                        logs_snippet="Consulte a URL da pipeline no GitLab para os logs detalhados do runner.",
                        environment="GitLab CI / Produção"
                    )

            # Caso 2: Payload Genérico de Alerta
            if "alert" in payload or "source" in payload or "failure_type" in payload:
                src_str = payload.get("source", "KUBERNETES_POD")
                src = FailureSource.PIPELINE_CI_CD if "PIPELINE" in src_str.upper() else FailureSource.KUBERNETES_POD
                return IncidentEvent(
                    source=src,
                    identifier=payload.get("identifier", "target-app"),
                    namespace_or_project=payload.get("namespace_or_project", "default"),
                    failure_type=payload.get("failure_type", "OperationalError"),
                    container_or_stage=payload.get("container_or_stage", "main"),
                    error_message=payload.get("error_message", "Falha reportada via webhook"),
                    logs_snippet=payload.get("logs_snippet", ""),
                    exit_code=payload.get("exit_code"),
                    restart_count=payload.get("restart_count")
                )

            return None

    return GuardianWebhookHandler

def run_webhook_server(host: str, port: int, callback: Callable[[IncidentEvent], None]):
    handler_cls = make_webhook_handler(callback)
    server = HTTPServer((host, port), handler_cls)
    print(f"[GuardianOps] 🛡️ Webhook Server ativo em http://{host}:{port}")
    print(f"  - Healthcheck: http://{host}:{port}/healthz")
    print(f"  - Endpoint para Webhook GitLab CI: POST http://{host}:{port}/webhook/gitlab")
    print("  - Pressione Ctrl+C para encerrar o listener.")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n[GuardianOps] Encerrando Webhook Server...")
        server.server_close()
