package main

import (
	"context"
	"regexp"
	"sort"
	"sync"
	"time"
)

// Stream-health monitor: subscribes to the log hub, pattern-matches against
// a list of known mediamtx / cloudflared signals, and surfaces them with
// plain-English fix instructions. Noise filters short-circuit chatter that
// is normal for LL-HLS (e.g. cloudflared "context canceled" preload churn).

type Incident struct {
	Key         string    `json:"key"`
	Severity    string    `json:"severity"` // "info" | "warn" | "err"
	Category    string    `json:"category"`
	Title       string    `json:"title"`
	Detail      string    `json:"detail"`
	Remediation string    `json:"remediation"`
	Count       int       `json:"count"`
	FirstSeen   time.Time `json:"firstSeen"`
	LastSeen    time.Time `json:"lastSeen"`
	Sample      string    `json:"sample"`
}

type detector struct {
	key         string
	source      string
	severity    string
	category    string
	title       string
	remediation string
	pattern     *regexp.Regexp
	silent      bool
	build       func([]string) string
}

// Order matters: silent/specific detectors first, generic catchers last.
var detectors = []detector{
	// ── Noise filters (silent — match but never surface) ─────────────
	{
		key:     "noise-llhls-cancel",
		source:  "cloudflared",
		pattern: regexp.MustCompile(`Incoming request ended abruptly: context canceled`),
		silent:  true,
	},

	// ── MediaMTX ─────────────────────────────────────────────────────
	{
		key:      "hls-part-drift",
		source:   "mediamtx",
		severity: "warn",
		category: "iOS / HLS",
		title:    "HLS part duration drifting",
		pattern:  regexp.MustCompile(`part duration changed from (\d+)ms to (\d+)ms`),
		remediation: "Pin OBS keyframe interval to a fixed 1s: OBS → Settings → Output → Streaming → Keyframe Interval = 1. Apple's HLS validator rejects parts of varying length, so iOS / Safari / Apple TV viewers will rebuffer or fail until OBS is fixed. Restart the broadcast after changing.",
		build: func(m []string) string {
			return "Mediamtx targets ~100ms parts but emitted " + m[1] + "ms then " + m[2] + "ms — Apple-spec violation."
		},
	},
	{
		key:      "srt-publisher-disconnect",
		source:   "mediamtx",
		severity: "warn",
		category: "Ingest",
		title:    "SRT publisher disconnected",
		pattern:  regexp.MustCompile(`\[SRT\].*closed|publisher.*closed`),
		remediation: "OBS stopped sending. Common causes: hit Stop in OBS, network drop, sleep mode, or the laptop went idle. Click Start Streaming in OBS to reconnect.",
	},
	{
		key:      "srt-publisher-connected",
		source:   "mediamtx",
		severity: "info",
		category: "Ingest",
		title:    "SRT publisher connected",
		pattern:  regexp.MustCompile(`is publishing to path|opened by SRT publisher`),
		remediation: "A publisher is sending. If unexpected, verify who has your stream key.",
	},
	{
		key:      "muxer-create-failed",
		source:   "mediamtx",
		severity: "err",
		category: "HLS",
		title:    "HLS muxer failed to start",
		pattern:  regexp.MustCompile(`(?i)muxer.*(failed|error|cannot)`),
		remediation: "The HLS muxer couldn't be created. Most often this means the publisher is sending an unsupported codec. Verify OBS is set to H.264 video + AAC audio (not Opus, not HEVC unless you've enabled it).",
	},
	{
		key:      "mediamtx-error",
		source:   "mediamtx",
		severity: "err",
		category: "Mediamtx",
		title:    "Mediamtx error",
		pattern:  regexp.MustCompile(`\bERR\b`),
		remediation: "Open the live logs panel below and read the message. Most ERRs are transient (a single dropped reader). If the same error repeats, OBS or the network usually needs attention.",
	},

	// ── Cloudflared ──────────────────────────────────────────────────
	{
		key:      "cf-tunnel-disconnect",
		source:   "cloudflared",
		severity: "err",
		category: "Tunnel",
		title:    "Cloudflare tunnel lost connection",
		pattern:  regexp.MustCompile(`(?i)unregistered tunnel|failed to connect to the edge|connection terminated|edge connection error`),
		remediation: "Check internet connectivity. The named tunnel will retry automatically. If failures continue >2 min, check the launchd agent: launchctl list | grep cloudflared",
	},
	{
		key:      "cf-tunnel-registered",
		source:   "cloudflared",
		severity: "info",
		category: "Tunnel",
		title:    "Cloudflare tunnel connected",
		pattern:  regexp.MustCompile(`Registered tunnel connection`),
		remediation: "Tunnel established (or re-established) a connection to a Cloudflare edge POP. Normal at startup or after network blips.",
	},
	{
		key:      "cf-origin-error",
		source:   "cloudflared",
		severity: "warn",
		category: "Tunnel",
		title:    "Cloudflare → origin error",
		pattern:  regexp.MustCompile(`(?i)dial tcp.*connect: connection refused|origin.*unreachable`),
		remediation: "Cloudflare can reach the public internet but cannot reach mediamtx on this Mac. Make sure mediamtx is running on the configured port (default :8888 for HLS).",
	},
}

