package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"guardian/internal/actions"
	"guardian/internal/api"
	"guardian/internal/config"
	"guardian/internal/providers/docker"
	"guardian/internal/providers/k8s"
	"guardian/internal/storage"
	"guardian/internal/supervisor"
	"guardian/web"
)

func main() {
	cfg := config.LoadConfig()

	portFlag := flag.Int("port", cfg.HTTPPort, "Porta HTTP do servidor")
	dryRunFlag := flag.Bool("dry-run", cfg.DryRun, "Executar em modo Dry-Run (sem criar chamados no Jira)")
	kubeFlag := flag.String("kubeconfig", cfg.KubeConfigPath, "Caminho do arquivo kubeconfig")
	dockerFlag := flag.String("docker-sock", cfg.DockerSocketPath, "Caminho do unix socket do Docker")
	flag.Parse()

	cfg.HTTPPort = *portFlag
	cfg.DryRun = *dryRunFlag
	cfg.KubeConfigPath = *kubeFlag
	cfg.DockerSocketPath = *dockerFlag

	printBanner(cfg)

	// 1. Inicializa Persistência de Targets
	store, err := storage.NewStorage(cfg.StoragePath)
	if err != nil {
		log.Fatalf("Erro ao inicializar storage: %v", err)
	}

	// 2. Inicializa Pipeline de Ações
	dedup := actions.NewDeduplicator(cfg.CachePath, cfg.CooldownMinutes)
	jira := actions.NewJiraClient(cfg)
	notifier := actions.NewNotifier(50)

	// 3. Inicializa Provedores
	k8sPool := k8s.NewClientPool(cfg.KubeConfigPath)
	k8sDisc := k8s.NewK8sDiscovery(k8sPool, cfg.KubeConfigPath)

	dockerPool := docker.NewDockerPool(cfg.DockerSocketPath)
	dockerDisc := docker.NewDockerDiscovery(dockerPool, store)

	// 4. Inicializa Supervisor de Targets Híbrido
	superv := supervisor.NewSupervisor(k8sPool, dockerPool, store, dedup, jira, notifier)
	superv.Start()

	// 5. Inicializa Servidor HTTP e Rotas da API
	apiServer := api.NewServer(cfg, store, superv, notifier, k8sDisc, dockerDisc)

	// Servir UI Web estática embutida (web.Assets)
	mainMux := http.NewServeMux()
	mainMux.Handle("/api/", apiServer.Handler())

	// Sub-FS para servir o index.html na raiz sem cache agressivo do navegador
	webFS, err := fs.Sub(web.Assets, ".")
	if err != nil {
		log.Fatalf("Erro ao carregar assets web: %v", err)
	}
	fileServer := http.FileServer(http.FS(webFS))
	mainMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		fileServer.ServeHTTP(w, r)
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", cfg.HTTPPort),
		Handler: mainMux,
	}

	// Inicia servidor HTTP em goroutine
	go func() {
		log.Printf("🌐 [Guardian Engine] Interface Web & API disponíveis em: http://localhost:%d", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erro no servidor HTTP: %v", err)
		}
	}()

	// Captura sinais de interrupção para encerramento gracioso
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("\n🛑 Encerrando Guardian Supervisor Híbrido...")
	superv.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Aviso no shutdown do HTTP: %v", err)
	}

	log.Println("👋 Guardian finalizado com segurança.")
}

func printBanner(cfg *config.Config) {
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║       GUARDIAN - SUPERVISOR HÍBRIDO (KUBERNETES & DOCKER)        ║")
	fmt.Println("║               Target Supervisor & Jira Automation                ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Printf(" [•] Jira Base URL   : %s\n", cfg.JiraBaseURL)
	fmt.Printf(" [•] Jira Usuário    : %s\n", cfg.JiraUser)
	fmt.Printf(" [•] Jira Projeto    : %s\n", cfg.JiraProjectKey)
	fmt.Printf(" [•] Modo Dry-Run    : %t\n", cfg.DryRun)
	fmt.Printf(" [•] Cooldown Minutos: %d min\n", cfg.CooldownMinutes)
	fmt.Printf(" [•] KubeConfig      : %s\n", cfg.KubeConfigPath)
	fmt.Printf(" [•] Docker Socket   : %s\n", cfg.DockerSocketPath)
	fmt.Printf(" [•] Porta HTTP      : %d\n", cfg.HTTPPort)
	fmt.Println("────────────────────────────────────────────────────────────────────")
}
