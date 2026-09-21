#!/usr/bin/env bash
# =============================================================================
# install-desktop.sh — Instalador Nativo do Guardian SRE no Linux
# =============================================================================
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${SCRIPT_DIR}"

MODE="user"
if [[ "${1:-}" == "--system" || "${EUID}" -eq 0 ]]; then
    MODE="system"
fi

banner() {
    echo -e "${BOLD}${BLUE}╔══════════════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${BOLD}${BLUE}║       GUARDIAN SRE — INSTALADOR NATIVO PARA LINUX (DESKTOP)      ║${RESET}"
    echo -e "${BOLD}${BLUE}║       Enterprise SRE, K8s/Docker Supervisor & Jira Platform      ║${RESET}"
    echo -e "${BOLD}${BLUE}╚══════════════════════════════════════════════════════════════════╝${RESET}"
    echo -e ""
}

banner

# 1. Compilação do Binário Go
echo -e "${CYAN}[1/6] Compilando binário nativo do Guardian em Go...${RESET}"
mkdir -p bin
go build -ldflags="-s -w" -o bin/guardian ./cmd/guardian
chmod +x bin/guardian
echo -e "${GREEN}[OK]${RESET} Binário compilado com sucesso (bin/guardian)."

# 2. Definição de Caminhos
REAL_HOME="${SNAP_REAL_HOME:-$HOME}"

if [[ "${MODE}" == "user" ]]; then
    echo -e "${CYAN}[2/6] Configurando diretórios do usuário (~/.local)...${RESET}"
    BIN_DIR="${REAL_HOME}/.local/bin"
    APPS_DIR="${REAL_HOME}/.local/share/applications"
    ICONS_DIR="${REAL_HOME}/.local/share/icons/hicolor/scalable/apps"
    SYSTEMD_DIR="${REAL_HOME}/.config/systemd/user"
    CONFIG_DIR="${REAL_HOME}/.config/guardian"
    DATA_DIR="${REAL_HOME}/.local/share/guardian"
    STATE_DIR="${REAL_HOME}/.local/state/guardian"
else
    echo -e "${CYAN}[2/6] Configurando diretórios globais do sistema (/usr/local)...${RESET}"
    BIN_DIR="/usr/local/bin"
    APPS_DIR="/usr/share/applications"
    ICONS_DIR="/usr/share/icons/hicolor/scalable/apps"
    SYSTEMD_DIR="/etc/systemd/system"
    CONFIG_DIR="/etc/guardian"
    DATA_DIR="/var/lib/guardian"
    STATE_DIR="/var/log/guardian"
fi

mkdir -p "${BIN_DIR}" "${APPS_DIR}" "${ICONS_DIR}" "${SYSTEMD_DIR}" "${CONFIG_DIR}" "${DATA_DIR}" "${STATE_DIR}"

# 3. Cópia de Binários e Utilitários
echo -e "${CYAN}[3/6] Instalando executáveis e atalhos...${RESET}"
cp bin/guardian "${BIN_DIR}/guardian"
cp packaging/bin/guardian-app "${BIN_DIR}/guardian-app"
cp packaging/bin/guardian-ctl "${BIN_DIR}/guardian-ctl"
chmod +x "${BIN_DIR}/guardian" "${BIN_DIR}/guardian-app" "${BIN_DIR}/guardian-ctl"

# 4. Instalação de Ícones e Atalho de Desktop
echo -e "${CYAN}[4/6] Registrando aplicativo no menu do sistema (Zorin/GNOME)...${RESET}"
cp packaging/desktop/guardian.svg "${ICONS_DIR}/guardian.svg"
cp packaging/desktop/guardian.desktop "${APPS_DIR}/guardian.desktop"
chmod +x "${APPS_DIR}/guardian.desktop"

# 5. Instalação do Serviço systemd
echo -e "${CYAN}[5/6] Instalando serviço de segundo plano (systemd)...${RESET}"
cp packaging/systemd/guardian.service "${SYSTEMD_DIR}/guardian.service"

# Configuração e Persistência Inicial
if [[ ! -f "${CONFIG_DIR}/guardian.env" ]]; then
    if [[ -f "${SCRIPT_DIR}/.env" ]]; then
        cp "${SCRIPT_DIR}/.env" "${CONFIG_DIR}/guardian.env"
        echo -e "${GREEN}[OK]${RESET} Configurações existentes migradas para ${CONFIG_DIR}/guardian.env"
    elif [[ -f "${SCRIPT_DIR}/.env.example" ]]; then
        cp "${SCRIPT_DIR}/.env.example" "${CONFIG_DIR}/guardian.env"
        echo -e "${GREEN}[OK]${RESET} Modelo .env.example copiado para ${CONFIG_DIR}/guardian.env"
    fi
