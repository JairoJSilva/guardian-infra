package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"guardian/internal/actions"
	"guardian/internal/config"
	"guardian/internal/domain"
	"guardian/internal/providers/docker"
	"guardian/internal/providers/k8s"
	"guardian/internal/storage"
	"guardian/internal/supervisor"
)

type Server struct {
	cfg             *config.Config
	storage         *storage.Storage
	supervisor      *supervisor.Supervisor
	notifier        *actions.Notifier
	k8sDiscovery    *k8s.K8sDiscovery
	dockerDiscovery *docker.DockerDiscovery
	mux             *http.ServeMux
}

func NewServer(
	cfg *config.Config,
	store *storage.Storage,
	superv *supervisor.Supervisor,
	notif *actions.Notifier,
	k8sDisc *k8s.K8sDiscovery,
	dockerDisc *docker.DockerDiscovery,
) *Server {
	s := &Server{
		cfg:             cfg,
		storage:         store,
		supervisor:      superv,
		notifier:        notif,
		k8sDiscovery:    k8sDisc,
		dockerDiscovery: dockerDisc,
		mux:             http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.corsMiddleware(s.mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/targets", s.handleTargets)
	s.mux.HandleFunc("/api/targets/", s.handleTargetByID)
	s.mux.HandleFunc("/api/discovery/environments", s.handleDiscoveryEnvironments)
	s.mux.HandleFunc("/api/discovery/docker/inspect", s.handleDockerInspect)
	s.mux.HandleFunc("/api/events/live", s.handleLiveEvents)
	s.mux.HandleFunc("/api/events/history", s.handleEventsHistory)
	s.mux.HandleFunc("/api/simulate", s.handleSimulate)
	s.mux.HandleFunc("/api/settings/toggle-dry-run", s.handleToggleDryRun)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "UP",
		"version":     "2.0.0-hybrid",
		"mode":        "hybrid-supervisor",
		"dry_run":     s.cfg.IsDryRun(),
		"jira_url":    s.cfg.JiraBaseURL,
		"project_key": s.cfg.JiraProjectKey,
		"time":        time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleToggleDryRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Método não permitido")
		return
	}

	newDryRun := s.cfg.ToggleDryRun()
	modeName := "PRODUÇÃO REAL (Chamados reais no Jira)"
	if newDryRun {
		modeName = "SIMULAÇÃO (Dry-Run Ativo - sem chamados reais)"
	}
	log.Printf("[API Settings] Modo de operação alterado dinamicamente pela interface para: %s", modeName)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"dry_run": newDryRun,
		"mode":    modeName,
		"message": "Modo de operação alternado com sucesso",
	})
}

func (s *Server) handleTargets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		targets := s.storage.List()
		respondJSON(w, http.StatusOK, targets)

	case http.MethodPost:
		var t domain.Target
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			respondError(w, http.StatusBadRequest, "Payload inválido: "+err.Error())
			return
		}

		if t.Name == "" || t.Endpoint == "" || len(t.Scopes) == 0 {
			respondError(w, http.StatusBadRequest, "Campos obrigatórios: name, endpoint, scopes")
			return
		}

		if t.Type != domain.EnvKubernetes && t.Type != domain.EnvDocker {
			t.Type = domain.EnvKubernetes
		}
		t.Status = domain.StatusActive
		if t.Rules.PollIntervalSeconds <= 0 {
			t.Rules.PollIntervalSeconds = 15
		}

		if err := s.supervisor.AddTarget(&t); err != nil {
			respondError(w, http.StatusInternalServerError, "Erro ao salvar target: "+err.Error())
			return
		}

		respondJSON(w, http.StatusCreated, t)

	default:
		respondError(w, http.StatusMethodNotAllowed, "Método não permitido")
	}
}

