package supervisor

import (
	"context"
	"fmt"
	"log"
	"sync"

	"guardian/internal/actions"
	"guardian/internal/domain"
	"guardian/internal/providers/docker"
	"guardian/internal/providers/k8s"
	"guardian/internal/storage"
)

type Supervisor struct {
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	k8sPool      *k8s.ClientPool
	dockerClient *docker.DockerClient
	storage      *storage.Storage
	deduplicator *actions.Deduplicator
	jiraClient   *actions.JiraClient
	notifier     *actions.Notifier
	eventChan    chan *domain.IncidentEvent
	workers      map[string]*TargetWorker
}

func NewSupervisor(
	k8sPool *k8s.ClientPool,
	dockerClient *docker.DockerClient,
	store *storage.Storage,
	dedup *actions.Deduplicator,
	jira *actions.JiraClient,
	notif *actions.Notifier,
) *Supervisor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Supervisor{
		ctx:          ctx,
		cancel:       cancel,
		k8sPool:      k8sPool,
		dockerClient: dockerClient,
		storage:      store,
		deduplicator: dedup,
		jiraClient:   jira,
		notifier:     notif,
		eventChan:    make(chan *domain.IncidentEvent, 100),
		workers:      make(map[string]*TargetWorker),
	}
}

func (s *Supervisor) Start() {
	log.Println("[Supervisor] 🚀 Inicializando Dynamic Target Supervisor...")

	// Inicia rotina de consumo do pipeline de ações
	go s.eventLoop()

	// Inicia workers para todos os targets com status ACTIVE
	targets := s.storage.List()
	for _, t := range targets {
		if t.Status == domain.StatusActive {
			s.startWorker(t)
		}
	}
}

func (s *Supervisor) eventLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case event := <-s.eventChan:
			s.processEvent(event)
		}
	}
}

func (s *Supervisor) processEvent(event *domain.IncidentEvent) {
	fingerprint := event.Fingerprint()

	// Anti-Spam / Deduplicação
	if s.deduplicator.IsInCooldown(fingerprint) {
		log.Printf("[Supervisor] 🛡️ [Anti-Spam] Incidente %s suprimido por cooldown ativo.", fingerprint)
		return
	}

	target, _ := s.storage.Get(event.TargetID)

	// Abertura de chamado no Jira se configurado
	if target == nil || target.Actions.CreateJiraIssue {
		log.Printf("[Supervisor] 🎫 Abertura de chamado Jira disparada para incidente: %s", event.EntityName)
		key, err := s.jiraClient.CreateIncidentIssue(event, target)
		if err != nil {
			log.Printf("[Supervisor] ❌ Falha ao criar chamado no Jira: %v", err)
		} else {
			event.JiraIssue = key
			s.deduplicator.Record(fingerprint)
		}
	}

	// Transmissão para Live Feed em tempo real
	s.notifier.Broadcast(event)
}

func (s *Supervisor) startWorker(target *domain.Target) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.workers[target.ID]; exists {
		// Já em execução
		return
	}

	worker := NewTargetWorker(target, s.ctx, s.k8sPool, s.dockerClient, s.eventChan)
	s.workers[target.ID] = worker
	worker.Start()
	log.Printf("[Supervisor] [✓] Worker iniciado para Target: %s (%s)", target.Name, target.ID)
}

func (s *Supervisor) stopWorker(targetID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if worker, exists := s.workers[targetID]; exists {
		worker.Stop()
		delete(s.workers, targetID)
		log.Printf("[Supervisor] [✓] Worker parado para Target: %s", targetID)
	}
}

func (s *Supervisor) AddTarget(target *domain.Target) error {
	if err := s.storage.Save(target); err != nil {
		return err
	}
	if target.Status == domain.StatusActive {
		s.startWorker(target)
	}
	return nil
}

func (s *Supervisor) ToggleTargetStatus(targetID string) (*domain.Target, error) {
	target, err := s.storage.Get(targetID)
	if err != nil {
		return nil, err
	}

	if target.Status == domain.StatusActive {
		target.Status = domain.StatusPaused
		s.stopWorker(targetID)
	} else {
		target.Status = domain.StatusActive
		s.startWorker(target)
	}

	if err := s.storage.Save(target); err != nil {
		return nil, err
	}
	return target, nil
}

func (s *Supervisor) DeleteTarget(targetID string) error {
	s.stopWorker(targetID)
	return s.storage.Delete(targetID)
}

func (s *Supervisor) InjectSimulatedIncident(event *domain.IncidentEvent) {
	select {
	case s.eventChan <- event:
	default:
		go func() { s.eventChan <- event }()
	}
}

func (s *Supervisor) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, worker := range s.workers {
		worker.Stop()
		delete(s.workers, id)
	}
	s.cancel()
	fmt.Println("[Supervisor] Todos os workers foram finalizados com sucesso.")
}