type HealthMonitor struct {
	mu        sync.Mutex
	incidents map[string]*Incident
}

func NewHealthMonitor() *HealthMonitor {
	return &HealthMonitor{incidents: map[string]*Incident{}}
}

func (h *HealthMonitor) Start(ctx context.Context, hub *LogHub) {
	// Process recent history first so the panel populates immediately
	// from whatever's already in the log buffer at startup.
	for _, line := range hub.AllHistory() {
		h.process(line)
	}
	ch, cancel := hub.Subscribe()
	go func() {
		<-ctx.Done()
		cancel()
	}()
	go func() {
		for line := range ch {
			h.process(line)
		}
	}()
	go func() {
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				h.prune(15 * time.Minute)
			}
		}
	}()
}

func (h *HealthMonitor) process(line LogLine) {
	for _, d := range detectors {
		if d.source != "" && d.source != line.Source {
			continue
		}
		matches := d.pattern.FindStringSubmatch(line.Text)
		if matches == nil {
			continue
		}
		if d.silent {
			return
		}
		h.mu.Lock()
		inc, ok := h.incidents[d.key]
		if !ok {
			inc = &Incident{
				Key:         d.key,
				Severity:    d.severity,
				Category:    d.category,
				Title:       d.title,
				Remediation: d.remediation,
				FirstSeen:   line.Time,
			}
			h.incidents[d.key] = inc
		}
		if d.build != nil {
			inc.Detail = d.build(matches)
		} else {
			detail := line.Text
			if len(detail) > 220 {
				detail = detail[:220] + "…"
			}
			inc.Detail = detail
		}
		inc.Count++
		inc.LastSeen = line.Time
		inc.Sample = line.Text
		h.mu.Unlock()
		return
	}
}

func (h *HealthMonitor) prune(maxAge time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	cut := time.Now().Add(-maxAge)
	for k, v := range h.incidents {
		if v.LastSeen.Before(cut) {
			delete(h.incidents, k)
		}
	}
}

func (h *HealthMonitor) Snapshot() []*Incident {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]*Incident, 0, len(h.incidents))
	for _, v := range h.incidents {
		c := *v
		out = append(out, &c)
	}
	rank := func(s string) int {
		switch s {
		case "err":
			return 0
		case "warn":
			return 1
		default:
			return 2
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if rank(out[i].Severity) != rank(out[j].Severity) {
			return rank(out[i].Severity) < rank(out[j].Severity)
		}
		return out[i].LastSeen.After(out[j].LastSeen)
	})
	return out
}

func (h *HealthMonitor) Overall() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	hasWarn := false
	for _, v := range h.incidents {
		if v.Severity == "err" {
			return "err"
		}
		if v.Severity == "warn" {
			hasWarn = true
		}
	}
	if hasWarn {
		return "warn"
	}
	return "ok"
}
