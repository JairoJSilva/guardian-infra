#!/usr/bin/env python3
import os
import sys
from pathlib import Path
from urllib.parse import urlparse
from dotenv import load_dotenv
import requests

env_path = Path(__file__).parent / ".env"
load_dotenv(dotenv_path=env_path)

raw_url = os.getenv("JIRA_BASE_URL", "https://jira.mv.com.br").strip()
parsed = urlparse(raw_url)
JIRA_BASE_URL = f"{parsed.scheme}://{parsed.netloc}" if parsed.netloc else raw_url.rstrip("/")
JIRA_USER = os.getenv("JIRA_USER", "").strip()
JIRA_PASSWORD = os.getenv("JIRA_PASSWORD", "").strip()

PARENT_ISSUE = "OPS-236"

issues_to_create = [
    {
        "num": 1,
        "summary": "Inclusão de Volumes de Dados de Produção no AWS Backup",
        "priority": "Média",
        "labels": ["gerado-por-ia"],
        "description": """*Referência:* OPS-236 — Governança e Rotina de Backups na AWS
*Data:* 16/09/2026
*Ambiente:* AWS Produção
*Origem:* Gerado por I.A. 🤖
*Tag de Rastreabilidade:* {{gerado-por-ia}}

----

h2. 📌 Chamado 1: Inclusão de Volumes de Dados de Produção no AWS Backup

* *Tipo:* Requisição de Serviço / Mudança Padrão (_Standard Change_)
* *Prioridade:* Média
* *Categoria:* Infraestrutura Cloud / AWS EC2 / Backup
* *Tag / Rótulo:* {{gerado-por-ia}}

h3. 1. Descrição & Justificativa
Aplicar a tag oficial de governança {{Backup = true}} nos volumes de dados adicionais (drive {{B:}}) dos ambientes de produção. Atualmente esses volumes estão sem a marcação requerida pelo plano de backup automatizado da organização, expondo os dados a riscos em caso de desastre.

h3. 2. Ativos Impactados
|| Host || Volume ID || Tamanho || Função ||
| {{EC2AMAZ-507JH62}} | {{vol-056633720c4c6a00f}} | 450 GB | Disco de dados (B:) |
| {{EC2AMAZ-SGV7DA5}} | {{vol-05c3d360e993f3191}} | 350 GB | Disco de dados (B:) |

h3. 3. Procedimento de Execução (Step-by-Step)
1. Conectar via AWS CLI com o profile autorizado.
2. Checar se os volumes existem e verificar tags atuais:
{code:bash}
aws ec2 describe-tags --filters "Name=resource-id,Values=vol-056633720c4c6a00f,vol-05c3d360e993f3191"
{code}
3. Aplicar a tag nos dois recursos:
{code:bash}
aws ec2 create-tags --resources vol-056633720c4c6a00f vol-05c3d360e993f3191 --tags Key=Backup,Value=true
{code}
4. Validar se a tag foi persistida:
{code:bash}
aws ec2 describe-tags --filters "Name=resource-id,Values=vol-056633720c4c6a00f,vol-05c3d360e993f3191" "Name=key,Values=Backup"
{code}

h3. 4. Critérios de Aceite
* Ambos os volumes apresentarem exatamente a tag {{Key: Backup}}, {{Value: true}}.

h3. 5. Plano de Rollback
Remover a tag caso haja necessidade de retirada da rotina:
{code:bash}
aws ec2 delete-tags --resources vol-056633720c4c6a00f vol-05c3d360e993f3191 --tags Key=Backup
{code}
"""
    },
    {
        "num": 2,
        "summary": "Saneamento e Correção de Tag de Backup (Governança)",
        "priority": "Baixa",
        "labels": ["gerado-por-ia"],
        "description": """*Referência:* OPS-236 — Governança e Rotina de Backups na AWS
*Data:* 16/09/2026
*Ambiente:* AWS Produção
*Origem:* Gerado por I.A. 🤖
*Tag de Rastreabilidade:* {{gerado-por-ia}}

----

h2. 📌 Chamado 2: Saneamento e Correção de Tag de Backup (Governança)

* *Tipo:* Incidente / Correção de Configuração
* *Prioridade:* Baixa / Média
* *Categoria:* Governança Cloud / Higienização de Metadados
* *Tag / Rótulo:* {{gerado-por-ia}}

h3. 1. Descrição & Justificativa
Identificado no Host que o volume possui uma chave de tag incorreta contendo espaço em branco ao final: {{"Backup "}}. A política de seleção do AWS Backup é estrita e sensível a caracteres (_case-sensitive_ e com espaços), de modo que a chave {{"Backup "}} não é contemplada na regra automatizada, deixando o volume desprotegido.

h3. 2. Ativos Impactados
|| Host || Volume ID || Problema Identificado || Correção Esperada ||
| Host 1 | {{vol-0832eddfaf41a1a62}} | Tag {{"Backup "}} (com espaço final) | Tag {{"Backup"}} (sem espaço) |

h3. 3. Procedimento de Execução (Step-by-Step)
1. Confirmar a existência da tag com espaço:
{code:bash}
aws ec2 describe-tags --filters "Name=resource-id,Values=vol-0832eddfaf41a1a62"
{code}
2. Deletar a chave incorreta {{"Backup "}}:
{code:bash}
aws ec2 delete-tags --resources vol-0832eddfaf41a1a62 --tags "Key=Backup "
{code}
3. Criar a chave correta {{"Backup"}} com valor {{true}}:
{code:bash}
aws ec2 create-tags --resources vol-0832eddfaf41a1a62 --tags Key=Backup,Value=true
{code}
4. Validar o saneamento:
{code:bash}
aws ec2 describe-tags --filters "Name=resource-id,Values=vol-0832eddfaf41a1a62" --query "Tags[?Key=='Backup']"
{code}

h3. 4. Critérios de Aceite
* Chave {{"Backup "}} inexistente.
* Chave {{"Backup"}} presente com valor {{true}}.

h3. 5. Plano de Rollback
Restaurar a tag original com espaço (caso algum script legado dependa dela):
{code:bash}
aws ec2 delete-tags --resources vol-0832eddfaf41a1a62 --tags Key=Backup
aws ec2 create-tags --resources vol-0832eddfaf41a1a62 --tags "Key=Backup ,Value=true"
{code}
"""
    },
    {
        "num": 3,
        "summary": "GMUD / RFC — Alteração da Janela de Início da Regra daily-ebs",
        "priority": "Média",
        "labels": ["gerado-por-ia"],
        "description": """*Referência:* OPS-236 — Governança e Rotina de Backups na AWS
*Data:* 16/09/2026
*Ambiente:* AWS Produção
*Origem:* Gerado por I.A. 🤖
*Tag de Rastreabilidade:* {{gerado-por-ia}}

----

h2. 📌 Chamado 3: GMUD / RFC — Alteração da Janela de Início da Regra daily-ebs

* *Tipo:* Gestão de Mudança (RFC / _Normal Change_)
* *Prioridade:* Média / Alta
* *Categoria:* AWS Backup / Janela de Manutenção
* *Necessita Aprovação:* Sim (Equipe de DBA / Infraestrutura)
* *Tag / Rótulo:* {{gerado-por-ia}}

h3. 1. Descrição & Justificativa
A regra de backup automatizado {{daily-ebs}} inicia atualmente às *03:00 UTC (00:00 Horário de Brasília)*. Neste mesmo horário, ocorrem as rotinas locais de backup nativo do SQL Server, gerando concorrência de I/O de disco e risco de snapshots inconsistentes ou incompletos.  
A ação consiste em mover a janela para *06:00 UTC (03:00 Horário de Brasília)*, garantindo que os backups de banco já estejam concluídos.

h3. 2. Ativos Impactados
|| Serviço || Recurso / Regra || Configuração Atual || Nova Configuração ||
| AWS Backup | Regra {{daily-ebs}} | {{03:00 UTC}} (00:00 BRT) | {{06:00 UTC}} (03:00 BRT) |
| Expressão Cron | Schedule Expression | {{cron(0 3 * * ? *)}} | {{cron(0 6 * * ? *)}} |

h3. 3. Procedimento de Execução (Step-by-Step)
1. Localizar o ID do plano de backup:
{code:bash}
aws backup list-backup-plans --query "BackupPlansList[?BackupPlanName=='daily-ebs'].BackupPlanId" --output text
{code}
2. Realizar backup da definição JSON atual:
{code:bash}
aws backup get-backup-plan --backup-plan-id "<PLAN_ID>" --query "BackupPlan" --output json > backup_plan_daily-ebs_bkp.json
{code}
3. Validar se a janela do SQL Server finaliza antes das 03:00 BRT.
4. Atualizar a regra no plano com o novo cron {{cron(0 6 * * ? *)}}.
5. Monitorar a execução no primeiro dia após a mudança no console do AWS Backup.

h3. 4. Critérios de Aceite
* Plano {{daily-ebs}} apresentando horário de início às 06:00 UTC.
* Nenhum job de backup disparado às 03:00 UTC no dia seguinte.
* Jobs disparados com sucesso às 06:00 UTC.

h3. 5. Plano de Rollback
Restaurar a regra para o horário de 03:00 UTC original:
{code:bash}
# Reverte a ScheduleExpression para cron(0 3 * * ? *)
aws backup update-backup-plan --backup-plan-id "<PLAN_ID>" --backup-plan file://backup_plan_restored.json
{code}
"""
    }
]

