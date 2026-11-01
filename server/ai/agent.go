package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"ssg/model"
)

// AgentStatus is the lifecycle state of an agent as reported to the UI.
type AgentStatus string

const (
	StatusStarting AgentStatus = "starting"
	StatusWaiting  AgentStatus = "waiting"
	StatusReady    AgentStatus = "ready"
	StatusError    AgentStatus = "error"
	StatusStopped  AgentStatus = "stopped"
	StatusPaused   AgentStatus = "paused"
)

// DefaultTickDelay is the minimum wait between an agent detecting it should
// act and actually submitting a turn. It throttles fast-advancing games so
// the UI stays responsive and gives the operator time to inspect or pause.
const DefaultTickDelay = 500 * time.Millisecond

// AgentSummary is the lightweight per-agent record returned by Manager.List.
type AgentSummary struct {
	ID           string      `json:"id"`
	Server       string      `json:"server"`
	GameID       string      `json:"game_id"`
	PlayerID     uint        `json:"player_id"`
	PolicyName   string      `json:"policy"`
	Status       AgentStatus `json:"status"`
	CurrentTurn  float64     `json:"current_turn"`
	OwnedPlanets int         `json:"owned_planets"`
	OwnedFleets  int         `json:"owned_fleets"`
	LastError    string      `json:"last_error,omitempty"`
	Paused       bool        `json:"paused"`
	TickDelayMs  int64       `json:"tick_delay_ms"`
}

type Agent struct {
	ID      string
	Client  *Client
	Policy  Policy
	Log     *Log
	Manager *Manager

	mu        sync.RWMutex
	snapshot  *VisibleGame
	status    AgentStatus
	lastErr   string
	lastTurn  float64
	cancel    context.CancelFunc
	ctx       context.Context
	stopped   chan struct{}
	startedAt time.Time
	tickDelay time.Duration
	paused    bool
}

func newAgent(id string, client *Client, policy Policy, m *Manager, tickDelay time.Duration) *Agent {
	if tickDelay < 0 {
		tickDelay = 0
	}
	return &Agent{
		ID:        id,
		Client:    client,
		Policy:    policy,
		Log:       NewLog(500),
		Manager:   m,
		status:    StatusStarting,
		stopped:   make(chan struct{}),
		startedAt: time.Now().UTC(),
		tickDelay: tickDelay,
	}
}

func (a *Agent) Summary() AgentSummary {
	a.mu.RLock()
	defer a.mu.RUnlock()
	sum := AgentSummary{
		ID:          a.ID,
		Server:      a.Client.BaseURL,
		GameID:      a.Client.GameID,
		PlayerID:    a.Client.PlayerID,
		PolicyName:  a.Policy.Name(),
		Status:      a.status,
		CurrentTurn: a.lastTurn,
		LastError:   a.lastErr,
		Paused:      a.paused,
		TickDelayMs: a.tickDelay.Milliseconds(),
	}
	if a.paused {
		sum.Status = StatusPaused
	}
	if a.snapshot != nil {
		for _, p := range myPlanets(a.snapshot, a.Client.PlayerID) {
			_ = p
			sum.OwnedPlanets++
		}
		for _, f := range myFleets(a.snapshot, a.Client.PlayerID) {
			_ = f
			sum.OwnedFleets++
		}
	}
	return sum
}

// Snapshot returns the latest VisibleGame the agent has seen, or nil.
func (a *Agent) Snapshot() *VisibleGame {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.snapshot
}

func (a *Agent) IsPaused() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.paused
}

func (a *Agent) tickDelayOrDefault() time.Duration {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.tickDelay
}

// SetPaused toggles whether the agent will submit turns. While paused it
// still consumes updates and keeps its snapshot current; it just refuses to
// submit. Unpausing triggers an immediate evaluation against the latest
// snapshot so the agent doesn't idle for a full WS update cycle.
func (a *Agent) SetPaused(paused bool) {
	a.mu.Lock()
	if a.paused == paused {
		a.mu.Unlock()
		return
	}
	a.paused = paused
	a.mu.Unlock()
	if paused {
		a.Log.Append(LogEntry{Kind: EntryInfo, Message: "paused"})
	} else {
		a.Log.Append(LogEntry{Kind: EntryInfo, Message: "resumed"})
	}
	a.Manager.notifyAgentChanged(a.ID)
	if !paused {
		// Re-evaluate against the latest snapshot now that we're live again.
		if vg := a.Snapshot(); vg != nil {
			go a.maybeAct(a.runCtx(), vg)
		}
	}
}

func (a *Agent) runCtx() context.Context {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

func (a *Agent) setStatus(s AgentStatus) {
	a.mu.Lock()
	a.status = s
	a.mu.Unlock()
	a.Manager.notifyAgentChanged(a.ID)
}

func (a *Agent) setError(err error) {
	a.mu.Lock()
	a.status = StatusError
	if err != nil {
		a.lastErr = err.Error()
	}
	a.mu.Unlock()
	a.Manager.notifyAgentChanged(a.ID)
}

func (a *Agent) recordSnapshot(vg *VisibleGame) {
	a.mu.Lock()
	a.snapshot = vg
	a.lastTurn = vg.CurrentTurn
	a.lastErr = ""
	if slices.Contains(vg.TurnStatus.WaitingPlayerIDs, a.Client.PlayerID) {
		a.status = StatusWaiting
	} else if slices.Contains(vg.TurnStatus.ReadyPlayerIDs, a.Client.PlayerID) {
		a.status = StatusReady
	}
	a.mu.Unlock()
	a.Manager.notifyAgentChanged(a.ID)
}

// Start runs the agent's main loop until Stop is called.
func (a *Agent) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	a.mu.Lock()
	a.cancel = cancel
	a.ctx = ctx
	a.mu.Unlock()
	go a.run(ctx)
}

