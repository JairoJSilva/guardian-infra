#!/usr/bin/env bash
# =============================================================================
# build-deb.sh — Empacotador Debian / Ubuntu (.deb) do Guardian SRE
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${SCRIPT_DIR}"

VERSION=$(cat "${SCRIPT_DIR}/VERSION" 2>/dev/null || echo "3.1.0")
ARCH="amd64"
PACKAGE_NAME="guardian"
BUILD_ROOT="${SCRIPT_DIR}/dist/deb_build"
DEB_FILE="${SCRIPT_DIR}/dist/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"

echo "🔨 [Debian Packager] Limpando diretórios de build..."
rm -rf "${BUILD_ROOT}"
mkdir -p "${BUILD_ROOT}/DEBIAN" \
         "${BUILD_ROOT}/usr/local/bin" \
         "${BUILD_ROOT}/usr/share/applications" \
         "${BUILD_ROOT}/usr/share/icons/hicolor/scalable/apps" \
         "${BUILD_ROOT}/etc/guardian" \
         "${BUILD_ROOT}/etc/systemd/system" \
         "${SCRIPT_DIR}/dist"

echo "⚙️ [Debian Packager] Compilando binário Go v${VERSION}..."
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "release")
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS="-s -w -X 'guardian/internal/version.Version=${VERSION}' -X 'guardian/internal/version.GitCommit=${GIT_COMMIT}' -X 'guardian/internal/version.BuildDate=${BUILD_DATE}'"
go build -ldflags="${LDFLAGS}" -o "${SCRIPT_DIR}/bin/guardian" ./cmd/guardian

echo "📦 [Debian Packager] Estruturando pacote..."
cp "${SCRIPT_DIR}/bin/guardian" "${BUILD_ROOT}/usr/local/bin/guardian"
cp "${SCRIPT_DIR}/packaging/bin/guardian-app" "${BUILD_ROOT}/usr/local/bin/guardian-app"
cp "${SCRIPT_DIR}/packaging/bin/guardian-ctl" "${BUILD_ROOT}/usr/local/bin/guardian-ctl"
chmod +x "${BUILD_ROOT}/usr/local/bin/"*

cp "${SCRIPT_DIR}/packaging/desktop/guardian.desktop" "${BUILD_ROOT}/usr/share/applications/"
cp "${SCRIPT_DIR}/packaging/desktop/guardian.svg" "${BUILD_ROOT}/usr/share/icons/hicolor/scalable/apps/"
cp "${SCRIPT_DIR}/.env.example" "${BUILD_ROOT}/etc/guardian/guardian.env.example"

# DEBIAN/control
cat <<EOF > "${BUILD_ROOT}/DEBIAN/control"
Package: ${PACKAGE_NAME}
Version: ${VERSION}
Section: devel
Priority: optional
Architecture: ${ARCH}
Maintainer: Flowti Platform Engineering <platform@flowti.com.br>
Description: Guardian Autonomous Enterprise SRE & Incident Automation Platform
 Guardian é uma plataforma autônoma de observabilidade, supervisão de
 infraestrutura Kubernetes e Docker e resposta automatizada a incidentes
 integrado ao Atlassian Jira.
EOF

# DEBIAN/postinst
cat <<'EOF' > "${BUILD_ROOT}/DEBIAN/postinst"
#!/bin/sh
set -e
if [ ! -f /etc/guardian/guardian.env ] && [ -f /etc/guardian/guardian.env.example ]; then
    cp /etc/guardian/guardian.env.example /etc/guardian/guardian.env
    chmod 600 /etc/guardian/guardian.env
fi
if which update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications || true
fi
if which gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -f -t /usr/share/icons/hicolor || true
fi
exit 0
EOF
chmod 755 "${BUILD_ROOT}/DEBIAN/postinst"

# DEBIAN/prerm
cat <<'EOF' > "${BUILD_ROOT}/DEBIAN/prerm"
#!/bin/sh
set -e
pkill -f "guardian" 2>/dev/null || true
exit 0
EOF
chmod 755 "${BUILD_ROOT}/DEBIAN/prerm"

echo "📦 [Debian Packager] Gerando arquivo .deb..."
BUILD_SUCCESS=false

# Tenta primeiro com dpkg-deb se disponível e com permissão
if command -v dpkg-deb >/dev/null 2>&1; then
    if dpkg-deb --build --root-owner-group "${BUILD_ROOT}" "${DEB_FILE}" 2>/dev/null; then
        BUILD_SUCCESS=true
    fi
fi

# Fallback nativo via ar + tar (compatível com snap e containers)
if [ "$BUILD_SUCCESS" = false ]; then
    echo "ℹ️  Construindo .deb diretamente via ar + tar..."
    STAGE_DIR="${SCRIPT_DIR}/dist/deb_stage"
    rm -rf "${STAGE_DIR}"
    mkdir -p "${STAGE_DIR}"

    echo "2.0" > "${STAGE_DIR}/debian-binary"

    # control.tar.gz
    (cd "${BUILD_ROOT}/DEBIAN" && tar --numeric-owner --owner=0 --group=0 -czf "${STAGE_DIR}/control.tar.gz" .)

    # data.tar.gz
    (cd "${BUILD_ROOT}" && tar --numeric-owner --owner=0 --group=0 --exclude="./DEBIAN" -czf "${STAGE_DIR}/data.tar.gz" usr etc)

    # Cria pacote .deb usando ar
    (cd "${STAGE_DIR}" && ar -rcD "${DEB_FILE}" debian-binary control.tar.gz data.tar.gz)
    rm -rf "${STAGE_DIR}"
    BUILD_SUCCESS=true
fi

rm -rf "${BUILD_ROOT}"

if [ -f "${DEB_FILE}" ]; then
    echo "🎉 Pacote Debian criado com sucesso em: ${DEB_FILE}"
    echo "Tamanho do pacote: $(du -h "${DEB_FILE}" | cut -f1)"
    echo "Para instalar no Ubuntu/Debian/Zorin OS:"
    echo "  sudo dpkg -i dist/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"
fi
