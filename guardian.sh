#!/usr/bin/env bash
# =============================================================================
# guardian.sh — Script de atalho para o GuardianOps v2.0 Multi-Agente
# =============================================================================
# Uso rápido:
#   ./guardian.sh                        → Menu interativo
#   ./guardian.sh agents                 → Listar agentes
#   ./guardian.sh k8s <pod> <namespace>  → Analisar pod K8s
#   ./guardian.sh docker <container>     → Analisar container Docker
#   ./guardian.sh db <componente> <tipo> → Analisar falha de banco
#   ./guardian.sh qa <componente> <tipo> → Analisar falha de QA
#   ./guardian.sh dev <componente> <tipo>→ Analisar falha de código
#   ./guardian.sh scan                   → Varredura em todos os namespaces
#   ./guardian.sh jira                   → Testar conexão com Jira
# =============================================================================

set -euo pipefail

# ── Cores ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

# ── Configuração ─────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_PATH="${SCRIPT_DIR}/../.venv"
PYTHON="${VENV_PATH}/bin/python3"

# Fallback para python do sistema se o venv não existir
if [[ ! -f "${PYTHON}" ]]; then
    PYTHON="python3"
fi

GUARDIAN="cd ${SCRIPT_DIR} && ${PYTHON} main.py"

# ── Banner ────────────────────────────────────────────────────────────────────
banner() {
    echo -e ""
    echo -e "${BOLD}${BLUE}╔══════════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${BOLD}${BLUE}║  🛡️  GuardianOps v2.0 — Sistema Multi-Agente Autônomo        ║${RESET}"
    echo -e "${BOLD}${BLUE}║     Modo: Determinístico | Read-Only | Jira Integration       ║${RESET}"
    echo -e "${BOLD}${BLUE}╚══════════════════════════════════════════════════════════════╝${RESET}"
    echo -e ""
}

# ── Helpers ───────────────────────────────────────────────────────────────────
info()    { echo -e "${CYAN}[INFO]${RESET} $*"; }
success() { echo -e "${GREEN}[OK]${RESET}   $*"; }
warn()    { echo -e "${YELLOW}[WARN]${RESET} $*"; }
error()   { echo -e "${RED}[ERRO]${RESET} $*"; exit 1; }

confirm() {
    echo -e ""
    read -r -p "$(echo -e "${YELLOW}Abrir chamado no Jira? [s/N]:${RESET} ")" resp
    [[ "${resp,,}" == "s" || "${resp,,}" == "sim" || "${resp,,}" == "y" ]]
}

run_agent() {
    local cmd="$1"
    shift
    eval "cd ${SCRIPT_DIR} && ${PYTHON} main.py ${cmd} $*"
}

# ── Subcomandos ───────────────────────────────────────────────────────────────

cmd_agents() {
    banner
    run_agent "list-agents"
}

cmd_jira() {
    banner
    info "Testando conectividade com o Jira..."
    run_agent "test-jira"
}

cmd_k8s() {
    local pod="${1:-target-app}"
    local namespace="${2:-default}"
    local failure="${3:-CrashLoopBackOff}"
    local container="${4:-app}"
    local environment="${5:-Produção}"
    local logs=""

    banner
    info "Analisando pod K8s: ${BOLD}${pod}${RESET} (namespace: ${namespace})"

    # Capturar logs reais se kubectl disponível
    if command -v kubectl &>/dev/null; then
        info "kubectl encontrado — coletando logs do pod..."
        logs=$(kubectl logs "${pod}" -n "${namespace}" --previous --tail=80 2>/dev/null \
               || kubectl logs "${pod}" -n "${namespace}" --tail=80 2>/dev/null \
               || echo "Logs não disponíveis via kubectl")
        success "Logs coletados (${#logs} bytes)"
    else
        warn "kubectl não encontrado — análise sem logs ao vivo"
    fi

    info "Executando orquestrador (dry-run para revisão)..."
    echo ""

    run_agent "orchestrate" \
        --source kubernetes \
        --failure-type "\"${failure}\"" \
        --component "\"${pod}\"" \
        --namespace "\"${namespace}\"" \
        --container "\"${container}\"" \
        --environment "\"${environment}\"" \
        ${logs:+--logs "\"${logs:0:1000}\""} \
        --dry-run

    if confirm; then
        info "Abrindo chamado no Jira..."
        run_agent "orchestrate" \
            --source kubernetes \
            --failure-type "\"${failure}\"" \
            --component "\"${pod}\"" \
            --namespace "\"${namespace}\"" \
            --container "\"${container}\"" \
            --environment "\"${environment}\"" \
            ${logs:+--logs "\"${logs:0:1000}}\""}
        success "Chamado aberto no Jira!"
    else
        warn "Chamado NÃO enviado (operação cancelada pelo usuário)."
    fi
}