func (a *Agent) Stop() {
	a.mu.Lock()
	c := a.cancel
	a.mu.Unlock()
	if c != nil {
		c()
	}
	<-a.stopped
}

func (a *Agent) run(ctx context.Context) {
	defer close(a.stopped)

	a.Log.Append(LogEntry{Kind: EntryInfo, Message: fmt.Sprintf("agent started against %s game=%s player=%d", a.Client.BaseURL, a.Client.GameID, a.Client.PlayerID)})

	initial, err := a.Client.FetchGame(ctx)
	if err != nil {
		a.Log.Append(LogEntry{Kind: EntryError, Message: "initial game fetch failed", Error: err.Error()})
		a.setError(err)
		return
	}
	a.recordSnapshot(initial)
	a.maybeAct(ctx, initial)

	for ctx.Err() == nil {
		updates, err := a.Client.Subscribe(ctx)
		if err != nil {
			a.Log.Append(LogEntry{Kind: EntryError, Message: "websocket subscribe failed", Error: err.Error()})
			a.setError(err)
			if !sleepCtx(ctx, 2*time.Second) {
				return
			}
			continue
		}
		a.consume(ctx, updates)
		if ctx.Err() != nil {
			break
		}
		a.Log.Append(LogEntry{Kind: EntryInfo, Message: "websocket dropped; reconnecting"})
		if !sleepCtx(ctx, time.Second) {
			return
		}
	}

	a.setStatus(StatusStopped)
}

func (a *Agent) consume(ctx context.Context, updates <-chan *VisibleGame) {
	for {
		select {
		case <-ctx.Done():
			return
		case vg, ok := <-updates:
			if !ok {
				return
			}
			a.recordSnapshot(vg)
			a.maybeAct(ctx, vg)
		}
	}
}

const maxSubmitRetries = 5

// TurnSubmitter is the submit surface used by maybeAct so tests can stub
// rejection/retry without a live server.
type TurnSubmitter interface {
	SubmitTurn(ctx context.Context, commands model.Commands) (*TurnResponse, error)
}

// maybeAct decides and submits if the player is in the waiting set. The check
// is idempotent: once we've submitted for this turn the server moves us to
// ready_player_ids and we'll early-return on subsequent updates.
//
// If the server rejects the submission (e.g. a fleet move outside fuel range
// when submit-time validation is on), the agent logs the reason and retries
// without the offending fleet moves, bounded by maxSubmitRetries.
func (a *Agent) maybeAct(ctx context.Context, vg *VisibleGame) {
	a.maybeActWithSubmitter(ctx, vg, a.Client)
}

func (a *Agent) maybeActWithSubmitter(ctx context.Context, vg *VisibleGame, submitter TurnSubmitter) {
	if !slices.Contains(vg.TurnStatus.WaitingPlayerIDs, a.Client.PlayerID) {
		return
	}
	// Wait so fast turn cycles don't spam the server and UI. Re-check pause
	// and ctx after the sleep — the user may have cancelled or paused during it.
	if delay := a.tickDelayOrDefault(); delay > 0 {
		if !sleepCtx(ctx, delay) {
			return
		}
	}
	if a.IsPaused() {
		return
	}
	decision := a.Policy.Decide(vg, a.Client.PlayerID)
	for _, note := range decision.Notes {
		a.Log.Append(LogEntry{Kind: EntryDecision, Turn: vg.CurrentTurn, Message: note})
	}
	a.submitDecisionWithRetries(ctx, vg, submitter, decision)
}

func (a *Agent) submitDecisionWithRetries(ctx context.Context, vg *VisibleGame, submitter TurnSubmitter, decision Decision) {
	commands := decision.Commands
	for attempt := 0; attempt < maxSubmitRetries; attempt++ {
		reqJSON, _ := json.Marshal(commands)
		resp, err := submitter.SubmitTurn(ctx, commands)
		entry := LogEntry{Kind: EntryCommand, Turn: vg.CurrentTurn, Request: reqJSON}
		if err == nil {
			entry.Response = resp.Raw
			entry.Advanced = resp.Advanced
			a.Log.Append(entry)
			return
		}
		entry.Error = err.Error()
		a.Log.Append(entry)
		a.Log.Append(LogEntry{
			Kind:    EntryError,
			Turn:    vg.CurrentTurn,
			Message: fmt.Sprintf("submit rejected (attempt %d/%d): %s", attempt+1, maxSubmitRetries, err.Error()),
			Error:   err.Error(),
		})

		next, stripped := stripRejectedFleetMoves(commands, err)
		if !stripped {
			a.setError(err)
			return
		}
		a.Log.Append(LogEntry{
			Kind:    EntryInfo,
			Turn:    vg.CurrentTurn,
			Message: "retrying turn without rejected fleet move(s)",
		})
		commands = next
	}
	a.setError(fmt.Errorf("submit failed after %d retries", maxSubmitRetries))
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// stripRejectedFleetMoves drops fleet-move commands when the server rejected
// the turn for fuel range (or a similar fleet-move validation error). Returns
// the adjusted commands and whether anything was stripped. If there are no
// fleet moves left to drop, stripped is false so the caller stops retrying.
func stripRejectedFleetMoves(commands model.Commands, err error) (model.Commands, bool) {
	if err == nil || len(commands.FleetMove) == 0 {
		return commands, false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "fuel range") && !strings.Contains(msg, "fleet move") && !strings.Contains(msg, "interstellar drive") && !strings.Contains(msg, "drive speed for ship") {
		return commands, false
	}
	// Server errors currently omit the fleet id; drop all scout moves for this
	// retry so a single bad destination cannot wedge the agent forever.
	commands.FleetMove = nil
	return commands, true
}
