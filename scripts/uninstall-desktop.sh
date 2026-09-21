#!/usr/bin/env bash
# =============================================================================
# uninstall-desktop.sh — Desinstalador do Guardian SRE no Linux
# =============================================================================
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

REAL_HOME="${SNAP_REAL_HOME:-$HOME}"

echo -e "${BOLD}${RED}╔══════════════════════════════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}${RED}║                 DESINSTALAÇÃO DO GUARDIAN SRE                    ║${RESET}"
echo -e "${BOLD}${RED}╚══════════════════════════════════════════════════════════════════╝${RESET}"
echo ""

# 1. Parando serviços e processos
echo -e "${CYAN}[1/4] Encerrando serviços e processos em execução...${RESET}"
if command -v systemctl >/dev/null 2>&1; then
    systemctl --user stop guardian 2>/dev/null || true
    systemctl --user disable guardian 2>/dev/null || true
fi
pkill -f "guardian" 2>/dev/null || true

# 2. Removendo executáveis
echo -e "${CYAN}[2/4] Removendo executáveis...${RESET}"
rm -f "${REAL_HOME}/.local/bin/guardian" \
      "${REAL_HOME}/.local/bin/guardian-app" \
      "${REAL_HOME}/.local/bin/guardian-ctl" \
      /usr/local/bin/guardian \
      /usr/local/bin/guardian-app \
      /usr/local/bin/guardian-ctl 2>/dev/null || true

# 3. Removendo atalhos do desktop e ícones
echo -e "${CYAN}[3/4] Removendo atalho de desktop e ícones...${RESET}"
rm -f "${REAL_HOME}/.local/share/applications/guardian.desktop" \
      /usr/share/applications/guardian.desktop \
      "${REAL_HOME}/.local/share/icons/hicolor/scalable/apps/guardian.svg" \
      /usr/share/icons/hicolor/scalable/apps/guardian.svg 2>/dev/null || true

rm -f "${REAL_HOME}/.config/systemd/user/guardian.service" \
      /etc/systemd/system/guardian.service 2>/dev/null || true

# 4. Atualizando caches
echo -e "${CYAN}[4/4] Atualizando caches do sistema...${RESET}"
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database "${REAL_HOME}/.local/share/applications" 2>/dev/null || true
fi
if command -v systemctl >/dev/null 2>&1; then
    systemctl --user daemon-reload 2>/dev/null || true
fi

echo ""
echo -e "${GREEN}[OK]${RESET} Desinstalação concluída com sucesso."
echo -e "${YELLOW}Nota:${RESET} Seus dados e configurações em ${BOLD}~/.config/guardian${RESET} e ${BOLD}~/.local/share/guardian${RESET} foram preservados por segurança."
echo -e "Para remover completamente os dados, execute:"
echo -e "  rm -rf ~/.config/guardian ~/.local/share/guardian ~/.local/state/guardian"
echo ""