cmd_docker() {
    local container="${1:-web-app}"
    local failure="${2:-CrashLoopBackOff}"
    local logs=""

    banner
    info "Analisando container Docker: ${BOLD}${container}${RESET}"

    # Capturar logs reais se Docker disponível
    if command -v docker &>/dev/null && docker ps -q --filter "name=${container}" 2>/dev/null | grep -q .; then
        info "Docker encontrado — coletando logs do container..."
        logs=$(docker logs "${container}" --tail=80 2>&1 || echo "Logs não disponíveis")
        success "Logs coletados (${#logs} bytes)"
    else
        warn "Container '${container}' não encontrado localmente — análise sem logs ao vivo"
    fi

    info "Executando orquestrador (dry-run para revisão)..."
    echo ""

    run_agent "orchestrate" \
        --source docker \
        --failure-type "\"${failure}\"" \
        --component "\"${container}\"" \
        --namespace "local" \
        --container "\"${container}\"" \
        --environment "Local (Docker)" \
        ${logs:+--logs "\"${logs:0:1000}\""} \
        --dry-run

    if confirm; then
        run_agent "orchestrate" \
            --source docker \
            --failure-type "\"${failure}\"" \
            --component "\"${container}\"" \
            --namespace "local" \
            --container "\"${container}\"" \
            --environment "Local (Docker)" \
            ${logs:+--logs "\"${logs:0:1000}}\""}
        success "Chamado aberto no Jira!"
    else
        warn "Chamado NÃO enviado."
    fi
}

cmd_db() {
    local component="${1:-api-backend}"
    local failure="${2:-Deadlock}"
    local db_type="${3:-PostgreSQL}"

    banner
    info "Analisando falha de banco de dados: ${BOLD}${component}${RESET} (${db_type} / ${failure})"
    echo ""

    run_agent "orchestrate" \
        --source database \
        --failure-type "\"${failure} ${db_type}\"" \
        --component "\"${component}\"" \
        --namespace "production" \
        --environment "Produção" \
        --dry-run

    if confirm; then
        run_agent "orchestrate" \
            --source database \
            --failure-type "\"${failure} ${db_type}\"" \
            --component "\"${component}\"" \
            --namespace "production" \
            --environment "Produção"
        success "Chamado aberto no Jira!"
    else
        warn "Chamado NÃO enviado."
    fi
}

cmd_qa() {
    local component="${1:-portal-paciente}"
    local failure="${2:-TestFailed Regression}"

    banner
    info "Analisando falha de QA: ${BOLD}${component}${RESET} (${failure})"
    echo ""

    run_agent "orchestrate" \
        --source application \
        --failure-type "\"${failure}\"" \
        --component "\"${component}\"" \
        --namespace "ci-cd" \
        --environment "CI/CD Pipeline" \
        --dry-run

    if confirm; then
        run_agent "orchestrate" \
            --source application \
            --failure-type "\"${failure}\"" \
            --component "\"${component}\"" \
            --namespace "ci-cd" \
            --environment "CI/CD Pipeline"
        success "Chamado aberto no Jira!"
    else
        warn "Chamado NÃO enviado."
    fi
}

