#!/usr/bin/env bash
# =============================================================================
# install-desktop.sh — Instalador & Atualizador Nativo do Guardian SRE no Linux
# =============================================================================
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${SCRIPT_DIR}"

MODE="user"
ACTION="install" # install, update, rollback

for arg in "$@"; do
    case "${arg}" in
        --system) MODE="system" ;;
        --update|-u) ACTION="update" ;;
        --rollback|-r) ACTION="rollback" ;;
    esac
done

if [[ "${EUID}" -eq 0 ]]; then
    MODE="system"
fi

REAL_HOME="${HOME}"
if [[ -n "${SNAP_REAL_HOME:-}" ]] && touch "${SNAP_REAL_HOME}/.snap_test_write" 2>/dev/null; then
    rm -f "${SNAP_REAL_HOME}/.snap_test_write"
    REAL_HOME="${SNAP_REAL_HOME}"
fi

if [[ "${MODE}" == "user" ]]; then
    BIN_DIR="${REAL_HOME}/.local/bin"
    APPS_DIR="${REAL_HOME}/.local/share/applications"
    ICONS_DIR="${REAL_HOME}/.local/share/icons/hicolor/scalable/apps"
    SYSTEMD_DIR="${REAL_HOME}/.config/systemd/user"
    CONFIG_DIR="${REAL_HOME}/.config/guardian"
    DATA_DIR="${REAL_HOME}/.local/share/guardian"
    STATE_DIR="${REAL_HOME}/.local/state/guardian"
else
    BIN_DIR="/usr/local/bin"
    APPS_DIR="/usr/share/applications"
    ICONS_DIR="/usr/share/icons/hicolor/scalable/apps"
    SYSTEMD_DIR="/etc/systemd/system"
    CONFIG_DIR="/etc/guardian"
    DATA_DIR="/var/lib/guardian"
    STATE_DIR="/var/log/guardian"
fi

banner() {
    local title="INSTALADOR NATIVO PARA LINUX (DESKTOP)"
    if [[ "${ACTION}" == "update" ]]; then
        title="ATUALIZADOR DE VERSÃO AUTOMÁTICO (HOT-UPGRADE)"
    elif [[ "${ACTION}" == "rollback" ]]; then
        title="REVERSÃO DE VERSÃO (ROLLBACK)"
    fi
    echo -e "${BOLD}${BLUE}╔══════════════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${BOLD}${BLUE}║       GUARDIAN SRE — ${title}       ║${RESET}"
    echo -e "${BOLD}${BLUE}║       Enterprise SRE, K8s/Docker Supervisor & Jira Platform      ║${RESET}"
    echo -e "${BOLD}${BLUE}╚══════════════════════════════════════════════════════════════════╝${RESET}"
    echo ""
}

# --- Rollback ---
if [[ "${ACTION}" == "rollback" ]]; then
    banner
    echo -e "${CYAN}[1/3] Verificando backup da versão anterior...${RESET}"
    if [[ ! -f "${BIN_DIR}/guardian.bak" ]]; then
        echo -e "${RED}[ERRO] Nenhum backup encontrado em ${BIN_DIR}/guardian.bak${RESET}"
        exit 1
    fi

    echo -e "${CYAN}[2/3] Parando serviço e restaurando binário anterior...${RESET}"
    if command -v systemctl >/dev/null 2>&1; then
        systemctl --user stop guardian 2>/dev/null || true
    fi
    pkill -f "guardian" 2>/dev/null || true
    sleep 0.5

    mv "${BIN_DIR}/guardian.bak" "${BIN_DIR}/guardian"
    chmod +x "${BIN_DIR}/guardian"

    echo -e "${CYAN}[3/3] Reiniciando serviço...${RESET}"
    if command -v systemctl >/dev/null 2>&1; then
        systemctl --user daemon-reload 2>/dev/null || true
        systemctl --user start guardian 2>/dev/null || true
    fi

    echo -e "${BOLD}${GREEN}✅ Rollback concluído com sucesso! Versão anterior restaurada.${RESET}"
    exit 0
fi

banner

# 1. Carrega Metadados de Versão
VERSION=$(cat "${SCRIPT_DIR}/VERSION" 2>/dev/null || echo "3.1.0")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "release")
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS="-s -w -X 'guardian/internal/version.Version=${VERSION}' -X 'guardian/internal/version.GitCommit=${GIT_COMMIT}' -X 'guardian/internal/version.BuildDate=${BUILD_DATE}'"

