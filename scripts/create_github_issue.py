#!/usr/bin/env python3
"""
Utilitário para criação de issues no repositório GitHub JairoJSilva/guardian-infra via REST API.
Uso:
  python3 scripts/create_github_issue.py --token <SEU_GITHUB_TOKEN>
Ou definindo a variável de ambiente GITHUB_TOKEN.
"""

import os
import sys
import argparse
from pathlib import Path
from dotenv import load_dotenv
import requests

# Carrega .env do projeto se existir
load_dotenv(Path(__file__).resolve().parent.parent / ".env")
load_dotenv()

DEFAULT_REPO = "JairoJSilva/guardian-infra"
DEFAULT_TITLE = "[Feature] Parametrização via Frontend de Múltiplos Serviços de Chamados (Jira, GLPI e Movidesk)"

DEFAULT_BODY = """## 🎯 Proposta / Objetivo
Implementar arquitetura de provedores de chamados plugáveis (**Ticketing Engine Adapter**) e interface web de parametrização dinâmica para suportar:
1. **Atlassian Jira** (Server / Data Center / Cloud)
2. **GLPI** (REST API v9 / v10)
3. **Movidesk** (Web API REST v1)

## ⚠️ Problema Atual
- Atualmente o Guardian possui acoplamento rígido com o Jira (via `internal/actions/jira.go` e `.env`).
- Não é possível utilizar GLPI ou Movidesk para abertura de incidentes.
- Credenciais e configurações são estáticas, exigindo edição de arquivo e reinício do binário.
- Não há roteamento por target (ex: target Homologação -> GLPI, target Produção -> Jira).

## 💡 Solução Proposta
- **Interface Go (`internal/actions/provider.go`)**: Definição da interface `TicketingProvider` com métodos `TestConnection` e `CreateIncidentTicket`.
- **Adaptadores**: Implementação de `JiraProvider`, `GLPIProvider` e `MovideskProvider`.
- **Frontend (Web UI)**: Nova view \"⚙️ Integrações ITSM\" no sidebar com abas dedicadas para cada provedor, campos com ofuscação de segredos e botão \"Testar Conexão\" em tempo real.
- **Roteamento por Target**: Configuração no modal de criação/edição de targets para escolher provedores ativos individualmente.
- **API REST**: Endpoints `/api/integrations/helpdesk` e `/api/integrations/helpdesk/test`.

## 📋 Critérios de Aceite
- [ ] Conexão com GLPI testável via `/initSession` pela interface web.
- [ ] Conexão com Movidesk testável via token pela interface web.
- [ ] Falha detectada em target abre chamado no(s) provedor(es) configurado(s).
- [ ] Credenciais e URLs persistidas com segurança sem expor segredos em logs.

## 🔗 Referência Técnica
Especificação detalhada disponível no repositório em:
`Documentações/issues/ISSUE-001-parametrizacao-multiprovedor-chamados-jira-glpi-movidesk.md`
"""

def create_issue(repo, token, title, body, labels=None):
    url = f"https://api.github.com/repos/{repo}/issues"
    headers = {
        "Authorization": f"Bearer {token}",
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
        "User-Agent": "Guardian-Issue-Creator"
    }
    payload = {
        "title": title,
        "body": body,
        "labels": labels or ["enhancement"]
    }
    
    resp = requests.post(url, json=payload, headers=headers, timeout=20)
    if resp.status_code == 201:
        data = resp.json()
        print("\n" + "=" * 60)
        print(">>> ISSUE CRIADA COM SUCESSO NO GITHUB! <<<")
        print("=" * 60)
        print(f"Número : #{data.get('number')}")
        print(f"Título : {data.get('title')}")
        print(f"URL    : {data.get('html_url')}")
        print("=" * 60)
        return data
    else:
        print(f"\n[!] Falha ao criar issue (HTTP {resp.status_code}):")
        try:
            err = resp.json()
            print(f"Mensagem: {err.get('message')}")
            if "errors" in err:
                for e in err["errors"]:
                    print(f"  - {e}")
        except Exception:
            print(resp.text)
        sys.exit(1)

def main():
    parser = argparse.ArgumentParser(description="Criar issue no GitHub via API")
    parser.add_argument("-t", "--token", default=os.getenv("GITHUB_TOKEN"), help="GitHub Personal Access Token")
    parser.add_argument("-r", "--repo", default=DEFAULT_REPO, help="Repositório (owner/repo)")
    parser.add_argument("-s", "--summary", default=DEFAULT_TITLE, help="Título da Issue")
    parser.add_argument("-d", "--description", default=DEFAULT_BODY, help="Corpo da Issue")
    args = parser.parse_args()

    token = args.token
    if not token:
        token = input("Digite o seu GitHub Token (Personal Access Token): ").strip()
        if not token:
            print("[!] GITHUB_TOKEN é obrigatório para autenticação na API do GitHub.")
            sys.exit(1)

    create_issue(args.repo, token, args.summary, args.description)

if __name__ == "__main__":
    main()