cmd_dev() {
    local component="${1:-backend-api}"
    local failure="${2:-NullPointerException}"
    local error_msg="${3:-}"

    banner
    info "Analisando falha de código: ${BOLD}${component}${RESET} (${failure})"
    echo ""

    run_agent "orchestrate" \
        --source application \
        --failure-type "\"${failure}\"" \
        --component "\"${component}\"" \
        --namespace "production" \
        --environment "Produção" \
        ${error_msg:+--error-message "\"${error_msg}\""} \
        --dry-run

    if confirm; then
        run_agent "orchestrate" \
            --source application \
            --failure-type "\"${failure}\"" \
            --component "\"${component}\"" \
            --namespace "production" \
            --environment "Produção" \
            ${error_msg:+--error-message "\"${error_msg}}\""}
        success "Chamado aberto no Jira!"
    else
        warn "Chamado NÃO enviado."
    fi
}

cmd_scan() {
    local namespaces="${1:-default producao monitoring}"

    banner
    info "Iniciando varredura K8s nos namespaces: ${namespaces}"
    warn "Modo: SOMENTE-LEITURA (Read-Only) — nenhuma alteração será feita"
    echo ""

    run_agent "scan-k8s" \
        --namespaces ${namespaces} \
        --dry-run
}

# ── Menu interativo ───────────────────────────────────────────────────────────
cmd_menu() {
    banner
    echo -e "${BOLD}Selecione o tipo de análise:${RESET}"
    echo ""
    echo -e "  ${CYAN}1)${RESET} Analisar falha de Pod (Kubernetes)"
    echo -e "  ${CYAN}2)${RESET} Analisar falha de Container (Docker)"
    echo -e "  ${CYAN}3)${RESET} Analisar falha de Banco de Dados"
    echo -e "  ${CYAN}4)${RESET} Analisar falha de QA / Testes"
    echo -e "  ${CYAN}5)${RESET} Analisar falha de Código (FullStack)"
    echo -e "  ${CYAN}6)${RESET} Varredura completa K8s (namespaces)"
    echo -e "  ${CYAN}7)${RESET} Listar agentes registrados"
    echo -e "  ${CYAN}8)${RESET} Testar conexão Jira"
    echo -e "  ${CYAN}q)${RESET} Sair"
    echo ""
    read -r -p "$(echo -e "${BOLD}Opção:${RESET} ")" choice

    case "${choice}" in
        1)
            read -r -p "Nome do pod: " pod
            read -r -p "Namespace [default]: " ns; ns="${ns:-default}"
            read -r -p "Tipo de falha [CrashLoopBackOff]: " ft; ft="${ft:-CrashLoopBackOff}"
            cmd_k8s "${pod}" "${ns}" "${ft}"
            ;;
        2)
            read -r -p "Nome do container: " cont
            read -r -p "Tipo de falha [CrashLoopBackOff]: " ft; ft="${ft:-CrashLoopBackOff}"
            cmd_docker "${cont}" "${ft}"
            ;;
        3)
            read -r -p "Componente/serviço afetado: " comp
            echo -e "Tipo de falha: ${CYAN}Deadlock${RESET} | ${CYAN}ConnectionPool${RESET} | ${CYAN}SlowQuery${RESET} | ${CYAN}Migration${RESET} | ${CYAN}Replication${RESET}"
            read -r -p "Tipo [Deadlock]: " ft; ft="${ft:-Deadlock}"
            echo -e "Banco: ${CYAN}PostgreSQL${RESET} | ${CYAN}MySQL${RESET} | ${CYAN}MongoDB${RESET} | ${CYAN}Redis${RESET} | ${CYAN}Elasticsearch${RESET}"
            read -r -p "Banco [PostgreSQL]: " db; db="${db:-PostgreSQL}"
            cmd_db "${comp}" "${ft}" "${db}"
            ;;
        4)
            read -r -p "Componente/projeto afetado: " comp
            echo -e "Tipo: ${CYAN}Regression${RESET} | ${CYAN}ApiContract${RESET} | ${CYAN}FlakyTest${RESET} | ${CYAN}Assertion${RESET} | ${CYAN}Timeout${RESET}"
            read -r -p "Tipo [Regression]: " ft; ft="${ft:-Regression}"
            cmd_qa "${comp}" "${ft}"
            ;;
        5)
            read -r -p "Componente/serviço afetado: " comp
            echo -e "Tipo: ${CYAN}NullPointerException${RESET} | ${CYAN}MemoryLeak${RESET} | ${CYAN}CORS${RESET} | ${CYAN}UnhandledPromise${RESET} | ${CYAN}500${RESET}"
            read -r -p "Tipo [NullPointerException]: " ft; ft="${ft:-NullPointerException}"
            read -r -p "Mensagem de erro (opcional): " errmsg
            cmd_dev "${comp}" "${ft}" "${errmsg}"
            ;;
        6)
            read -r -p "Namespaces separados por espaço [default producao]: " ns
            ns="${ns:-default producao}"
            cmd_scan "${ns}"
            ;;
        7) cmd_agents ;;
        8) cmd_jira ;;
        q|Q) echo -e "${GREEN}Saindo...${RESET}"; exit 0 ;;
        *) error "Opção inválida: ${choice}" ;;
    esac
}

