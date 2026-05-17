package ai

import (
	"encoding/json"
	"sync"
	"time"
)

// EntryKind distinguishes log entries the UI renders differently.
type EntryKind string

const (
	EntryDecision EntryKind = "decision"
	EntryCommand  EntryKind = "command"
	EntryError    EntryKind = "error"
	EntryInfo     EntryKind = "info"
)

// LogEntry is the single record type in an agent's ring buffer. Decision and
// command entries fill different fields; the UI keys off Kind.
type LogEntry struct {
	Seq       uint64          `json:"seq"`
	Timestamp time.Time       `json:"timestamp"`
	Kind      EntryKind       `json:"kind"`
	Turn      float64         `json:"turn"`
	Message   string          `json:"message,omitempty"`
	Request   json.RawMessage `json:"request,omitempty"`
	Response  json.RawMessage `json:"response,omitempty"`
	Advanced  bool            `json:"advanced,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// Log is a fixed-capacity ring buffer. Older entries get overwritten silently
// once the buffer fills. Reads return a chronological snapshot.
type Log struct {
	mu      sync.Mutex
	entries []LogEntry
	head    int
	size    int
	seq     uint64
	onAppend func(LogEntry)
}

func NewLog(capacity int) *Log {
	if capacity <= 0 {
		capacity = 200
	}
	return &Log{entries: make([]LogEntry, capacity)}
}

// SetSink registers a callback fired (synchronously, under the log's lock)
// each time an entry is appended. The manager uses this to fan entries out
// to UI subscribers.
func (l *Log) SetSink(fn func(LogEntry)) {
	l.mu.Lock()
	l.onAppend = fn
	l.mu.Unlock()
}

func (l *Log) Append(e LogEntry) LogEntry {
	l.mu.Lock()
	l.seq++
	e.Seq = l.seq
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	idx := (l.head + l.size) % cap(l.entries)
	if l.size < cap(l.entries) {
		l.size++
	} else {
		l.head = (l.head + 1) % cap(l.entries)
	}
	l.entries[idx] = e
	sink := l.onAppend
	l.mu.Unlock()
	if sink != nil {
		sink(e)
	}
	return e
}

// Snapshot returns the current entries in chronological order.
func (l *Log) Snapshot() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]LogEntry, l.size)
	for i := 0; i < l.size; i++ {
		out[i] = l.entries[(l.head+i)%cap(l.entries)]
	}
	return out
}
