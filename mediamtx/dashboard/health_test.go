package main

import (
	"strings"
	"testing"
	"time"
)

func TestHealthDetectorPartDrift(t *testing.T) {
	h := NewHealthMonitor()
	h.process(LogLine{
		Source: "mediamtx",
		Time:   time.Now(),
		Text:   "2026/04/24 16:39:03 WAR [HLS] [muxer program] part duration changed from 140ms to 134ms - this will cause an error in iOS clients",
	})
	snap := h.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(snap))
	}
	inc := snap[0]
	if inc.Key != "hls-part-drift" {
		t.Errorf("Key = %q, want hls-part-drift", inc.Key)
	}
	if inc.Severity != "warn" {
		t.Errorf("Severity = %q, want warn", inc.Severity)
	}
	if !strings.Contains(inc.Detail, "140") || !strings.Contains(inc.Detail, "134") {
		t.Errorf("Detail %q should include captured 140/134", inc.Detail)
	}
	if inc.Remediation == "" {
		t.Error("Remediation should not be empty for known incident")
	}
}

func TestHealthDetectorNoiseFilterSilent(t *testing.T) {
	h := NewHealthMonitor()
	h.process(LogLine{
		Source: "cloudflared",
		Time:   time.Now(),
		Text:   `ERR error="Incoming request ended abruptly: context canceled" connIndex=2`,
	})
	if len(h.Snapshot()) != 0 {
		t.Errorf("LL-HLS preload-cancel noise should be silent, got %d incidents", len(h.Snapshot()))
	}
}

func TestHealthDetectorWrongSourceIgnored(t *testing.T) {
	h := NewHealthMonitor()
	// part-drift detector is scoped to source=mediamtx; same text on cloudflared shouldn't match
	h.process(LogLine{
		Source: "cloudflared",
		Time:   time.Now(),
		Text:   "part duration changed from 140ms to 134ms",
	})
	if len(h.Snapshot()) != 0 {
		t.Errorf("part-drift on cloudflared source should not match, got %d", len(h.Snapshot()))
	}
}

func TestHealthDetectorSrtPublisherEvents(t *testing.T) {
	h := NewHealthMonitor()
	h.process(LogLine{
		Source: "mediamtx",
		Time:   time.Now(),
		Text:   "2026/04/28 17:18:52 INF [SRT] [conn 10.130.80.183:62368] is publishing to path 'actor1'",
	})
	h.process(LogLine{
		Source: "mediamtx",
		Time:   time.Now(),
		Text:   "2026/04/28 17:18:59 INF [SRT] [conn 10.130.80.183:62368] closed: EOF",
	})
	snap := h.Snapshot()
	keys := map[string]bool{}
	for _, inc := range snap {
		keys[inc.Key] = true
	}
	if !keys["srt-publisher-connected"] {
		t.Error("expected srt-publisher-connected incident")
	}
	if !keys["srt-publisher-disconnect"] {
		t.Error("expected srt-publisher-disconnect incident")
	}
}

func TestHealthIncidentCounting(t *testing.T) {
	h := NewHealthMonitor()
	for i := 0; i < 5; i++ {
		h.process(LogLine{
			Source: "mediamtx",
			Time:   time.Now(),
			Text:   "part duration changed from 100ms to 134ms",
		})
	}
	snap := h.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("repeated matches should collapse into 1 incident, got %d", len(snap))
	}
	if snap[0].Count != 5 {
		t.Errorf("Count = %d, want 5", snap[0].Count)
	}
	if snap[0].FirstSeen.After(snap[0].LastSeen) {
		t.Error("FirstSeen should not be after LastSeen")
	}
}

func TestHealthOverallSeverityTransitions(t *testing.T) {
	h := NewHealthMonitor()
	if got := h.Overall(); got != "ok" {
		t.Errorf("empty Overall = %q, want ok", got)
	}
	// warn-class incident
	h.process(LogLine{Source: "mediamtx", Time: time.Now(), Text: "part duration changed from 140ms to 134ms"})
	if got := h.Overall(); got != "warn" {
		t.Errorf("after warn Overall = %q, want warn", got)
	}
	// err-class wins over warn
	h.process(LogLine{Source: "mediamtx", Time: time.Now(), Text: "2026/04/24 ERR something fatal happened"})
	if got := h.Overall(); got != "err" {
		t.Errorf("after err Overall = %q, want err", got)
	}
}

func TestHealthPruneAgesOutOldIncidents(t *testing.T) {
	h := NewHealthMonitor()
	h.process(LogLine{Source: "mediamtx", Time: time.Now(), Text: "ERR thing"})
	if len(h.Snapshot()) != 1 {
		t.Fatalf("setup: expected 1 incident")
	}
	// Backdate the LastSeen so the prune drops it
	old := time.Now().Add(-30 * time.Minute)
	h.mu.Lock()
	for _, inc := range h.incidents {
		inc.LastSeen = old
	}
	h.mu.Unlock()

	h.prune(15 * time.Minute)
	if len(h.Snapshot()) != 0 {
		t.Errorf("incident older than 15min should be pruned")
	}
}

func TestHealthSnapshotSortsErrorsFirst(t *testing.T) {
	h := NewHealthMonitor()
	// info incident
	h.process(LogLine{Source: "cloudflared", Time: time.Now(), Text: "Registered tunnel connection blah"})
	// warn incident (later)
	h.process(LogLine{Source: "mediamtx", Time: time.Now(), Text: "part duration changed from 100ms to 130ms"})
	// err incident (latest)
	h.process(LogLine{Source: "mediamtx", Time: time.Now(), Text: "ERR fatal kaboom"})

	snap := h.Snapshot()
	if len(snap) < 3 {
		t.Fatalf("expected 3+ incidents, got %d", len(snap))
	}
	if snap[0].Severity != "err" {
		t.Errorf("first incident should be err, got %s", snap[0].Severity)
	}
	// last should be info
	if snap[len(snap)-1].Severity != "info" {
		t.Errorf("last incident should be info, got %s", snap[len(snap)-1].Severity)
	}
}
