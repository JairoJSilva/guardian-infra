import os
import json
import socket
import http.client
import subprocess
from typing import List, Optional, Dict, Any
from guardian_ops.models import IncidentEvent, FailureSource

class UnixSocketHTTPConnection(http.client.HTTPConnection):
    """Conexão HTTP sobre Unix Domain Socket para comunicação direta com a Docker Engine API."""
    def __init__(self, socket_path: str = "/var/run/docker.sock", timeout: int = 5):
        super().__init__("localhost", timeout=timeout)
        self.socket_path = socket_path

    def connect(self):
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.sock.settimeout(self.timeout)
        self.sock.connect(self.socket_path)

class DockerScanner:
    """
    Scanner de Containers Docker Locais.
    DIRETRIZ DE SEGURANÇA MANDATÓRIA:
    Opera em modo estritamente SOMENTE-LEITURA (Read-Only) via /var/run/docker.sock.
    Apenas requisições GET para inspeção de status, saúde e logs de containers.
    Nenhuma ação de mutação, parada, reinicialização ou exclusão é executada.
    """

    def __init__(self, socket_path: str = "/var/run/docker.sock", target_names: Optional[List[str]] = None):
        self.socket_path = socket_path
        self.target_names = target_names  # Ex: ["flowti-app", "flowti-mysql", "flowti-phpmyadmin"]
        self.previous_states: Dict[str, Dict[str, Any]] = {}

    def is_available(self) -> bool:
        """Verifica se o socket do Docker está acessível no ambiente."""
        return os.path.exists(self.socket_path)

    def _query_docker_api(self, path: str) -> Optional[Any]:
        """Executa requisição GET na API Docker via socket unix."""
        if not self.is_available():
            return None
        try:
            conn = UnixSocketHTTPConnection(self.socket_path, timeout=5)
            conn.request("GET", path)
            response = conn.getresponse()
            if response.status == 200:
                body = response.read().decode("utf-8", errors="replace")
                conn.close()
                return json.loads(body)
            conn.close()
        except Exception:
            pass
        return None

    def _get_container_logs_api(self, container_id: str, tail: int = 50) -> str:
        """Coleta logs do container via Docker API."""
        try:
            conn = UnixSocketHTTPConnection(self.socket_path, timeout=5)
            conn.request("GET", f"/containers/{container_id}/logs?stdout=1&stderr=1&tail={tail}")
            response = conn.getresponse()
            if response.status == 200:
                raw = response.read()
                conn.close()
                # Remove os cabeçalhos de 8 bytes de stream do docker se presentes
                lines = []
                idx = 0
                while idx < len(raw):
                    if idx + 8 <= len(raw):
                        stream_type = raw[idx]
                        size = int.from_bytes(raw[idx+4:idx+8], byteorder="big")
                        idx += 8
                        line = raw[idx:idx+size].decode("utf-8", errors="replace")
                        lines.append(line.strip())
                        idx += size
                    else:
                        break
                return "\n".join(lines[-tail:]) if lines else raw.decode("utf-8", errors="replace")
            conn.close()
        except Exception:
            pass
        return ""

    def list_containers(self) -> List[Dict[str, Any]]:
        """Retorna a lista de containers ativos/monitorados."""
        data = self._query_docker_api("/containers/json?all=1")
        if data is not None and isinstance(data, list):
            containers = []
            for item in data:
                raw_names = item.get("Names", [])
                clean_names = [n.lstrip("/") for n in raw_names]
                name = clean_names[0] if clean_names else item.get("Id", "")[:12]
                
                # Filtro por targets se especificado
                if self.target_names:
                    if not any(t in name for t in self.target_names):
                        continue

                containers.append({
                    "id": item.get("Id"),
                    "name": name,
                    "image": item.get("Image"),
                    "state": item.get("State"),
                    "status": item.get("Status"),
                })
            return containers
        return []

    def scan_containers(self) -> List[IncidentEvent]:
        """
        Inspeciona containers Docker locais buscando falhas, crash loops,
        estouros de memória (OOM) ou probes de saúde com falha.
        """
        incidents: List[IncidentEvent] = []
        containers = self.list_containers()

        for c in containers:
            c_id = c["id"]
            name = c["name"]
            
            # Obtém detalhes completos do container
            details = self._query_docker_api(f"/containers/{c_id}/json")
            if not details:
                continue

            state = details.get("State", {})
            status = state.get("Status", "")
            restarting = state.get("Restarting", False)
            oom_killed = state.get("OOMKilled", False)
            exit_code = state.get("ExitCode", 0)
            restart_count = details.get("RestartCount", 0)
            health = state.get("Health", {})
            health_status = health.get("Status", "")

            # Compara com estado anterior para evitar duplicatas em tempo real
            prev = self.previous_states.get(c_id, {})
            prev_restarts = prev.get("restart_count", 0)
            prev_status = prev.get("status", "")

            # Atualiza histórico
            self.previous_states[c_id] = {
                "status": status,
                "restart_count": restart_count,
                "exit_code": exit_code,
                "health_status": health_status,
            }

            anomaly_reason = None
            failure_type = None

            # 1. Caso OOMKilled
            if oom_killed or exit_code == 137:
                anomaly_reason = f"Container '{name}' foi encerrado por esgotamento de memória (OOMKilled - Exit Code 137)."
                failure_type = "OOMKilled"

            # 2. Caso Restarting / CrashLoop
            elif restarting or (restart_count > prev_restarts and exit_code != 0):
                anomaly_reason = f"Container '{name}' em loop de reinicialização (Exit Code: {exit_code}, Restarts: {restart_count})."
                failure_type = "CrashLoopBackOff"

            # 3. Caso Exited com erro inesperado
            elif status == "exited" and exit_code != 0 and prev_status != "exited":
                anomaly_reason = f"Container '{name}' encerrou inesperadamente com código de erro {exit_code}."
                failure_type = f"ExitCode_{exit_code}"

            # 4. Caso Healthcheck Unhealthy
            elif health_status == "unhealthy" and prev.get("health_status") != "unhealthy":
                last_log = ""
                log_entries = health.get("Log", [])
                if log_entries:
                    last_log = log_entries[-1].get("Output", "").strip()
                anomaly_reason = f"Healthcheck do container '{name}' falhou: status 'unhealthy'. {last_log}"
                failure_type = "FailedHealthCheck"

            if failure_type:
                logs = self._get_container_logs_api(c_id, tail=50)
                image_name = details.get("Config", {}).get("Image", "unknown-image")

                incident = IncidentEvent(
                    source=FailureSource.DOCKER_CONTAINER,
                    identifier=name,
                    namespace_or_project="local-docker",
                    failure_type=failure_type,
                    environment="Ambiente Local (Docker)",
                    container_or_stage=name,
                    error_message=anomaly_reason or f"Falha detectada no container {name}",
                    logs_snippet=logs,
                    exit_code=exit_code,
                    restart_count=restart_count,
                    details={
                        "container_id": c_id[:12],
                        "image": image_name,
                        "health_status": health_status,
                        "oom_killed": oom_killed,
                    }
                )
                incidents.append(incident)

        return incidents

    @staticmethod
    def create_mock_docker_failure(
        container_name: str = "flowti-app",
        failure_type: str = "CrashLoopBackOff",
        exit_code: int = 1,
        restarts: int = 4
    ) -> IncidentEvent:
        """Gera mock de falha de container Docker local para testes."""
        logs = """[Thu Sep 17 16:20:01 2026] [php:error] [pid 18] PHP Fatal error: Uncaught PDOException: SQLSTATE[HY000] [2002] Connection refused in /var/www/html/src/Config/Database.php:32
Stack trace:
#0 /var/www/html/src/Config/Database.php(32): PDO->__construct()
#1 /var/www/html/src/bootstrap.php(19): App\\Config\\Database::getConnection()
#2 /var/www/html/public/index.php(13): require_once('/var/www/html/s...')
#3 {main}
  thrown in /var/www/html/src/Config/Database.php on line 32
AH00052: child pid 18 exit signal Segmentation fault (11)"""

        return IncidentEvent(
            source=FailureSource.DOCKER_CONTAINER,
            identifier=container_name,
            namespace_or_project="local-docker",
            failure_type=failure_type,
            environment="Ambiente Local (Docker)",
            container_or_stage=container_name,
            error_message=f"Container '{container_name}' falhou com exit code {exit_code} ({failure_type})",
            logs_snippet=logs,
            exit_code=exit_code,
            restart_count=restarts,
            details={"image": "flowti-hub-app", "container_id": "a1b2c3d4e5f6"}
        )
