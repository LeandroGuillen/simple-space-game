package ai

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager owns the live set of AI agents and a broadcast channel that UI
// subscribers consume for incremental updates.
type Manager struct {
	mu     sync.RWMutex
	agents map[string]*Agent
	ctx    context.Context
	cancel context.CancelFunc

	subsMu sync.RWMutex
	subs   map[chan Event]struct{}

	defaultTickDelay time.Duration
}

// Event is the wire shape pushed to UI subscribers.
type Event struct {
	Kind    string `json:"kind"` // "agent_changed" | "agent_log" | "agent_removed"
	AgentID string `json:"agent_id"`

	Summary *AgentSummary `json:"summary,omitempty"`
	Entry   *LogEntry     `json:"entry,omitempty"`
}

func NewManager(parent context.Context, defaultTickDelay time.Duration) *Manager {
	if defaultTickDelay < 0 {
		defaultTickDelay = 0
	}
	ctx, cancel := context.WithCancel(parent)
	return &Manager{
		agents:           map[string]*Agent{},
		ctx:              ctx,
		cancel:           cancel,
		subs:             map[chan Event]struct{}{},
		defaultTickDelay: defaultTickDelay,
	}
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	agents := make([]*Agent, 0, len(m.agents))
	for _, a := range m.agents {
		agents = append(agents, a)
	}
	m.mu.Unlock()
	for _, a := range agents {
		a.Stop()
	}
	m.cancel()
}

// Add creates and starts a new agent.
func (m *Manager) Add(serverURL, gameID string, playerID uint, policy Policy) (*Agent, error) {
	if serverURL == "" {
		return nil, errors.New("server URL is required")
	}
	if gameID == "" {
		return nil, errors.New("game id is required")
	}
	if playerID == 0 {
		return nil, errors.New("player id is required")
	}
	if policy == nil {
		policy = SimplePolicy{}
	}
	id := uuid.NewString()
	client := NewClient(serverURL, gameID, playerID)
	a := newAgent(id, client, policy, m, m.defaultTickDelay)
	a.Log.SetSink(func(e LogEntry) {
		m.broadcast(Event{Kind: "agent_log", AgentID: id, Entry: &e})
	})
	m.mu.Lock()
	m.agents[id] = a
	m.mu.Unlock()
	a.Start(m.ctx)
	sum := a.Summary()
	m.broadcast(Event{Kind: "agent_changed", AgentID: id, Summary: &sum})
	return a, nil
}

func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	a, ok := m.agents[id]
	if ok {
		delete(m.agents, id)
	}
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("agent %s not found", id)
	}
	a.Stop()
	m.broadcast(Event{Kind: "agent_removed", AgentID: id})
	return nil
}

func (m *Manager) Get(id string) (*Agent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.agents[id]
	return a, ok
}

func (m *Manager) List() []AgentSummary {
	m.mu.RLock()
	out := make([]AgentSummary, 0, len(m.agents))
	for _, a := range m.agents {
		out = append(out, a.Summary())
	}
	m.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Subscribe returns a buffered channel of events. The caller must call the
// returned cancel function to unsubscribe.
func (m *Manager) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, 64)
	m.subsMu.Lock()
	m.subs[ch] = struct{}{}
	m.subsMu.Unlock()
	return ch, func() {
		m.subsMu.Lock()
		if _, ok := m.subs[ch]; ok {
			delete(m.subs, ch)
			close(ch)
		}
		m.subsMu.Unlock()
	}
}

func (m *Manager) broadcast(e Event) {
	m.subsMu.RLock()
	defer m.subsMu.RUnlock()
	for ch := range m.subs {
		select {
		case ch <- e:
		default:
			// Drop on slow subscribers; UI will resync via the REST snapshot.
		}
	}
}

func (m *Manager) notifyAgentChanged(id string) {
	a, ok := m.Get(id)
	if !ok {
		return
	}
	sum := a.Summary()
	m.broadcast(Event{Kind: "agent_changed", AgentID: id, Summary: &sum})
}
