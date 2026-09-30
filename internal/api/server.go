package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"guardian/internal/actions"
	"guardian/internal/analyzer"
	"guardian/internal/config"
	"guardian/internal/domain"
	"guardian/internal/providers/docker"
	"guardian/internal/providers/k8s"
	"guardian/internal/storage"
	"guardian/internal/supervisor"
	"guardian/internal/version"
)

type Server struct {
	cfg             *config.Config
	storage         *storage.Storage
	integStorage    *storage.IntegrationStorage
	supervisor      *supervisor.Supervisor
	notifier        *actions.Notifier
	k8sDiscovery    *k8s.K8sDiscovery
	dockerDiscovery *docker.DockerDiscovery
	mux             *http.ServeMux
}

func NewServer(
	cfg *config.Config,
	store *storage.Storage,
	integStore *storage.IntegrationStorage,
	superv *supervisor.Supervisor,
	notif *actions.Notifier,
	k8sDisc *k8s.K8sDiscovery,
	dockerDisc *docker.DockerDiscovery,
) *Server {
	s := &Server{
		cfg:             cfg,
		storage:         store,
		integStorage:    integStore,
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
	s.mux.HandleFunc("/api/integrations", s.handleIntegrations)
	s.mux.HandleFunc("/api/integrations/", s.handleIntegrationByID)
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
		"version":     version.Version,
		"build":       version.FullVersion(),
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

	if len(parts) == 2 && parts[1] == "toggle-jira" && r.Method == http.MethodPost {
		t, err := s.storage.Get(targetID)
		if err != nil {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		t.Actions.CreateJiraIssue = !t.Actions.CreateJiraIssue
		t.UpdatedAt = time.Now()
		if err := s.storage.Save(t); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
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
	var (
		k8sEnvs    []domain.EnvironmentInfo
		dockerEnvs []domain.EnvironmentInfo
		wg         sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		k8sEnvs = s.k8sDiscovery.DiscoverEnvironments()
	}()
	go func() {
		defer wg.Done()
		dockerEnvs = s.dockerDiscovery.DiscoverEnvironments()
	}()
	wg.Wait()

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
		simSuffix := fmt.Sprintf("%04d", time.Now().UnixNano()%10000)
		if req.Type == domain.EnvKubernetes {
			req.EntityName = fmt.Sprintf("payment-service-%s", simSuffix)
			req.Environment = "k8s-cluster"
			req.Scope = "billing"
			req.Reason = "CrashLoopBackOff"
			req.ExitCode = 1
			req.Severity = "WARNING"
			req.Logs = "[FATAL] Conexão recusada no banco de dados após 3 tentativas\n[ERROR] panic: runtime error: invalid memory address or nil pointer dereference"
		} else {
			req.EntityName = fmt.Sprintf("redis-cache-%s", simSuffix)
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

	if analysis := analyzer.AnalyzeIncident(event); analysis != nil {
		event.AnalysisCategory = analysis.Category
		event.AnalysisSummary = analysis.Summary
		event.RootCause = analysis.RootCause
		event.SuggestedFix = analysis.SuggestedFix
		event.ActionSteps = analysis.ActionSteps
		event.SuggestedCommands = analysis.SuggestedCommands
	}

	s.supervisor.InjectSimulatedIncident(event)
	log.Printf("[API Simulate] Incidente simulado injetado com sucesso: %s (%s)", event.EntityName, event.Reason)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Incidente simulado disparado com sucesso!",
		"event":   event,
	})
}

func (s *Server) handleIntegrations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list := s.integStorage.ListSafe()
		respondJSON(w, http.StatusOK, list)

	case http.MethodPost:
		var item domain.TicketingIntegration
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			respondError(w, http.StatusBadRequest, "Payload inválido: "+err.Error())
			return
		}
		item.Name = strings.TrimSpace(item.Name)
		item.BaseURL = strings.TrimSpace(item.BaseURL)
		if item.Name == "" || item.BaseURL == "" {
			respondError(w, http.StatusBadRequest, "Nome e URL Base são obrigatórios")
			return
		}
		if item.Type != domain.ProviderJira && item.Type != domain.ProviderGLPI && item.Type != domain.ProviderMovidesk {
			item.Type = domain.ProviderJira
		}
		if item.AuthType == "" {
			item.AuthType = "BASIC"
		}
		if item.ID == "" {
			item.ID = fmt.Sprintf("portal-%s-%d", strings.ToLower(string(item.Type)), time.Now().UnixNano()%1000000)
		}
		item.CreatedAt = time.Now()
		item.UpdatedAt = time.Now()

		if err := s.integStorage.Save(&item); err != nil {
			respondError(w, http.StatusInternalServerError, "Erro ao salvar integração: "+err.Error())
			return
		}
		respondJSON(w, http.StatusCreated, item.ToSafe())

	default:
		respondError(w, http.StatusMethodNotAllowed, "Método não permitido")
	}
}

func (s *Server) handleIntegrationByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, http.StatusBadRequest, "ID da integração ausente")
		return
	}
	id := parts[2]

	if id == "test" {
		s.handleTestIntegration(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		item, err := s.integStorage.Get(id)
		if err != nil {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, item.ToSafe())

	case http.MethodPut:
		existing, err := s.integStorage.Get(id)
		if err != nil {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}

		var update domain.TicketingIntegration
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			respondError(w, http.StatusBadRequest, "Payload inválido: "+err.Error())
			return
		}

		if strings.TrimSpace(update.Name) != "" {
			existing.Name = strings.TrimSpace(update.Name)
		}
		if strings.TrimSpace(update.BaseURL) != "" {
			existing.BaseURL = strings.TrimSpace(update.BaseURL)
		}
		if update.Type != "" {
			existing.Type = update.Type
		}
		if update.AuthType != "" {
			existing.AuthType = update.AuthType
		}
		existing.Username = strings.TrimSpace(update.Username)
		existing.ProjectKey = strings.TrimSpace(update.ProjectKey)
		existing.DefaultType = strings.TrimSpace(update.DefaultType)
		existing.Enabled = update.Enabled

		secret := strings.TrimSpace(update.TokenSecret)
		if secret != "" && secret != "••••••••" {
			existing.TokenSecret = secret
		}

		if err := s.integStorage.Save(existing); err != nil {
			respondError(w, http.StatusInternalServerError, "Erro ao atualizar integração: "+err.Error())
			return
		}
		respondJSON(w, http.StatusOK, existing.ToSafe())

	case http.MethodDelete:
		if err := s.integStorage.Delete(id); err != nil {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"message": "Portal de chamados removido com sucesso", "id": id})

	default:
		respondError(w, http.StatusMethodNotAllowed, "Método não permitido")
	}
}

func (s *Server) handleTestIntegration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Método não permitido")
		return
	}

	var req domain.TicketingIntegration
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Payload inválido: "+err.Error())
		return
	}

	// Se veio um ID existente e a senha está em branco ou mascarada, resgata o segredo salvo
	if req.ID != "" && (req.TokenSecret == "" || req.TokenSecret == "••••••••") {
		if saved, err := s.integStorage.Get(req.ID); err == nil {
			req.TokenSecret = saved.TokenSecret
			if req.Username == "" {
				req.Username = saved.Username
			}
			if req.BaseURL == "" {
				req.BaseURL = saved.BaseURL
			}
			if req.Type == "" {
				req.Type = saved.Type
			}
			if req.AuthType == "" {
				req.AuthType = saved.AuthType
			}
		}
	}

	result := actions.TestIntegration(r.Context(), &req)
	respondJSON(w, http.StatusOK, result)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
