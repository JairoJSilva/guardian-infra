import json
import time
from pathlib import Path
from typing import Optional, Dict, Any, Tuple
import requests

from guardian_ops.config import Config
from guardian_ops.models import AnalysisResult, IncidentEvent

class JiraClient:
    def __init__(self):
        self.base_url = Config.JIRA_BASE_URL
        self.user = Config.JIRA_USER
        self.password = Config.JIRA_PASSWORD
        self.cache_file = Config.CACHE_FILE
        self.cooldown_seconds = Config.COOLDOWN_MINUTES * 60
        self._load_cache()

    def _load_cache(self):
        """Carrega cache de incidentes para deduplicação (anti-spam de chamados)."""
        self.cache: Dict[str, float] = {}
        if self.cache_file.exists():
            try:
                with open(self.cache_file, "r", encoding="utf-8") as f:
                    self.cache = json.load(f)
            except Exception as e:
                print(f"[GuardianOps] Aviso: Falha ao ler cache ({e}). Reiniciando em memória.")
                self.cache = {}

    def _save_cache(self):
        """Persiste cache de deduplicação em arquivo JSON."""
        try:
            # Limpa entradas expiradas
            now = time.time()
            self.cache = {k: ts for k, ts in self.cache.items() if now - ts < self.cooldown_seconds}
            with open(self.cache_file, "w", encoding="utf-8") as f:
                json.dump(self.cache, f, ensure_ascii=False, indent=2)
        except Exception as e:
            print(f"[GuardianOps] Aviso: Falha ao salvar cache de deduplicação: {e}")

    def is_in_cooldown(self, fingerprint: str) -> bool:
        """Verifica se um incidente com o mesmo fingerprint foi aberto recentemente."""
        if fingerprint in self.cache:
            elapsed = time.time() - self.cache[fingerprint]
            if elapsed < self.cooldown_seconds:
                remaining_mins = int((self.cooldown_seconds - elapsed) / 60)
                print(f"[GuardianOps] ⏳ Cooldown ativo para fingerprint {fingerprint[:8]}... (restam ~{remaining_mins} min). Chamado duplicado suprimido.")
                return True
        return False

    def record_incident(self, fingerprint: str):
        """Registra timestamp do incidente para evitar duplicidade dentro da janela de cooldown."""
        self.cache[fingerprint] = time.time()
        self._save_cache()

    def _fetch_parent_contract(self, parent_key: str) -> Optional[Dict[str, str]]:
        """Consulta o chamado pai no Jira para herdar o campo de Contrato FLOWTI."""
        if not self.user or not self.password:
            return None
        try:
            url = f"{self.base_url}/rest/api/2/issue/{parent_key}?fields={Config.JIRA_CUSTOMFIELD_CONTRATO_ID}"
            res = requests.get(url, auth=(self.user, self.password), timeout=10)
            if res.status_code == 200:
                fields = res.json().get("fields", {})
                contrato_obj = fields.get(Config.JIRA_CUSTOMFIELD_CONTRATO_ID)
                if contrato_obj and isinstance(contrato_obj, dict) and "id" in contrato_obj and "value" in contrato_obj:
                    return {"id": contrato_obj["id"], "value": contrato_obj["value"]}
        except Exception as e:
            print(f"[GuardianOps] Aviso: Falha ao inspecionar contrato do chamado pai {parent_key}: {e}")
        return None

    def create_incident_issue(
        self,
        event: IncidentEvent,
        analysis: AnalysisResult,
        parent_issue_key: Optional[str] = None
    ) -> Optional[Dict[str, Any]]:
        """
        Cria chamado no Jira utilizando a API REST v2 com garantia estrita de UTF-8.
        Respeita cooldown e regras de integridade do projeto OPS.
        """
        fingerprint = event.fingerprint
        if self.is_in_cooldown(fingerprint):
            return None

        # Prepara campos do chamado
        fields: Dict[str, Any] = {
            "project": {"key": Config.JIRA_PROJECT_KEY},
            "summary": analysis.summary,
            "description": analysis.jira_description,
            "issuetype": {"name": analysis.issue_type},
            "priority": {"name": analysis.priority.value},
            "labels": analysis.labels,
        }

        if self.user:
            fields["reporter"] = {"name": self.user}

        # Resolução inteligente de Contrato FLOWTI:
        # 1. Contrato explícito no resultado da análise
        # 2. Contrato explícito nos detalhes do evento (ex: label de Pod ou flag CLI)
        # 3. Herança do chamado pai (se houver parent_issue_key)
        # 4. Mapeamento heurístico por namespace, projeto ou identificador
        # 5. Fallback padrão: INTERNO (ID 22514)
        contract = None
        if analysis.contract_field:
            contract = analysis.contract_field
        elif event.details.get("contrato"):
            contract = Config.resolve_contract(event.details["contrato"])
        elif parent_issue_key and not Config.DRY_RUN:
            contract = self._fetch_parent_contract(parent_issue_key)

        if not contract:
            context_hint = f"{event.namespace_or_project} {event.identifier}"
            contract = Config.resolve_contract(context_hint)

        if Config.JIRA_CUSTOMFIELD_CONTRATO_ID and contract:
            fields[Config.JIRA_CUSTOMFIELD_CONTRATO_ID] = contract

        payload = {"fields": fields}

        if Config.DRY_RUN:
            print("\n" + "=" * 60)
            print("[GuardianOps DRY-RUN] Simulação de Abertura de Chamado:")
            print("=" * 60)
            print(f"Projeto   : {Config.JIRA_PROJECT_KEY}")
            print(f"Título    : {analysis.summary}")
            print(f"Prioridade: {analysis.priority.value}")
            print(f"Tipo      : {analysis.issue_type}")
            print(f"Contrato  : {contract.get('value')} (ID: {contract.get('id')})")
            print(f"Labels    : {analysis.labels}")
            print("-" * 60)
            print("DESCRIÇÃO FORMATADA (Jira Markup - UTF-8):")
            print(analysis.jira_description)
            print("=" * 60)
            self.record_incident(fingerprint)
            return {"key": "SIMULATED-OPS-001", "dry_run": True}

        if not self.user or not self.password:
            raise ValueError("JIRA_USER e JIRA_PASSWORD devem estar configurados no arquivo .env")

        # Serialização com garantia estrita de UTF-8 (ensure_ascii=False)
        payload_bytes = json.dumps(payload, ensure_ascii=False).encode("utf-8")

        headers = {
            "Content-Type": "application/json; charset=utf-8",
            "Accept": "application/json"
        }

        url = f"{self.base_url}/rest/api/2/issue"
        print(f"[GuardianOps] 🛡️ Enviando chamado para o Jira: '{analysis.summary}' ...")

        response = requests.post(
            url,
            data=payload_bytes,
            auth=(self.user, self.password),
            headers=headers,
            timeout=30
        )

        if response.status_code == 201:
            data = response.json()
            issue_key = data.get("key")
            issue_url = f"{self.base_url}/browse/{issue_key}"
            print("\n" + "=" * 60)
            print(f" [✓] GUARDIANOPS: CHAMADO CRIADO COM SUCESSO NO JIRA!")
            print(f" Chave : {issue_key}")
            print(f" Link  : {issue_url}")
            print("=" * 60)

            # Se houver issue pai informada, realiza o vínculo
            if parent_issue_key:
                self.link_issue(issue_key, parent_issue_key)

            self.record_incident(fingerprint)
            return data
        else:
            print(f"\n[GuardianOps] ❌ Falha ao criar chamado no Jira (HTTP {response.status_code}):")
            try:
                err_data = response.json()
                errors = err_data.get("errors", {})
                error_messages = err_data.get("errorMessages", [])
                for k, v in errors.items():
                    print(f"    - Campo '{k}': {v}")
                for msg in error_messages:
                    print(f"    - {msg}")
            except Exception:
                print(response.text)
            return None

    def link_issue(self, inward_key: str, outward_key: str, link_type: str = "Relates"):
        """Vincula o chamado gerado a um chamado de referência/pai."""
        url = f"{self.base_url}/rest/api/2/issueLink"
        payload = {
            "type": {"name": link_type},
            "inwardIssue": {"key": inward_key},
            "outwardIssue": {"key": outward_key},
            "comment": {
                "body": f"Vinculado automaticamente pelo GuardianOps à referência {outward_key}"
            }
        }
        payload_bytes = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        headers = {
            "Content-Type": "application/json; charset=utf-8",
            "Accept": "application/json"
        }
        res = requests.post(
            url,
            data=payload_bytes,
            auth=(self.user, self.password),
            headers=headers,
            timeout=15
        )
        if res.status_code in [200, 201, 204]:
            print(f"  [+] Chamado {inward_key} vinculado ao pai {outward_key} com sucesso!")
        else:
            print(f"  [!] Aviso: Não foi possível vincular a {outward_key}: HTTP {res.status_code}")
