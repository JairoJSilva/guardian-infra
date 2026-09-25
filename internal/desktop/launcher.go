package desktop

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	if runtime.GOOS == "windows" {
		winCandidates := []string{
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("LocalAppData"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("LocalAppData"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		}
		for _, path := range winCandidates {
			if path != "" {
				if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
					return path, true
				}
			}
		}
		for _, name := range []string{"msedge.exe", "chrome.exe", "brave.exe", "msedge", "chrome"} {
			if path, err := exec.LookPath(name); err == nil && path != "" {
				return path, true
			}
		}
	}

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
	var profileDir string
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData, _ = os.UserConfigDir()
		}
		if appData == "" {
			appData = os.Getenv("USERPROFILE")
		}
		profileDir = filepath.Join(appData, "Guardian", "desktop-profile")
	} else {
		home := os.Getenv("SNAP_REAL_HOME")
		if home == "" {
			home = os.Getenv("HOME")
		}
		profileDir = filepath.Join(home, ".config", "guardian", "desktop-profile")
	}
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

	// 3. Fallback no Windows (cmd /c start)
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "start", appURL)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err == nil {
			log.Printf("🖥️ [Desktop Window] URL aberta via default browser (cmd /c start)")
			return nil
		}
	}

	// 4. Fallback para xdg-open (Linux)
	if xdgPath, err := exec.LookPath("xdg-open"); err == nil {
		cmd := exec.Command(xdgPath, appURL)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err == nil {
			log.Printf("🖥️ [Desktop Window] Janela aberta via xdg-open")
			return nil
		}
	}

	// 5. Fallback para sensible-browser (Debian/Ubuntu)
	if sbPath, err := exec.LookPath("sensible-browser"); err == nil {
		cmd := exec.Command(sbPath, appURL)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err == nil {
			log.Printf("🖥️ [Desktop Window] Janela aberta via sensible-browser")
			return nil
		}
	}

	return fmt.Errorf("nenhum navegador ou launcher gráfico encontrado no sistema")
}
