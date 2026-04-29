package main

import (
	"testing"
	"time"
)

func TestDetectLevel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"2026/04/24 INF something happened", "INF"},
		{"2026/04/24 INFO something happened", "INF"},
		{"2026/04/24 WAR drift detected", "WRN"},
		{"2026/04/24 WARN drift detected", "WRN"},
		{"2026/04/24 WRN drift detected", "WRN"},
		{"2026/04/24 ERR fatal kaboom", "ERR"},
		{"2026/04/24 ERROR fatal kaboom", "ERR"},
		{"2026/04/24 FTL hard stop", "ERR"},
		{"2026/04/24 DBG verbose", "DBG"},
		{"2026/04/24 DEBUG verbose", "DBG"},
		{"2026/04/24 TRACE deepest", "DBG"},
		{"line with no level token", "INF"},
		{"", "INF"},
	}
	for _, c := range cases {
		if got := detectLevel(c.in); got != c.want {
			t.Errorf("detectLevel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLogHubHistoryCappedAtMax(t *testing.T) {
	h := NewLogHub()
	for i := 0; i < historyMax+50; i++ {
		h.Publish(LogLine{Source: "test", Time: time.Now(), Level: "INF", Text: "x"})
	}
	hist := h.History("test")
	if len(hist) != historyMax {
		t.Errorf("history len = %d, want %d (cap)", len(hist), historyMax)
	}
}

func TestLogHubHistoryPerSource(t *testing.T) {
	h := NewLogHub()
	h.Publish(LogLine{Source: "a", Text: "1"})
	h.Publish(LogLine{Source: "b", Text: "2"})
	h.Publish(LogLine{Source: "a", Text: "3"})
	if got := len(h.History("a")); got != 2 {
		t.Errorf("source 'a' history = %d, want 2", got)
	}
	if got := len(h.History("b")); got != 1 {
		t.Errorf("source 'b' history = %d, want 1", got)
	}
}

func TestLogHubAllHistory(t *testing.T) {
	h := NewLogHub()
	h.Publish(LogLine{Source: "a", Text: "1"})
	h.Publish(LogLine{Source: "b", Text: "2"})
	all := h.AllHistory()
	if len(all) != 2 {
		t.Errorf("AllHistory = %d, want 2", len(all))
	}
}

func TestLogHubSubscribeDelivers(t *testing.T) {
	h := NewLogHub()
	ch, cancel := h.Subscribe()
	defer cancel()

	go h.Publish(LogLine{Source: "test", Text: "first"})
	select {
	case got := <-ch:
		if got.Text != "first" {
			t.Errorf("channel got %q, want 'first'", got.Text)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("did not receive published line within 200ms")
	}
}

func TestLogHubSubscribeCancelClosesChannel(t *testing.T) {
	h := NewLogHub()
	ch, cancel := h.Subscribe()
	cancel()

	// channel should be closed; subsequent receive should not block
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed after cancel")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("receive on cancelled channel timed out instead of returning closed")
	}
}