fi

if [[ ! -f "${DATA_DIR}/targets.json" && -f "${SCRIPT_DIR}/targets.json" ]]; then
    cp "${SCRIPT_DIR}/targets.json" "${DATA_DIR}/targets.json"
    echo -e "${GREEN}[OK]${RESET} Targets existentes copiados para ${DATA_DIR}/targets.json"
fi

# 6. Atualização de Caches do Sistema
echo -e "${CYAN}[6/6] Atualizando caches do desktop e ativando serviços...${RESET}"

if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database "${APPS_DIR}" 2>/dev/null || true
fi

if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -f -t "${REAL_HOME}/.local/share/icons/hicolor" 2>/dev/null || true
fi

if [[ "${MODE}" == "user" ]]; then
    if command -v systemctl >/dev/null 2>&1; then
        systemctl --user daemon-reload 2>/dev/null || true
        systemctl --user enable guardian 2>/dev/null || true
        echo -e "${GREEN}[OK]${RESET} Serviço de segundo plano habilitado no systemd do usuário."
    fi
else
    if command -v systemctl >/dev/null 2>&1; then
        systemctl daemon-reload 2>/dev/null || true
        systemctl enable guardian 2>/dev/null || true
    fi
fi

echo ""
echo -e "${BOLD}${GREEN}====================================================================${RESET}"
echo -e "${BOLD}${GREEN}  🎉 INSTALAÇÃO CONCLUÍDA COM SUCESSO!                              ${RESET}"
echo -e "${BOLD}${GREEN}====================================================================${RESET}"
echo ""
echo -e "O ${BOLD}Guardian SRE${RESET} agora é um aplicativo nativo do seu Linux."
echo ""
echo -e "📌 ${BOLD}Formas de Uso:${RESET}"
echo -e "  1. ${CYAN}Menu de Aplicativos / Dash:${RESET} Procure por ${BOLD}Guardian SRE${RESET} e clique no ícone."
echo -e "  2. ${CYAN}Linha de Comando (CLI):${RESET}"
echo -e "     • Abrir janela desktop:   ${BOLD}guardian-app${RESET} ou ${BOLD}guardian-ctl open${RESET}"
echo -e "     • Status do serviço:      ${BOLD}guardian-ctl status${RESET}"
echo -e "     • Iniciar daemon:         ${BOLD}guardian-ctl start${RESET}"
echo -e "     • Parar daemon:           ${BOLD}guardian-ctl stop${RESET}"
echo -e "     • Ver logs:               ${BOLD}guardian-ctl logs${RESET}"
echo -e "  3. ${CYAN}Serviço systemd:${RESET}"
echo -e "     • Iniciar:                ${BOLD}systemctl --user start guardian${RESET}"
echo -e "     • Status:                 ${BOLD}systemctl --user status guardian${RESET}"
echo ""
echo -e "📁 ${BOLD}Arquivos Principais:${RESET}"
echo -e "  • Binário:         ${CYAN}${BIN_DIR}/guardian${RESET}"
echo -e "  • Launcher GUI:    ${CYAN}${BIN_DIR}/guardian-app${RESET}"
echo -e "  • CLI Control:     ${CYAN}${BIN_DIR}/guardian-ctl${RESET}"
echo -e "  • Configuração:    ${CYAN}${CONFIG_DIR}/guardian.env${RESET}"
echo -e "  • Dados (targets): ${CYAN}${DATA_DIR}/targets.json${RESET}"
echo -e "  • Atalho Desktop:  ${CYAN}${APPS_DIR}/guardian.desktop${RESET}"
echo ""
if [[ ":$PATH:" != *":${BIN_DIR}:"* ]]; then
    echo -e "${YELLOW}⚠️  Aviso: O diretório ${BIN_DIR} não está no seu PATH.${RESET}"
    echo -e "Adicione a seguinte linha ao seu ${BOLD}~/.bashrc${RESET} ou ${BOLD}~/.zshrc${RESET}:"
    echo -e "  ${CYAN}export PATH=\"\$HOME/.local/bin:\$PATH\"${RESET}"
    echo ""
fi
