# =============================================================================
# Makefile — Guardian Autonomous Enterprise SRE Platform (Native Linux App)
# =============================================================================

SHELL := /bin/bash
VERSION := $(shell cat VERSION 2>/dev/null || echo "3.1.0")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X 'guardian/internal/version.Version=$(VERSION)' -X 'guardian/internal/version.GitCommit=$(GIT_COMMIT)' -X 'guardian/internal/version.BuildDate=$(BUILD_DATE)'
BIN := bin/guardian

.PHONY: all build install install-system update rollback uninstall purge status version deb run app clean bump-patch bump-minor bump-major help

all: build

## Compila o binário standalone nativo em Go com metadados de versão
build:
	@echo "⚙️  Compilando Guardian v$(VERSION) ($(GIT_COMMIT), $(BUILD_DATE))..."
	@mkdir -p bin
	go build -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd/guardian
	@chmod +x $(BIN)
	@echo "✅ Binário compilado com sucesso em $(BIN)"

## Instala no desktop do usuário local (~/.local) sem necessidade de sudo
install:
	@chmod +x install.sh scripts/*.sh packaging/bin/*
	@./install.sh --install

## Atualiza para a nova versão na máquina local com backup e auto-restart
update:
	@chmod +x install.sh scripts/*.sh packaging/bin/*
	@./install.sh --update

## Reverte para a versão anterior (.bak) em caso de emergência
rollback:
	@chmod +x install.sh scripts/*.sh packaging/bin/*
	@./install.sh --rollback

## Verifica status e telemetria da aplicação instalada
status:
	@chmod +x install.sh
	@./install.sh --status

## Exibe versão semântica e commit
version:
	@echo "Guardian SRE v$(VERSION) (commit: $(GIT_COMMIT), data: $(BUILD_DATE))"

## Incrementa versão de patch (ex: 3.1.0 -> 3.1.1)
bump-patch:
	@chmod +x scripts/bump-version.sh
	@./scripts/bump-version.sh patch

## Incrementa versão minor (ex: 3.1.0 -> 3.2.0)
bump-minor:
	@chmod +x scripts/bump-version.sh
	@./scripts/bump-version.sh minor

## Incrementa versão major (ex: 3.1.0 -> 4.0.0)
bump-major:
	@chmod +x scripts/bump-version.sh
	@./scripts/bump-version.sh major

## Instala globalmente no sistema (/usr/local) com privilégios de root
install-system:
	@chmod +x install.sh scripts/*.sh packaging/bin/*
	@./install.sh --system

## Desinstala o aplicativo, atalhos do desktop e serviços (preserva configs)
uninstall:
	@chmod +x install.sh scripts/*.sh
	@./install.sh --uninstall

## Desinstala completamente eliminando também configurações e bancos
purge:
	@chmod +x install.sh scripts/*.sh
	@./install.sh --purge

## Gera o pacote de distribuição nativo Debian/Ubuntu (.deb)
deb:
	@chmod +x scripts/build-deb.sh
	@./scripts/build-deb.sh

## Executa o servidor Guardian no terminal em modo desenvolvimento
run: build
	@./$(BIN)

## Inicia o Guardian abrindo diretamente a janela de aplicativo desktop nativa
app: build
	@./$(BIN) -gui

## Limpa artefatos temporários e binários compilados
clean:
	@rm -rf bin dist guardian_cache.json
	@echo "🧹 Limpeza concluída."
