package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"guardian/internal/domain"
)

type Storage struct {
	mu       sync.RWMutex
	filePath string
	targets  map[string]*domain.Target
}

func NewStorage(filePath string) (*Storage, error) {
	s := &Storage{
		filePath: filePath,
		targets:  make(map[string]*domain.Target),
	}

	if err := s.load(); err != nil {
		// Se não existir, inicializa com os targets padrão da especificação
		s.seedDefaults()
		_ = s.save()
	}

	return s, nil
}

func (s *Storage) seedDefaults() {
	k8sTarget := &domain.Target{
		ID:       "target-k8s-prod",
		Name:     "Pagamentos & Checkout",
		Type:     domain.EnvKubernetes,
		Endpoint: "aks-prod-brazil",
		Scopes:   []string{"billing", "checkout"},
		Rules: domain.MonitoringRules{
			WatchCrashLoop:      true,
			WatchOOM:            true,
			WatchHealthCheck:    true,
			PollIntervalSeconds: 15,
		},
		Actions: domain.ActionConfig{
			CreateJiraIssue: true,
			JiraProjectKey:  "OPS",
			Contrato:        "INTERNO",
		},
		Status:    domain.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	dockerTarget := &domain.Target{
		ID:       "target-docker-local",
		Name:     "Bancos & Caches Locais",
		Type:     domain.EnvDocker,
		Endpoint: "docker-local",
		Scopes:   []string{"db-postgres", "redis-cache", "ecommerce-stack"},
		Rules: domain.MonitoringRules{
			WatchCrashLoop:      true,
			WatchOOM:            true,
			WatchHealthCheck:    true,
			PollIntervalSeconds: 10,
		},
		Actions: domain.ActionConfig{
			CreateJiraIssue: true,
			JiraProjectKey:  "OPS",
			Contrato:        "INTERNO",
		},
		Status:    domain.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.targets[k8sTarget.ID] = k8sTarget
	s.targets[dockerTarget.ID] = dockerTarget
}

func (s *Storage) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var list []*domain.Target
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	s.targets = make(map[string]*domain.Target)
	for _, t := range list {
		s.targets[t.ID] = t
	}
	return nil
}

func (s *Storage) save() error {
	list := make([]*domain.Target, 0, len(s.targets))
	for _, t := range s.targets {
		list = append(list, t)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

func (s *Storage) List() []*domain.Target {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*domain.Target, 0, len(s.targets))
	for _, t := range s.targets {
		// Retorna cópia rasa
		copyT := *t
		out = append(out, &copyT)
	}
	return out
}

func (s *Storage) Get(id string) (*domain.Target, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.targets[id]
	if !ok {
		return nil, fmt.Errorf("target com ID %s não encontrado", id)
	}
	copyT := *t
	return &copyT, nil
}

func (s *Storage) Save(t *domain.Target) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t.ID == "" {
		t.ID = fmt.Sprintf("target-%d", time.Now().UnixNano())
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	t.UpdatedAt = time.Now()

	s.targets[t.ID] = t
	return s.save()
}

func (s *Storage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.targets[id]; !ok {
		return errors.New("target não encontrado")
	}
	delete(s.targets, id)
	return s.save()
}

func (s *Storage) UpdateStatus(id string, status domain.TargetStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.targets[id]
	if !ok {
		return errors.New("target não encontrado")
	}
	t.Status = status
	t.UpdatedAt = time.Now()
	return s.save()
}
