#!/usr/bin/env python3
import os
import sys
import argparse
from pathlib import Path
from urllib.parse import urlparse
from dotenv import load_dotenv
import requests

# Carrega configurações do .env
env_path = Path(__file__).parent / ".env"
load_dotenv(dotenv_path=env_path)

raw_url = os.getenv("JIRA_BASE_URL", "https://jira.example.com").strip()
parsed = urlparse(raw_url)
JIRA_BASE_URL = f"{parsed.scheme}://{parsed.netloc}" if parsed.netloc else raw_url.rstrip("/")
JIRA_USER = os.getenv("JIRA_USER", "").strip()
JIRA_PASSWORD = os.getenv("JIRA_PASSWORD", "").strip()

def create_issue(summary: str, description: str = "", project_key: str = "OPS", issue_type: str = "Incident", contrato: str = None):
    if not JIRA_USER or not JIRA_PASSWORD:
        print("[!] ERRO: JIRA_USER ou JIRA_PASSWORD não configurados no .env")
        sys.exit(1)

    url = f"{JIRA_BASE_URL}/rest/api/2/issue"

    fields = {
        "project": {"key": project_key},
        "summary": summary,
        "description": description,
        "issuetype": {"name": issue_type},
        "reporter": {"name": JIRA_USER}
    }

    # Contrato FLOWTI (customfield_30118) - padrão INTERNO (22514)
    contrato_map = {
        "INTERNO": {"id": "22514", "value": "INTERNO"},
        "DENTALIS": {"id": "22300", "value": "DENTALIS"},
        "DENTALIS (SNOW FLAKE)": {"id": "22500", "value": "DENTALIS (SNOW FLAKE)"},
        "FARMACIA DIGITAL (AWS)": {"id": "22502", "value": "FARMACIA DIGITAL (AWS)"},
        "GIF": {"id": "22517", "value": "GIF"},
        "MAIDA (DIGITAL OCEAN AKAMAI)": {"id": "22303", "value": "MAIDA (DIGITAL OCEAN AKAMAI)"},
        "MAIDA (GCP)": {"id": "22301", "value": "MAIDA (GCP)"},
        "MAIDA (LEGADO AWS)": {"id": "22302", "value": "MAIDA (LEGADO AWS)"},
        "MAIDA BI CLOUDOPS (AZURE/OCI)": {"id": "22501", "value": "MAIDA BI CLOUDOPS (AZURE/OCI)"},
    }
    target_contrato = (contrato or "INTERNO").upper()
    if target_contrato in contrato_map:
        fields["customfield_30118"] = contrato_map[target_contrato]
    else:
        fields["customfield_30118"] = contrato_map["INTERNO"]

    payload = {"fields": fields}

    print(f"[*] Criando chamado no projeto '{project_key}'...")
    print(f"    Resumo: {summary}")

    response = requests.post(
        url,
        json=payload,
        auth=(JIRA_USER, JIRA_PASSWORD),
        headers={"Content-Type": "application/json", "Accept": "application/json"},
        timeout=20
    )

    if response.status_code == 201:
        data = response.json()
        issue_key = data.get("key")
        issue_url = f"{JIRA_BASE_URL}/browse/{issue_key}"
        print("\n" + "=" * 50)
        print(f">>> CHAMADO CRIADO COM SUCESSO! <<<")
        print("=" * 50)
        print(f"Chave  : {issue_key}")
        print(f"Link   : {issue_url}")
        print("=" * 50)
        return data
    else:
        print(f"\n[!] Falha ao criar chamado (HTTP {response.status_code}):")
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
        sys.exit(1)

def main():
    parser = argparse.ArgumentParser(description="Criar chamado no Jira via API")
    parser.add_argument("-s", "--summary", required=False, help="Título / Resumo do chamado")
    parser.add_argument("-d", "--description", default="", help="Descrição detalhada do chamado")
    parser.add_argument("-p", "--project", default="OPS", help="Chave do projeto (padrão: OPS)")
    parser.add_argument("-t", "--type", default="Solicitação de serviço", help="Tipo do chamado (padrão: Solicitação de serviço)")
    parser.add_argument("-c", "--contrato", default=None, help="Contrato FLOWTI (ex: DENTALIS)")

    args = parser.parse_args()

    summary = args.summary
    if not summary:
        summary = input("Digite o título/resumo do chamado: ").strip()
        if not summary:
            print("[!] Resumo não pode ser vazio.")
            sys.exit(1)

    description = args.description
    if not description:
        description = input("Digite a descrição (opcional, pressione Enter para pular): ").strip()

    create_issue(
        summary=summary,
        description=description,
        project_key=args.project,
        issue_type=args.type,
        contrato=args.contrato
    )

if __name__ == "__main__":
    main()
