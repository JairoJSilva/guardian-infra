# 📝 RMT-20260921-01: Transformação do Guardian em Aplicativo Local Nativo de Linux (Desktop & Daemon)

> **RMT (Registro de Modificação Técnica)**  
> **Status**: Aplicada  
> **Data**: 2026-09-21  
> **Autor / Agente Responsável**: Antigravity Platform Engineering  
> **Tipo de Mudança**: Architecture / Desktop Integration / Packaging / Systemd  
> **Versão Afetada**: v3.0.0-Enterprise-NativeLinux  

---

## 1. 🎯 Contexto e Motivação
Para transformar a plataforma Guardian em uma aplicação local nativa de Linux (compatível com Ubuntu, Zorin OS, Debian e outros ambientes com GNOME/X11/Wayland), implementou-se a infraestrutura completa de aplicação desktop:
1. Integração com o menu de aplicativos do Linux via FreeDesktop `.desktop` e ícone SVG em alta resolução.
2. Modo de janela de aplicativo nativa (App-Mode borderless via Chromium/Brave/Edge/Firefox) desacoplada do navegador padrão, comportando-se 100% como app desktop independente no dock e Alt-Tab.
3. Resolução inteligente de caminhos conforme a especificação XDG Base Directory (`~/.config/guardian` e `~/.local/share/guardian`).
4. Utilitário de linha de comando (`guardian-ctl`) e serviço systemd de usuário (`guardian.service`).
5. Scripts de instalação local sem root (`scripts/install-desktop.sh`), desinstalação limpa (`scripts/uninstall-desktop.sh`) e gerador de pacote `.deb` (`scripts/build-deb.sh`).

---

## 2. 📁 Arquivos e Componentes Afetados

| Arquivo / Caminho | Tipo de Alteração | Descrição Resumida |
|:---|:---|:---|
| `internal/config/config.go` | Modificado | Suporte à especificação XDG, resolução de home real sob snap/sudo e paths resilientes |
| `internal/desktop/launcher.go` | Criado | Lançador nativo de janelas em modo App/Kiosk para Linux |
| `cmd/guardian/main.go` | Modificado | Suporte a flags `-gui`, `-version` e subcomandos `guardian gui`, `guardian open` |
| `internal/api/server.go` | Modificado | Atualização de telemetria e versão no endpoint `/api/health` |
| `packaging/desktop/guardian.desktop` | Criado | Entrada Desktop oficial do FreeDesktop com ações rápidas |
| `packaging/desktop/guardian.svg` | Criado | Ícone vetorial SVG escalável estilizado Obsidian-Gold-Jade |
| `packaging/systemd/guardian.service` | Criado | Unidade systemd user para background daemon |
| `packaging/bin/guardian-app` | Criado | Launcher desktop com detecção de daemon e abertura de janela nativa |
| `packaging/bin/guardian-ctl` | Criado | Painel de controle via terminal (start, stop, status, logs, open) |
| `scripts/install-desktop.sh` | Criado | Instalador de um clique para desktop de usuário ou sistema |
| `scripts/uninstall-desktop.sh` | Criado | Desinstalador limpo |
| `scripts/build-deb.sh` | Criado | Gerador de pacote Debian `.deb` nativo via `dpkg-deb` ou `ar+tar` |
| `Makefile` | Criado | Metas de build, install, deb, run, app, uninstall |

---

## 3. ⚙️ Detalhamento Técnico das Modificações

### 3.1. Conformidade XDG Base Directory
Quando executado fora da pasta do repositório (ex: lançado pelo menu do Zorin OS ou GNOME), a aplicação agora:
- Lê `.env` local caso exista; caso contrário busca em `~/.config/guardian/guardian.env` ou `/etc/guardian/guardian.env`.
- Salva e lê `targets.json` e `guardian_cache.json` em `~/.local/share/guardian/`.

### 3.2. Experiência de Janela Desktop Nativa
O lançador `guardian-app` detecta navegadores baseados em Chromium e executa com os parâmetros `--app=http://localhost:8092 --user-data-dir=~/.config/guardian/desktop-profile --class=guardian`. Isso produz uma janela isolada, sem barra de endereços, sem abas e com ícone próprio no dock do Zorin/GNOME.

### 3.3. Empacotamento Debian (.deb)
O script `scripts/build-deb.sh` compila e monta `dist/guardian_3.0.0_amd64.deb` contendo executáveis, ícone escalável, atalho desktop e scripts pós-instalação, permitindo instalação via `sudo dpkg -i`.

---

## 4. 🧪 Testes e Validação
- [x] Compilação do binário Go em `bin/guardian` com sucesso.
- [x] Teste de execução com `./bin/guardian -version` validando versão `v3.0.0-Enterprise-NativeLinux`.
- [x] Teste do endpoint `/api/health` retornando `status: UP` e versão `3.0.0-enterprise-linux`.
- [x] Teste de geração e extração do pacote `.deb` em `dist/guardian_3.0.0_amd64.deb`.
