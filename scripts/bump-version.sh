#!/usr/bin/env bash
# =============================================================================
# bump-version.sh — Gerenciador de Versão Semântica do Guardian SRE
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="${SCRIPT_DIR}/VERSION"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

if [[ ! -f "${VERSION_FILE}" ]]; then
    echo "3.1.0" > "${VERSION_FILE}"
fi

CURRENT_VERSION=$(tr -d ' \t\r\n' < "${VERSION_FILE}")
ACTION="${1:-}"

if [[ -z "${ACTION}" ]]; then
    echo -e "${BOLD}${BLUE}🛡️  Guardian SRE — Versão Atual: ${GREEN}v${CURRENT_VERSION}${RESET}"
    echo ""
    echo "Uso: $0 [patch | minor | major | X.Y.Z]"
    echo ""
    echo "Exemplos:"
    echo -e "  $0 ${CYAN}patch${RESET}   → 3.1.0 ➔ 3.1.1 (correções e melhorias internas)"
    echo -e "  $0 ${CYAN}minor${RESET}   → 3.1.0 ➔ 3.2.0 (novas funcionalidades compatíveis)"
    echo -e "  $0 ${CYAN}major${RESET}   → 3.1.0 ➔ 4.0.0 (mudanças estruturais / breaking changes)"
    echo -e "  $0 ${CYAN}3.2.5${RESET}   → Define diretamente a versão desejada"
    exit 0
fi

# Divide versão em MAJOR, MINOR, PATCH
IFS='.' read -r MAJOR MINOR PATCH <<< "${CURRENT_VERSION}"
MAJOR="${MAJOR:-0}"
MINOR="${MINOR:-0}"
PATCH="${PATCH:-0}"

NEW_VERSION=""
case "${ACTION}" in
    patch)
        PATCH=$((PATCH + 1))
        NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
        ;;
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
        ;;
    [0-9]*.[0-9]*.[0-9]*)
        NEW_VERSION="${ACTION}"
        ;;
    *)
        echo -e "${RED}[ERRO] Ação ou formato inválido: '${ACTION}'. Use 'patch', 'minor', 'major' ou 'X.Y.Z'.${RESET}"
        exit 1
        ;;
esac

echo "${NEW_VERSION}" > "${VERSION_FILE}"

echo -e "${BOLD}${GREEN}====================================================================${RESET}"
echo -e "${BOLD}${GREEN}  ✨ VERSÃO ATUALIZADA: ${YELLOW}v${CURRENT_VERSION}${GREEN} ➔ ${CYAN}v${NEW_VERSION}${RESET}"
echo -e "${BOLD}${GREEN}====================================================================${RESET}"
echo ""
echo -e "O arquivo ${BOLD}VERSION${RESET} foi atualizado com sucesso."
echo ""
echo -e "📌 ${BOLD}Próximos Passos:${RESET}"
echo -e "  1. Para instalar a nova versão na sua máquina agora:"
echo -e "     ${CYAN}${BOLD}make update${RESET}  ou  ${CYAN}${BOLD}./install.sh --update${RESET}"
echo ""
echo -e "  2. Para criar a tag Git correspondente (opcional):"
echo -e "     ${CYAN}git add VERSION && git commit -m \"chore(release): bump to v${NEW_VERSION}\"${RESET}"
echo -e "     ${CYAN}git tag v${NEW_VERSION}${RESET}"
echo ""