func (s *Server) handleTargetByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/targets/")
	parts := strings.Split(path, "/")
	targetID := parts[0]

	if targetID == "" {
		respondError(w, http.StatusBadRequest, "Target ID não especificado")
		return
	}

	if len(parts) == 2 && parts[1] == "toggle" && r.Method == http.MethodPost {
		t, err := s.supervisor.ToggleTargetStatus(targetID)
		if err != nil {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, t)
		return
	}

	switch r.Method {
	case http.MethodGet:
		t, err := s.storage.Get(targetID)
		if err != nil {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, t)

	case http.MethodDelete:
		if err := s.supervisor.DeleteTarget(targetID); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"message": "Target removido com sucesso"})

	default:
		respondError(w, http.StatusMethodNotAllowed, "Método não permitido")
	}
}

func (s *Server) handleDiscoveryEnvironments(w http.ResponseWriter, r *http.Request) {
	k8sEnvs := s.k8sDiscovery.DiscoverEnvironments()
	dockerEnvs := s.dockerDiscovery.DiscoverEnvironments()

	all := append(k8sEnvs, dockerEnvs...)
	respondJSON(w, http.StatusOK, all)
}

func (s *Server) handleDockerInspect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Método não permitido")
		return
	}

	endpoint := strings.TrimSpace(r.URL.Query().Get("endpoint"))
	if endpoint == "" {
		endpoint = "/var/run/docker.sock"
	}

	res, err := s.dockerDiscovery.InspectEndpoint(r.Context(), endpoint)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, res)
}

func (s *Server) handleEventsHistory(w http.ResponseWriter, r *http.Request) {
	history := s.notifier.GetHistory()
	respondJSON(w, http.StatusOK, history)
}

func (s *Server) handleLiveEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming SSE não suportado", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := s.notifier.Subscribe()
	defer s.notifier.Unsubscribe(ch)

	// Heartbeat / ping inicial
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case event := <-ch:
			data, err := json.Marshal(event)
			if err == nil {
				fmt.Fprintf(w, "data: %s\n\n", string(data))
				flusher.Flush()
			}
		}
	}
}

func (s *Server) handleSimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Método deve ser POST")
		return
	}

	var req struct {
		Type        domain.EnvironmentType `json:"type"`
		Environment string                 `json:"environment"`
		Scope       string                 `json:"scope"`
		EntityName  string                 `json:"entity_name"`
		Reason      string                 `json:"reason"`
		ExitCode    int                    `json:"exit_code"`
		Logs        string                 `json:"logs"`
		Severity    string                 `json:"severity"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Type == "" {
		req.Type = domain.EnvKubernetes
	}
	if req.EntityName == "" {
		if req.Type == domain.EnvKubernetes {
			req.EntityName = "payment-service-84f98d7b"
			req.Environment = "k8s-cluster"
			req.Scope = "billing"
			req.Reason = "CrashLoopBackOff"
			req.ExitCode = 1
			req.Severity = "WARNING"
			req.Logs = "[FATAL] Conexão recusada no banco de dados após 3 tentativas\n[ERROR] panic: runtime error: invalid memory address or nil pointer dereference"
		} else {
			req.EntityName = "redis-cache-main-1"
			req.Environment = "docker-local"
			req.Scope = "redis-cache"
			req.Reason = "OOMKilled"
			req.ExitCode = 137
			req.Severity = "CRITICAL"
			req.Logs = "1:M 18 Sep 17:15:00.123 # Out of memory: cannot allocate 524288 bytes.\n1:M 18 Sep 17:15:00.124 # Emergency logging completed."
		}
	}

	event := &domain.IncidentEvent{
		ID:          fmt.Sprintf("evt-sim-%d", time.Now().UnixNano()),
		Type:        req.Type,
		TargetID:    "target-simulated",
		Environment: req.Environment,
		Scope:       req.Scope,
		EntityName:  req.EntityName,
		Image:       req.EntityName + ":v1.2",
		Reason:      req.Reason,
		ExitCode:    req.ExitCode,
		Logs:        req.Logs,
		Timestamp:   time.Now(),
		Severity:    req.Severity,
	}

	s.supervisor.InjectSimulatedIncident(event)
	log.Printf("[API Simulate] Incidente simulado injetado com sucesso: %s (%s)", event.EntityName, event.Reason)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Incidente simulado disparado com sucesso!",
		"event":   event,
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
