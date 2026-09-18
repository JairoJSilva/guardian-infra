package actions

import (
	"sync"

	"guardian/internal/domain"
)

type Notifier struct {
	mu          sync.RWMutex
	subscribers map[chan *domain.IncidentEvent]struct{}
	history     []*domain.IncidentEvent
	maxHistory  int
}

func NewNotifier(maxHistory int) *Notifier {
	if maxHistory <= 0 {
		maxHistory = 50
	}
	return &Notifier{
		subscribers: make(map[chan *domain.IncidentEvent]struct{}),
		history:     make([]*domain.IncidentEvent, 0, maxHistory),
		maxHistory:  maxHistory,
	}
}

func (n *Notifier) Subscribe() chan *domain.IncidentEvent {
	n.mu.Lock()
	defer n.mu.Unlock()

	ch := make(chan *domain.IncidentEvent, 100)
	n.subscribers[ch] = struct{}{}
	return ch
}

func (n *Notifier) Unsubscribe(ch chan *domain.IncidentEvent) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if _, ok := n.subscribers[ch]; ok {
		delete(n.subscribers, ch)
		close(ch)
	}
}

func (n *Notifier) Broadcast(event *domain.IncidentEvent) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Adiciona ao histórico circular
	if len(n.history) >= n.maxHistory {
		n.history = n.history[1:]
	}
	n.history = append(n.history, event)

	// Envia para todos os canais de forma não-bloqueante
	for ch := range n.subscribers {
		select {
		case ch <- event:
		default:
			// Buffer cheio, descarta para evitar travar o loop
		}
	}
}

func (n *Notifier) GetHistory() []*domain.IncidentEvent {
	n.mu.RLock()
	defer n.mu.RUnlock()

	out := make([]*domain.IncidentEvent, len(n.history))
	copy(out, n.history)
	return out
}