echo -e "📦 Alvo: ${BOLD}Guardian SRE v${VERSION}${RESET} (Commit: ${GIT_COMMIT}, Data: ${BUILD_DATE})"
echo ""

# 2. Pré-verificação e Encerramento Gracioso (Evita erro 'Text file busy' no Linux)
WAS_RUNNING=false
if command -v curl >/dev/null 2>&1; then
    if curl -s --connect-timeout 1 "http://localhost:8092/api/health" >/dev/null 2>&1; then
        WAS_RUNNING=true
    fi
fi

if [[ "${WAS_RUNNING}" == "true" ]]; then
    echo -e "${YELLOW}[INFO] Instância do Guardian em execução detectada. Pausando para atualização segura...${RESET}"
    if command -v systemctl >/dev/null 2>&1; then
        systemctl --user stop guardian 2>/dev/null || true
    fi
    pkill -f "guardian" 2>/dev/null || true
    sleep 1
fi

# 3. Compilação do Binário Go com Injeção de Versão
echo -e "${CYAN}[1/6] Compilando binário nativo Go (v${VERSION})...${RESET}"
mkdir -p bin
go build -ldflags="${LDFLAGS}" -o bin/guardian ./cmd/guardian
chmod +x bin/guardian
echo -e "${GREEN}[OK]${RESET} Binário compilado com sucesso (bin/guardian)."

# 4. Definição e Criação de Diretórios
echo -e "${CYAN}[2/6] Configurando diretórios (${MODE})...${RESET}"
mkdir -p "${BIN_DIR}" "${APPS_DIR}" "${ICONS_DIR}" "${SYSTEMD_DIR}" "${CONFIG_DIR}" "${DATA_DIR}" "${STATE_DIR}"

# 5. Backup do Binário Anterior (se existir) e Instalação dos Novos Executáveis
echo -e "${CYAN}[3/6] Instalando executáveis e utilitários...${RESET}"
if [[ -f "${BIN_DIR}/guardian" ]]; then
    cp -f "${BIN_DIR}/guardian" "${BIN_DIR}/guardian.bak" 2>/dev/null || true
fi

cp bin/guardian "${BIN_DIR}/guardian"
cp packaging/bin/guardian-app "${BIN_DIR}/guardian-app"
cp packaging/bin/guardian-ctl "${BIN_DIR}/guardian-ctl"
chmod +x "${BIN_DIR}/guardian" "${BIN_DIR}/guardian-app" "${BIN_DIR}/guardian-ctl"

# 6. Instalação de Ícones e Atalhos de Desktop
echo -e "${CYAN}[4/6] Registrando aplicativo no menu do sistema (Zorin/GNOME)...${RESET}"
cp packaging/desktop/guardian.svg "${ICONS_DIR}/guardian.svg"
if [[ -f "packaging/desktop/guardian.png" ]]; then
    mkdir -p "${ICONS_DIR%/*/*}/512x512/apps" 2>/dev/null || true
    cp packaging/desktop/guardian.png "${ICONS_DIR%/*/*}/512x512/apps/guardian.png" 2>/dev/null || true
fi
cp packaging/desktop/guardian.desktop "${APPS_DIR}/guardian.desktop"
chmod +x "${APPS_DIR}/guardian.desktop"

# 7. Serviço systemd e Preservação de Configurações
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

# Salva o caminho do repositório no arquivo de configuração para atualizações automáticas via 'guardian-ctl update'
if [[ -f "${CONFIG_DIR}/guardian.env" ]]; then
    if grep -q "^GUARDIAN_REPO_DIR=" "${CONFIG_DIR}/guardian.env" 2>/dev/null; then
        sed -i "s|^GUARDIAN_REPO_DIR=.*|GUARDIAN_REPO_DIR=\"${SCRIPT_DIR}\"|" "${CONFIG_DIR}/guardian.env"
    else
        echo "" >> "${CONFIG_DIR}/guardian.env"
        echo "# Repositório de código-fonte para auto-atualizações" >> "${CONFIG_DIR}/guardian.env"
        echo "GUARDIAN_REPO_DIR=\"${SCRIPT_DIR}\"" >> "${CONFIG_DIR}/guardian.env"
    fi
fi

