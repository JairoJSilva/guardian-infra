# =============================================================================
# Makefile — Guardian Autonomous Enterprise SRE Platform (Native Linux App)
# =============================================================================

SHELL := /bin/bash
VERSION := 3.0.0
BIN := bin/guardian

.PHONY: all build install install-system uninstall deb run app clean test

all: build

## Compila o binário standalone nativo em Go
build:
	@echo "⚙️ Compilando Guardian $(VERSION)..."
	@mkdir -p bin
	go build -ldflags="-s -w" -o $(BIN) ./cmd/guardian
	@chmod +x $(BIN)
	@echo "✅ Binário compilado em $(BIN)"

## Instala no desktop do usuário local (~/.local) sem necessidade de sudo
install: build
	@chmod +x scripts/install-desktop.sh packaging/bin/*
	@./scripts/install-desktop.sh

## Instala globalmente no sistema (/usr/local) com privilégios de root
install-system: build
	@chmod +x scripts/install-desktop.sh packaging/bin/*
	@sudo ./scripts/install-desktop.sh --system

## Desinstala o aplicativo, atalhos do desktop e serviços
uninstall:
	@chmod +x scripts/uninstall-desktop.sh
	@./scripts/uninstall-desktop.sh

## Gera o pacote de distribuição nativo Debian/Ubuntu (.deb)
deb: build
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
