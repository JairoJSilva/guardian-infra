package desktop

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ChromiumCandidates são os navegadores baseados em Chromium compatíveis com o modo --app
var ChromiumCandidates = []string{
	"google-chrome",
	"google-chrome-stable",
	"chromium",
	"chromium-browser",
	"brave-browser",
	"microsoft-edge",
	"microsoft-edge-stable",
	"vivaldi",
}

// FindAppBrowser descobre o melhor executável para abrir a janela nativa sem barras do navegador
func FindAppBrowser() (string, bool) {
	for _, name := range ChromiumCandidates {
		if path, err := exec.LookPath(name); err == nil && path != "" {
			return path, true
		}
	}
	return "", false
}

// WaitForServerReady aguarda o endpoint /api/health responder 200 OK antes de abrir a janela
func WaitForServerReady(baseURL string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	healthURL := strings.TrimRight(baseURL, "/") + "/api/health"

	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return true
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// LaunchAppWindow abre a aplicação em uma janela desktop nativa (Modo App / Standalone)
func LaunchAppWindow(appURL string) error {
	// Determina diretório de perfil isolado para não misturar sessões
	home := os.Getenv("SNAP_REAL_HOME")
	if home == "" {
		home = os.Getenv("HOME")
	}
	profileDir := filepath.Join(home, ".config", "guardian", "desktop-profile")
	_ = os.MkdirAll(profileDir, 0755)

	// 1. Tenta navegador baseado em Chromium com --app (janela nativa sem barra de URL/abas)
	if browserPath, ok := FindAppBrowser(); ok {
		args := []string{
			fmt.Sprintf("--app=%s", appURL),
			fmt.Sprintf("--user-data-dir=%s", profileDir),
			"--class=guardian",
			"--name=guardian",
			"--no-first-run",
			"--no-default-browser-check",
		}
		cmd := exec.Command(browserPath, args...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err == nil {
			log.Printf("🖥️ [Desktop Window] Janela de aplicativo aberta com sucesso via: %s", filepath.Base(browserPath))
			return nil
		}
	}

	// 2. Tenta Firefox em nova janela
	if ffPath, err := exec.LookPath("firefox"); err == nil {
		cmd := exec.Command(ffPath, "--new-window", appURL)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err == nil {
			log.Printf("🖥️ [Desktop Window] Janela aberta via Firefox (--new-window)")
			return nil
		}
	}

	// 3. Fallback para xdg-open
	if xdgPath, err := exec.LookPath("xdg-open"); err == nil {
		cmd := exec.Command(xdgPath, appURL)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err == nil {
			log.Printf("🖥️ [Desktop Window] Janela aberta via xdg-open")
			return nil
		}
	}

	// 4. Fallback para sensible-browser
	if sbPath, err := exec.LookPath("sensible-browser"); err == nil {
		cmd := exec.Command(sbPath, appURL)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err == nil {
			log.Printf("🖥️ [Desktop Window] Janela aberta via sensible-browser")
			return nil
		}
	}

	return fmt.Errorf("nenhum navegador ou launcher gráfico (xdg-open) encontrado no sistema")
}