# Persistência de Targets
if [[ ! -f "${DATA_DIR}/targets.json" && -f "${SCRIPT_DIR}/targets.json" ]]; then
    cp "${SCRIPT_DIR}/targets.json" "${DATA_DIR}/targets.json"
    echo -e "${GREEN}[OK]${RESET} Targets existentes copiados para ${DATA_DIR}/targets.json"
fi

# 8. Atualização de Caches e Ativação do Serviço
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
        systemctl --user restart guardian 2>/dev/null || true
        echo -e "${GREEN}[OK]${RESET} Serviço de segundo plano habilitado e iniciado no systemd do usuário."
    fi
else
    if command -v systemctl >/dev/null 2>&1; then
        systemctl daemon-reload 2>/dev/null || true
        systemctl enable guardian 2>/dev/null || true
        systemctl restart guardian 2>/dev/null || true
    fi
fi

# 9. Verificação de Saúde Pós-Instalação
echo ""
echo -e "${CYAN}🔍 Validando inicialização da nova versão...${RESET}"
HEALTHY=false
for i in {1..20}; do
    if curl -s --connect-timeout 1 "http://localhost:8092/api/health" >/dev/null 2>&1; then
        HEALTHY=true
        break
    fi
    sleep 0.2
done

if [[ "${HEALTHY}" == "true" ]]; then
    echo -e "${GREEN}[OK] API e Supervisor ativos e respondendo na porta 8092.${RESET}"
else
    echo -e "${YELLOW}[AVISO] Serviço em inicialização. Você pode acompanhar o status com 'guardian-ctl status'.${RESET}"
fi

echo ""
echo -e "${BOLD}${GREEN}====================================================================${RESET}"
if [[ "${ACTION}" == "update" ]]; then
    echo -e "${BOLD}${GREEN}  🎉 ATUALIZAÇÃO CONCLUÍDA COM SUCESSO! (v${VERSION})                  ${RESET}"
else
    echo -e "${BOLD}${GREEN}  🎉 INSTALAÇÃO CONCLUÍDA COM SUCESSO! (v${VERSION})                  ${RESET}"
fi
echo -e "${BOLD}${GREEN}====================================================================${RESET}"
echo ""
echo -e "O ${BOLD}Guardian SRE v${VERSION}${RESET} está pronto para uso na sua máquina."
echo ""
echo -e "📌 ${BOLD}Formas de Acesso Rápido:${RESET}"
echo -e "  1. ${CYAN}Menu de Aplicativos:${RESET} Procure por ${BOLD}Guardian SRE${RESET} e clique no ícone."
echo -e "  2. ${CYAN}Linha de Comando (CLI):${RESET}"
echo -e "     • Abrir janela desktop:   ${BOLD}guardian-app${RESET} ou ${BOLD}guardian-ctl open${RESET}"
echo -e "     • Status em tempo real:   ${BOLD}guardian-ctl status${RESET}"
echo -e "     • Atualizar nova versão:  ${BOLD}guardian-ctl update${RESET}"
echo -e "     • Reverter versão:        ${BOLD}guardian-ctl rollback${RESET}"
echo -e "     • Ver logs:               ${BOLD}guardian-ctl logs${RESET}"
echo ""
echo -e "📁 ${BOLD}Arquivos Principais:${RESET}"
echo -e "  • Binário Ativo:   ${CYAN}${BIN_DIR}/guardian${RESET}"
if [[ -f "${BIN_DIR}/guardian.bak" ]]; then
    echo -e "  • Backup Anterior: ${CYAN}${BIN_DIR}/guardian.bak${RESET}"
fi
echo -e "  • CLI Control:     ${CYAN}${BIN_DIR}/guardian-ctl${RESET}"
echo -e "  • Configuração:    ${CYAN}${CONFIG_DIR}/guardian.env${RESET}"
echo -e "  • Dados (targets): ${CYAN}${DATA_DIR}/targets.json${RESET}"
echo ""
if [[ ":$PATH:" != *":${BIN_DIR}:"* ]]; then
    echo -e "${YELLOW}⚠️  Aviso: O diretório ${BIN_DIR} não está no seu PATH.${RESET}"
    echo -e "Adicione a seguinte linha ao seu ${BOLD}~/.bashrc${RESET} ou ${BOLD}~/.zshrc${RESET}:"
    echo -e "  ${CYAN}export PATH=\"\$HOME/.local/bin:\$PATH\"${RESET}"
    echo ""
fi
