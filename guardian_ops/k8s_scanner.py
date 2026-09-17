import json
import subprocess
from typing import List, Optional
from guardian_ops.models import IncidentEvent, FailureSource

class K8sScanner:
    """
    Scanner de Pods e Eventos do Kubernetes.
    REGRA DE SEGURANÇA MANDATÓRIA:
    Apenas comandos e chamadas de SOMENTE-LEITURA ('get', 'describe', 'logs').
    Nenhum comando de modificação, exclusão ou mutação é executado.
    """

    def __init__(self, namespaces: Optional[List[str]] = None):
        self.namespaces = namespaces or ["default"]

    def scan_cluster_pods(self) -> List[IncidentEvent]:
        """
        Inspeciona pods nos namespaces configurados buscando estados anômalos
        (CrashLoopBackOff, OOMKilled, ImagePullBackOff, Error, etc.).
        """
        incidents: List[IncidentEvent] = []
        for ns in self.namespaces:
            cmd = ["kubectl", "get", "pods", "-n", ns, "-o", "json"]
            try:
                result = subprocess.run(cmd, capture_output=True, text=True, timeout=15)
                if result.returncode != 0:
                    continue
                data = json.loads(result.stdout)
                for item in data.get("items", []):
                    incident = self._evaluate_pod(item, ns)
                    if incident:
                        incidents.append(incident)
            except (subprocess.SubprocessError, json.JSONDecodeError, FileNotFoundError):
                # Caso kubectl não esteja configurado ou sem conexão
                pass
        return incidents

    def _evaluate_pod(self, pod: dict, namespace: str) -> Optional[IncidentEvent]:
        metadata = pod.get("metadata", {})
        pod_name = metadata.get("name", "unknown-pod")
        status = pod.get("status", {})
        phase = status.get("phase", "")

        container_statuses = status.get("containerStatuses", [])
        for c_status in container_statuses:
            c_name = c_status.get("name", "app")
            restarts = c_status.get("restartCount", 0)
            state = c_status.get("state", {})

            # 1. Checa estado Waiting (CrashLoopBackOff, ImagePullBackOff, ErrImagePull)
            waiting = state.get("waiting")
            if waiting:
                reason = waiting.get("reason", "")
                message = waiting.get("message", "")
                if reason in ["CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull", "CreateContainerConfigError"]:
                    logs = self._get_pod_logs(pod_name, namespace, c_name)
                    return IncidentEvent(
                        source=FailureSource.KUBERNETES_POD,
                        identifier=pod_name,
                        namespace_or_project=namespace,
                        failure_type=reason,
                        container_or_stage=c_name,
                        error_message=message,
                        logs_snippet=logs,
                        restart_count=restarts,
                        details={"pod_phase": phase, "state": "waiting"}
                    )

            # 2. Checa estado Terminated (OOMKilled, Error)
            terminated = state.get("terminated")
            if terminated:
                reason = terminated.get("reason", "")
                exit_code = terminated.get("exitCode", 0)
                if exit_code != 0 or reason in ["OOMKilled", "Error"]:
                    logs = self._get_pod_logs(pod_name, namespace, c_name)
                    return IncidentEvent(
                        source=FailureSource.KUBERNETES_POD,
                        identifier=pod_name,
                        namespace_or_project=namespace,
                        failure_type=reason or f"ExitCode_{exit_code}",
                        container_or_stage=c_name,
                        exit_code=exit_code,
                        error_message=terminated.get("message", f"Container terminado com Exit Code {exit_code}"),
                        logs_snippet=logs,
                        restart_count=restarts,
                        details={"pod_phase": phase, "state": "terminated"}
                    )
        return None

    def _get_pod_logs(self, pod_name: str, namespace: str, container: str) -> str:
        """Coleta logs recentes (STDOUT/STDERR) com segurança."""
        cmd = ["kubectl", "logs", pod_name, "-n", namespace, "-c", container, "--tail=50"]
        try:
            res = subprocess.run(cmd, capture_output=True, text=True, timeout=10)
            if res.returncode == 0 and res.stdout.strip():
                return res.stdout.strip()
            # Tenta --previous se o container acabou de reiniciar
            cmd_prev = ["kubectl", "logs", pod_name, "-n", namespace, "-c", container, "--previous", "--tail=50"]
            res_prev = subprocess.run(cmd_prev, capture_output=True, text=True, timeout=10)
            if res_prev.returncode == 0 and res_prev.stdout.strip():
                return res_prev.stdout.strip()
        except Exception:
            pass
        return "Nenhum log disponível via kubectl."

    @staticmethod
    def create_mock_pod_failure(
        pod_name: str = "portal-flowti-backend-69b7bf656b-x82d9",
        namespace: str = "producao",
        failure_type: str = "CrashLoopBackOff",
        container: str = "flowti-api",
        exit_code: int = 1,
        restarts: int = 8
    ) -> IncidentEvent:
        """Cria evento simulado de falha de Pod para testes locais e validação."""
        logs = """[2026-09-17 11:20:01] INFO  [Server] Starting application server on port 8080...
[2026-09-17 11:20:03] INFO  [Config] Loading environment variables from Secret 'flowti-db-credentials'...
[2026-09-17 11:20:04] ERROR [DatabasePool] Connection to PostgreSQL at 'db-prod.internal:5432' failed: Connection timed out.
[2026-09-17 11:20:05] FATAL [Main] Application failed to initialize required database connection pool.
org.postgresql.util.PSQLException: The connection attempt timed out after 5000ms.
    at org.postgresql.core.v3.ConnectionFactoryImpl.openConnectionImpl(ConnectionFactoryImpl.java:319)
    at org.postgresql.core.ConnectionFactory.openConnection(ConnectionFactoryImpl.java:54)
    at com.flowti.core.Application.main(Application.java:42)
[2026-09-17 11:20:05] SYSTEM [Process] Exited with status code 1."""

        return IncidentEvent(
            source=FailureSource.KUBERNETES_POD,
            identifier=pod_name,
            namespace_or_project=namespace,
            failure_type=failure_type,
            container_or_stage=container,
            error_message="Back-off 5m0s restarting failed container=flowti-api pod=portal-flowti-backend-69b7bf656b-x82d9",
            logs_snippet=logs,
            exit_code=exit_code,
            restart_count=restarts,
            environment="AWS Produção (EKS)"
        )
