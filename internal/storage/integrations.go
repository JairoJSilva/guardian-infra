package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"guardian/internal/config"
	"guardian/internal/domain"
)

type IntegrationStorage struct {
	mu           sync.RWMutex
	filePath     string
	integrations map[string]*domain.TicketingIntegration
}

func NewIntegrationStorage(filePath string, cfg *config.Config) (*IntegrationStorage, error) {
	s := &IntegrationStorage{
		filePath:     filePath,
		integrations: make(map[string]*domain.TicketingIntegration),
	}

	if err := s.load(); err != nil || len(s.integrations) == 0 {
		s.seedDefaults(cfg)
		_ = s.save()
	}

	return s, nil
}

func (s *IntegrationStorage) seedDefaults(cfg *config.Config) {
	if cfg == nil {
		return
	}
	baseURL := cfg.JiraBaseURL
	if baseURL == "" {
		baseURL = "https://jira.atlassian.net"
	}
	proj := cfg.JiraProjectKey
	if proj == "" {
		proj = "OPS"
	}
	issueType := cfg.JiraIssueType
	if issueType == "" {
		issueType = "Bug"
	}

	defaultJira := &domain.TicketingIntegration{
		ID:          "jira-default",
		Name:        "Jira Corporativo (Padrão)",
		Type:        domain.ProviderJira,
		BaseURL:     baseURL,
		AuthType:    "BASIC",
		Username:    cfg.JiraUser,
		TokenSecret: cfg.JiraPassword,
		ProjectKey:  proj,
		DefaultType: issueType,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	s.integrations[defaultJira.ID] = defaultJira
}

func (s *IntegrationStorage) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var list []*domain.TicketingIntegration
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	s.integrations = make(map[string]*domain.TicketingIntegration)
	for _, it := range list {
		s.integrations[it.ID] = it
	}
	return nil
}

func (s *IntegrationStorage) save() error {
	list := make([]*domain.TicketingIntegration, 0, len(s.integrations))
	for _, it := range s.integrations {
		list = append(list, it)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

func (s *IntegrationStorage) List() []*domain.TicketingIntegration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*domain.TicketingIntegration, 0, len(s.integrations))
	for _, it := range s.integrations {
		copyIt := *it
		out = append(out, &copyIt)
	}
	return out
}

func (s *IntegrationStorage) ListSafe() []*domain.TicketingIntegrationSafe {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*domain.TicketingIntegrationSafe, 0, len(s.integrations))
	for _, it := range s.integrations {
		out = append(out, it.ToSafe())
	}
	return out
}

func (s *IntegrationStorage) Get(id string) (*domain.TicketingIntegration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	it, ok := s.integrations[id]
	if !ok {
		return nil, fmt.Errorf("integração com ID %s não encontrada", id)
	}
	copyIt := *it
	return &copyIt, nil
}

func (s *IntegrationStorage) GetDefault() *domain.TicketingIntegration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, it := range s.integrations {
		if it.Enabled {
			copyIt := *it
			return &copyIt
		}
	}
	for _, it := range s.integrations {
		copyIt := *it
		return &copyIt
	}
	return nil
}

func (s *IntegrationStorage) Save(it *domain.TicketingIntegration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(it.ID) == "" {
		it.ID = fmt.Sprintf("portal-%s-%d", strings.ToLower(string(it.Type)), time.Now().UnixNano()%1000000)
	}
	if it.CreatedAt.IsZero() {
		it.CreatedAt = time.Now()
	}
	it.UpdatedAt = time.Now()

	s.integrations[it.ID] = it
	return s.save()
}

func (s *IntegrationStorage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.integrations[id]; !ok {
		return errors.New("integração não encontrada")
	}
	delete(s.integrations, id)
	return s.save()
}

func (s *IntegrationStorage) ToggleEnabled(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	it, ok := s.integrations[id]
	if !ok {
		return false, errors.New("integração não encontrada")
	}
	it.Enabled = !it.Enabled
	it.UpdatedAt = time.Now()
	err := s.save()
	return it.Enabled, err
}