def link_issues(inward_key, outward_key, link_type="Relates"):
    url = f"{JIRA_BASE_URL}/rest/api/2/issueLink"
    payload = {
        "type": {"name": link_type},
        "inwardIssue": {"key": inward_key},
        "outwardIssue": {"key": outward_key},
        "comment": {
            "body": f"Vinculado automaticamente ao chamado de referência {outward_key}"
        }
    }
    res = requests.post(url, json=payload, auth=(JIRA_USER, JIRA_PASSWORD), timeout=15)
    if res.status_code in [200, 201, 204]:
        print(f"  [+] Vinculado com sucesso a {outward_key}!")
    else:
        print(f"  [!] Não foi possível vincular a {outward_key}: HTTP {res.status_code}")

def main():
    print(f"Iniciando criação dos 3 chamados detalhados no projeto OPS...")
    created_list = []

    for item in issues_to_create:
        print(f"\n---> Criando Chamado {item['num']}: {item['summary']}...")
        payload = {
            "fields": {
                "project": {"key": "OPS"},
                "summary": item["summary"],
                "description": item["description"],
                "issuetype": {"name": "Solicitação de serviço"},
                "priority": {"name": item["priority"]},
                "reporter": {"name": JIRA_USER},
                "labels": item["labels"],
                "customfield_30118": {"id": "22300", "value": "DENTALIS"}
            }
        }

        res = requests.post(
            f"{JIRA_BASE_URL}/rest/api/2/issue",
            json=payload,
            auth=(JIRA_USER, JIRA_PASSWORD),
            headers={"Content-Type": "application/json", "Accept": "application/json"},
            timeout=25
        )

        if res.status_code == 201:
            data = res.json()
            key = data.get("key")
            created_list.append((item["num"], key, item["summary"]))
            print(f"  [✓] Criado com sucesso: {key} ({JIRA_BASE_URL}/browse/{key})")
            
            # Vincular ao OPS-236
            link_issues(inward_key=key, outward_key=PARENT_ISSUE)
        else:
            print(f"  [!] Erro ao criar Chamado {item['num']}: HTTP {res.status_code}")
            print(res.text)

    print("\n" + "=" * 60)
    print("RESUMO DOS CHAMADOS CRIADOS:")
    print("=" * 60)
    for num, key, title in created_list:
        print(f"• Chamado {num}: {key} - {title}")
        print(f"  Link: {JIRA_BASE_URL}/browse/{key}")
    print("=" * 60)

if __name__ == "__main__":
    main()