# ── Entry point ───────────────────────────────────────────────────────────────
main() {
    local cmd="${1:-menu}"
    shift 2>/dev/null || true

    case "${cmd}" in
        install)        ./install.sh --install ;;
        update|upgrade) ./install.sh --update ;;
        rollback)       ./install.sh --rollback ;;
        uninstall)      ./install.sh --uninstall ;;
        status)         ./install.sh --status ;;
        app|gui)        ./bin/guardian -gui ;;
        menu|m)         cmd_menu ;;
        agents|a)       cmd_agents ;;
        jira|j)         cmd_jira ;;
        k8s|kubernetes) cmd_k8s "$@" ;;
        docker|d)       cmd_docker "$@" ;;
        db|database)    cmd_db "$@" ;;
        qa|test)        cmd_qa "$@" ;;
        dev|fullstack)  cmd_dev "$@" ;;
        scan|s)         cmd_scan "$@" ;;
        help|h|--help)
            banner
            echo -e "${BOLD}Uso:${RESET}"
            echo -e "  ./guardian.sh                              → Menu interativo"
            echo -e "  ./guardian.sh agents                       → Listar agentes"
            echo -e "  ./guardian.sh jira                         → Testar conexão Jira"
            echo -e "  ./guardian.sh k8s <pod> <ns> <falha>       → Analisar pod K8s"
            echo -e "  ./guardian.sh docker <container> <falha>   → Analisar container"
            echo -e "  ./guardian.sh db <componente> <falha> <db> → Analisar banco de dados"
            echo -e "  ./guardian.sh qa <componente> <falha>      → Analisar falha de QA"
            echo -e "  ./guardian.sh dev <componente> <falha>     → Analisar falha de código"
            echo -e "  ./guardian.sh scan [namespaces...]         → Varredura K8s"
            echo ""
            echo -e "${BOLD}Exemplos:${RESET}"
            echo -e "  ./guardian.sh k8s target-app default CrashLoopBackOff"
            echo -e "  ./guardian.sh docker mysql-db OOMKilled"
            echo -e "  ./guardian.sh db api-backend Deadlock PostgreSQL"
            echo -e "  ./guardian.sh qa web-portal Regression"
            echo -e "  ./guardian.sh dev backend-api NullPointerException"
            echo ""
            ;;
        *)
            error "Comando desconhecido: '${cmd}'. Use './guardian.sh help' para ver os comandos disponíveis."
            ;;
    esac
}

main "$@"
