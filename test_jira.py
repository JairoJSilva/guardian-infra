#!/usr/bin/env python3
import os
import sys
from pathlib import Path
from dotenv import load_dotenv
import requests

from urllib.parse import urlparse

# Carrega variáveis do arquivo .env no diretório do script
env_path = Path(__file__).parent / ".env"
load_dotenv(dotenv_path=env_path)

raw_url = os.getenv("JIRA_BASE_URL", "https://jira.mv.com.br").strip()
parsed = urlparse(raw_url)
JIRA_BASE_URL = f"{parsed.scheme}://{parsed.netloc}" if parsed.netloc else raw_url.rstrip("/")
JIRA_USER = os.getenv("JIRA_USER", "").strip()
JIRA_PASSWORD = os.getenv("JIRA_PASSWORD", "").strip()

def main():
    print(f"[*] Conectando a: {JIRA_BASE_URL}")
    
    if not JIRA_USER or not JIRA_PASSWORD:
        print("\n[!] ERRO: 'JIRA_USER' ou 'JIRA_PASSWORD' não estão configurados no arquivo .env!")
        print(f"Por favor, edite o arquivo:\n  {env_path}")
        print("e preencha seu usuário e senha.")
        sys.exit(1)

    print(f"[*] Testando autenticação básica para o usuário: {JIRA_USER} ...")

    endpoint = f"{JIRA_BASE_URL}/rest/api/2/myself"
    headers = {
        "Accept": "application/json"
    }

    try:
        response = requests.get(
            endpoint,
            auth=(JIRA_USER, JIRA_PASSWORD),
            headers=headers,
            timeout=15
        )

        if response.status_code == 200:
            data = response.json()
            print("\n" + "=" * 50)
            print(">>> SUCESSO NA AUTENTICAÇÃO! <<<")
            print("=" * 50)
            print(f"Nome completo : {data.get('displayName')}")
            print(f"Usuário (login): {data.get('name')}")
            print(f"E-mail        : {data.get('emailAddress', 'N/A')}")
            print(f"Fuso horário  : {data.get('timeZone', 'N/A')}")
            print(f"Conta ativa   : {'Sim' if data.get('active') else 'Não'}")
            print("=" * 50)
            print("\nA API do Jira está pronta para ser usada nas automações!")
        elif response.status_code == 401:
            print("\n[!] Erro 401 (Não autorizado): Usuário ou senha incorretos no .env.")
        elif response.status_code == 403:
            print("\n[!] Erro 403 (Proibido): Acesso bloqueado ou captcha ativo. Tente logar no navegador.")
        else:
            print(f"\n[!] Resposta inesperada do Jira: HTTP {response.status_code}")
            print(response.text[:500])

    except requests.exceptions.RequestException as e:
        print(f"\n[!] Falha de conexão com o Jira: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
