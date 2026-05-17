package ai

import "testing"

func TestLogRingBufferWrapsCleanly(t *testing.T) {
	l := NewLog(3)
	for i := 1; i <= 5; i++ {
		l.Append(LogEntry{Message: string(rune('0' + i))})
	}
	got := l.Snapshot()
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	want := []string{"3", "4", "5"}
	for i, e := range got {
		if e.Message != want[i] {
			t.Errorf("entry %d: got %q want %q", i, e.Message, want[i])
		}
	}
	// Seq must be monotonic and reflect the original insertion order, not the slot index.
	if got[0].Seq != 3 || got[2].Seq != 5 {
		t.Errorf("seq numbers wrong: %d ... %d", got[0].Seq, got[2].Seq)
	}
}

func TestLogSinkFiresOnAppend(t *testing.T) {
	l := NewLog(4)
	var received []string
	l.SetSink(func(e LogEntry) { received = append(received, e.Message) })
	l.Append(LogEntry{Message: "a"})
	l.Append(LogEntry{Message: "b"})
	if len(received) != 2 || received[0] != "a" || received[1] != "b" {
		t.Fatalf("sink got %v", received)
	}
}
