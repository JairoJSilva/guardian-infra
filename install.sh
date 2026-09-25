#!/usr/bin/env bash
# =============================================================================
# install.sh — Gerenciador Central de Ciclo de Vida do Guardian SRE Platform
# Instalação, Atualização, Rollback, Diagnóstico e Remoção no Linux
# =============================================================================
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; MAGENTA='\033[0;35m'
BOLD='\033[1m'; RESET='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

REAL_HOME="${HOME}"
if [[ -n "${SNAP_REAL_HOME:-}" ]] && touch "${SNAP_REAL_HOME}/.snap_test_write" 2>/dev/null; then
    rm -f "${SNAP_REAL_HOME}/.snap_test_write"
    REAL_HOME="${SNAP_REAL_HOME}"
fi
BIN_PATH="${REAL_HOME}/.local/bin/guardian"
CONFIG_FILE="${REAL_HOME}/.config/guardian/guardian.env"

get_repo_version() {
    cat "${SCRIPT_DIR}/VERSION" 2>/dev/null || echo "3.1.0"
}

get_installed_version() {
    local bin=""
    for candidate in "${REAL_HOME}/.local/bin/guardian" "/usr/local/bin/guardian" "$(command -v guardian 2>/dev/null || true)"; do
        if [[ -n "${candidate}" && -x "${candidate}" ]]; then
            bin="${candidate}"
            break
        fi
    done
    if [[ -n "${bin}" ]]; then
        local ver_output
        ver_output=$("${bin}" -version 2>/dev/null || true)
        local ver_matched
        ver_matched=$(echo "${ver_output}" | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' || true)
        if [[ -n "${ver_matched}" ]]; then
            echo "${ver_matched}"
        else
            echo "instalado"
        fi
    else
        echo "não instalado"
    fi
}

is_active() {
    if command -v curl >/dev/null 2>&1; then
        curl -s --connect-timeout 1 "http://localhost:8092/api/health" >/dev/null 2>&1
    else
        pgrep -f "guardian" >/dev/null 2>&1
    fi
}

show_header() {
    clear 2>/dev/null || true
    local repo_ver
    local inst_ver
    local status_str

    repo_ver=$(get_repo_version)
    inst_ver=$(get_installed_version)

    if is_active; then
        status_str="${GREEN}● EM EXECUÇÃO (porta 8092)${RESET}"
    elif [[ "${inst_ver}" != "não instalado" ]]; then
        status_str="${YELLOW}○ PARADO${RESET}"
    else
        status_str="${RED}○ NÃO INSTALADO${RESET}"
    fi

    echo -e "${BOLD}${BLUE}╔══════════════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${BOLD}${BLUE}║        🛡️  GUARDIAN SRE — GERENCIADOR CENTRAL NATIVO LINUX        ║${RESET}"
    echo -e "${BOLD}${BLUE}║       Enterprise SRE, K8s/Docker Supervisor & Jira Platform      ║${RESET}"
    echo -e "${BOLD}${BLUE}╚══════════════════════════════════════════════════════════════════╝${RESET}"
    echo ""
    echo -e " 🏷️  ${BOLD}Versão no Repositório:${RESET}  ${CYAN}v${repo_ver}${RESET}"
    echo -e " 💻 ${BOLD}Versão no seu Sistema:${RESET}  ${MAGENTA}${inst_ver}${RESET}"
    echo -e " ⚡ ${BOLD}Estado do Serviço:${RESET}      ${status_str}"
    echo -e " 📂 ${BOLD}Diretório Raiz:${RESET}         ${SCRIPT_DIR}"
    echo -e "────────────────────────────────────────────────────────────────────"
}

cmd_status() {
    show_header
    echo ""
    echo -e "${BOLD}=== Diagnóstico Detalhado do Guardian SRE ===${RESET}"
    if is_active; then
        echo -e "Status:     ${GREEN}${BOLD}ATIVO E OPERACIONAL${RESET}"
        echo -e "URL Local:  ${CYAN}http://localhost:8092${RESET}"
        if command -v curl >/dev/null 2>&1; then
            echo -e "Telemetria: $(curl -s "http://localhost:8092/api/health" 2>/dev/null || echo '{}')"
        fi
    else
        echo -e "Status:     ${RED}INATIVO${RESET}"
    fi

    if [[ -x "${REAL_HOME}/.local/bin/guardian-ctl" ]]; then
        echo ""
        "${REAL_HOME}/.local/bin/guardian-ctl" status || true
    fi
}

cmd_install() {
    chmod +x scripts/install-desktop.sh packaging/bin/*
    ./scripts/install-desktop.sh
}

cmd_update() {
    chmod +x scripts/install-desktop.sh packaging/bin/*
    ./scripts/install-desktop.sh --update
}

cmd_rollback() {
    chmod +x scripts/install-desktop.sh
    ./scripts/install-desktop.sh --rollback
}

cmd_system_install() {
    chmod +x scripts/install-desktop.sh packaging/bin/*
    sudo ./scripts/install-desktop.sh --system
}

cmd_uninstall() {
    chmod +x scripts/uninstall-desktop.sh
    ./scripts/uninstall-desktop.sh
}

cmd_purge() {
    chmod +x scripts/uninstall-desktop.sh
    ./scripts/uninstall-desktop.sh --purge
}

menu() {
    while true; do
        show_header
        echo ""
        echo -e " ${BOLD}Escolha uma operação:${RESET}"
        echo ""
        echo -e "  ${CYAN}[1]${RESET} 🚀 ${BOLD}Instalar Guardian SRE${RESET} (~/.local - Usuário Recomendado)"
        echo -e "  ${CYAN}[2]${RESET} 🔄 ${BOLD}Atualizar para Nova Versão${RESET} (Hot-Upgrade com Backup e Auto-Restart)"
        echo -e "  ${CYAN}[3]${RESET} 📊 ${BOLD}Verificar Status & Saúde${RESET} (Telemetria do Supervisor)"
        echo -e "  ${CYAN}[4]${RESET} ⏪ ${BOLD}Reverter Versão (Rollback)${RESET} (Voltar ao binário anterior)"
        echo -e "  ${CYAN}[5]${RESET} 🌐 ${BOLD}Instalação Global${RESET} (/usr/local - Requer sudo)"
        echo -e "  ${CYAN}[6]${RESET} 🗑️  ${BOLD}Desinstalar Guardian SRE${RESET} (Preservar dados e configurações)"
        echo -e "  ${CYAN}[7]${RESET} ⚠️  ${BOLD}Desinstalar e Limpar Tudo (Purge)${RESET} (Remover dados/configs)"
        echo -e "  ${CYAN}[0]${RESET} 🚪 Sair"
        echo ""
        read -r -p " Digite a opção [0-7]: " opt
        case "${opt}" in
            1) cmd_install; break ;;
            2) cmd_update; break ;;
            3) cmd_status; echo ""; read -r -p "Pressione Enter para continuar...";;
            4) cmd_rollback; break ;;
            5) cmd_system_install; break ;;
            6) cmd_uninstall; break ;;
            7) 
                read -r -p "Tem certeza que deseja apagar todos os dados e configurações? [s/N]: " confirm
                if [[ "${confirm,,}" == "s" || "${confirm,,}" == "sim" || "${confirm,,}" == "y" ]]; then
                    cmd_purge
                fi
                break
                ;;
            0|q|exit) echo -e "\nAté mais!"; exit 0 ;;
            *) echo -e "${RED}Opção inválida!${RESET}"; sleep 1 ;;
        esac
    done
}

# Tratamento de flags CLI não-interativas
case "${1:-}" in
    --install|-i)
        cmd_install
        ;;
    --update|-u|update)
        cmd_update
        ;;
    --rollback|-r|rollback)
        cmd_rollback
        ;;
    --status|-s|status)
        cmd_status
        ;;
    --system)
        cmd_system_install
        ;;
    --uninstall|-d|uninstall)
        cmd_uninstall
        ;;
    --purge|purge)
        cmd_purge
        ;;
    --help|-h|help)
        echo -e "${BOLD}${BLUE}Guardian SRE — Gerenciador Central${RESET}"
        echo "Uso: ./install.sh [opção]"
        echo ""
        echo "Opções:"
        echo "  (sem argumentos)   Abre o menu interativo com diagnóstico"
        echo "  --install, -i      Instala localmente (~/.local)"
        echo "  --update, -u       Compila e atualiza a máquina para a versão atual do repo"
        echo "  --rollback, -r     Restaura a versão anterior se houver falhas"
        echo "  --status, -s       Exibe status e telemetria da aplicação"
        echo "  --system           Instala globalmente no sistema (/usr/local)"
        echo "  --uninstall, -d    Remove os executáveis e atalhos (mantém configs)"
        echo "  --purge            Remove tudo, incluindo dados e configurações"
        echo "  --help, -h         Exibe esta mensagem de ajuda"
        ;;
    *)
        menu
        ;;
esac
