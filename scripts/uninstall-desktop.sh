#!/usr/bin/env bash
# =============================================================================
# uninstall-desktop.sh — Desinstalador do Guardian SRE no Linux
# =============================================================================
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

REAL_HOME="${HOME}"
if [[ -n "${SNAP_REAL_HOME:-}" ]] && touch "${SNAP_REAL_HOME}/.snap_test_write" 2>/dev/null; then
    rm -f "${SNAP_REAL_HOME}/.snap_test_write"
    REAL_HOME="${SNAP_REAL_HOME}"
fi
PURGE=false

for arg in "$@"; do
    case "${arg}" in
        --purge|--clean|-p)
            PURGE=true
            ;;
    esac
done

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
sleep 0.5

# 2. Removendo executáveis
echo -e "${CYAN}[2/4] Removendo executáveis e backups...${RESET}"
rm -f "${REAL_HOME}/.local/bin/guardian" \
      "${REAL_HOME}/.local/bin/guardian.bak" \
      "${REAL_HOME}/.local/bin/guardian-app" \
      "${REAL_HOME}/.local/bin/guardian-ctl" \
      /usr/local/bin/guardian \
      /usr/local/bin/guardian.bak \
      /usr/local/bin/guardian-app \
      /usr/local/bin/guardian-ctl 2>/dev/null || true

# 3. Removendo atalhos do desktop e ícones
echo -e "${CYAN}[3/4] Removendo atalhos de desktop e ícones do sistema...${RESET}"
rm -f "${REAL_HOME}/.local/share/applications/guardian.desktop" \
      /usr/share/applications/guardian.desktop \
      "${REAL_HOME}/.local/share/icons/hicolor/scalable/apps/guardian.svg" \
      /usr/share/icons/hicolor/scalable/apps/guardian.svg 2>/dev/null || true

rm -f "${REAL_HOME}/.config/systemd/user/guardian.service" \
      /etc/systemd/system/guardian.service 2>/dev/null || true

# 4. Atualizando caches
echo -e "${CYAN}[4/4] Atualizando caches do sistema operacional...${RESET}"
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database "${REAL_HOME}/.local/share/applications" 2>/dev/null || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -f -t "${REAL_HOME}/.local/share/icons/hicolor" 2>/dev/null || true
fi
if command -v systemctl >/dev/null 2>&1; then
    systemctl --user daemon-reload 2>/dev/null || true
fi

# Tratamento de Dados e Configurações
if [[ "${PURGE}" == "true" ]]; then
    echo -e "${YELLOW}[PURGE] Removendo configurações e bancos de dados locais...${RESET}"
    rm -rf "${REAL_HOME}/.config/guardian" \
           "${REAL_HOME}/.local/share/guardian" \
           "${REAL_HOME}/.local/state/guardian" \
           /etc/guardian /var/lib/guardian /var/log/guardian 2>/dev/null || true
    echo -e "${GREEN}[OK]${RESET} Todos os dados e configurações foram eliminados permanentemente."
else
    echo ""
    echo -e "${YELLOW}Nota:${RESET} Seus dados e configurações em ${BOLD}~/.config/guardian${RESET} e ${BOLD}~/.local/share/guardian${RESET} foram preservados."
    echo -e "Para remover completamente seus dados no futuro, use:"
    echo -e "  ${CYAN}./install.sh --purge${RESET}  ou  ${CYAN}rm -rf ~/.config/guardian ~/.local/share/guardian ~/.local/state/guardian${RESET}"
fi

echo ""
echo -e "${BOLD}${GREEN}====================================================================${RESET}"
echo -e "${BOLD}${GREEN}  ✅ DESINSTALAÇÃO CONCLUÍDA COM SUCESSO!                           ${RESET}"
echo -e "${BOLD}${GREEN}====================================================================${RESET}"
echo ""
