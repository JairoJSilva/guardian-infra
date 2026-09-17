def build_jira_description(
    reference: str,
    date_str: str,
    environment: str,
    title: str,
    issue_type: str,
    priority: str,
    category: str,
    root_cause_explanation: str,
    evidence_snippet: str,
    impacted_assets_table: str,
    remediation_procedure: str,
    acceptance_criteria: str,
    rollback_plan: str
) -> str:
    """
    Constrói a descrição do chamado no formato padrão de Wiki Markup do Jira,
    seguindo o template institucional estabelecido e garantindo compatibilidade total
    com o parser de formatação do Jira Server/Data Center.
    """
    evidence_block = ""
    if evidence_snippet and evidence_snippet.strip():
        # Trunca evidências longas para não estourar o limite de campos do Jira
        snippet = evidence_snippet.strip()
        if len(snippet) > 2500:
            snippet = snippet[:2500] + "\n... [TRECHO TRUNCADO PELO GUARDIANOPS PARA OTIMIZAÇÃO] ..."
        evidence_block = f"""
*Evidências Coletadas:*
{{code:text}}
{snippet}
{{code}}
"""

    description = f"""*Referência:* {reference}
*Data:* {date_str}
*Ambiente:* {environment}
*Origem:* Gerado por I.A. 🤖 (GuardianOps 🛡️)
*Tag de Rastreabilidade:* {{{{gerado-por-ia}}}}, {{{{guardian-ops}}}}

----

h2. 📌 {title}

* *Tipo:* {issue_type}
* *Prioridade:* {priority}
* *Categoria:* {category}
* *Tag / Rótulo:* {{{{gerado-por-ia}}}}, {{{{guardian-ops}}}}

h3. 1. Diagnóstico & Análise de Causa Raiz (RCA)
{root_cause_explanation.strip()}
{evidence_block}
h3. 2. Ativos Impactados
{impacted_assets_table.strip()}

h3. 3. Procedimento de Correção Sugerido (Step-by-Step)
{remediation_procedure.strip()}

h3. 4. Critérios de Aceite
{acceptance_criteria.strip()}

h3. 5. Plano de Rollback
{rollback_plan.strip()}
"""
    return description.strip()
