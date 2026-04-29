package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type Server struct {
	cfg    Config
	hub    *LogHub
	health *HealthMonitor
	http   *http.Server
}

func NewServer(cfg Config, hub *LogHub, health *HealthMonitor) *Server {
	s := &Server{cfg: cfg, hub: hub, health: health}
	mux := http.NewServeMux()

	sub, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(sub)))

	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/mediamtx/", s.handleMediaMTXProxy)
	mux.HandleFunc("/api/system", s.handleSystem)
	mux.HandleFunc("/api/network", s.handleNetwork)
	mux.HandleFunc("/api/cloudflare", s.handleCloudflare)
	mux.HandleFunc("/api/github", s.handleGitHub)
	mux.HandleFunc("/api/logs/stream", s.handleLogStream)
	mux.HandleFunc("/api/logs/history", s.handleLogHistory)
	mux.HandleFunc("/api/action/", s.handleAction)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/health/stream", s.handleStreamHealth)

	s.http = &http.Server{
		Addr:         cfg.Addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
	}
	return s
}

func (s *Server) ListenAndServe() error            { return s.http.ListenAndServe() }
func (s *Server) Shutdown(ctx context.Context) error { return s.http.Shutdown(ctx) }

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"publicHost":   s.cfg.PublicHost,
		"ghRepo":       s.cfg.GHRepo,
		"autoOpenPath": s.cfg.AutoOpenPath,
		"cfTokenSet":   s.cfg.CFAPIToken != "",
		"ports":        s.cfg.Ports,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	paths, pathsErr := FetchPaths(ctx, s.cfg.MediaMTXAPI)
	hls, _ := FetchHLSMuxers(ctx, s.cfg.MediaMTXAPI)
	srt, _ := FetchSRTConns(ctx, s.cfg.MediaMTXAPI)
	webrtc, _ := FetchWebRTCSessions(ctx, s.cfg.MediaMTXAPI)

	onAir := false
	for _, p := range paths.Items {
		if p.Name == s.cfg.AutoOpenPath && p.Ready {
			onAir = true
			break
		}
	}

	launchdOK := LaunchdRunning("tv.interdependent.mediamtx")
	tunnelOK := LaunchdRunning("tv.interdependent.cloudflared")

	writeJSON(w, map[string]any{
		"onAir":        onAir,
		"mediamtxOK":   pathsErr == nil,
		"launchdMtx":   launchdOK,
		"launchdTun":   tunnelOK,
		"paths":        paths.Items,
		"hlsMuxers":    hls.Items,
		"srtConns":     srt.Items,
		"webrtcSess":   webrtc.Items,
		"publishers":   countPublishers(paths),
		"readers":      countReaders(paths),
	})
}

func (s *Server) handleMediaMTXProxy(w http.ResponseWriter, r *http.Request) {
	sub := strings.TrimPrefix(r.URL.Path, "/api/mediamtx/")
	if sub == "" {
		http.Error(w, "missing path", 400)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	url := s.cfg.MediaMTXAPI + "/v3/" + sub
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) handleSystem(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	writeJSON(w, CollectSystem(ctx))
}

func (s *Server) handleNetwork(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	writeJSON(w, CollectNetwork(ctx, s.cfg))
}

func (s *Server) handleCloudflare(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	writeJSON(w, CollectCloudflare(ctx, s.cfg))
}

func (s *Server) handleGitHub(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	writeJSON(w, CollectGitHub(ctx, s.cfg))
}

func (s *Server) handleStreamHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"overall":   s.health.Overall(),
		"incidents": s.health.Snapshot(),
	})
}

func (s *Server) handleLogHistory(w http.ResponseWriter, r *http.Request) {
	src := r.URL.Query().Get("source")
	if src == "" {
		writeJSON(w, s.hub.AllHistory())
		return
	}
	writeJSON(w, s.hub.History(src))
}

func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch, cancel := s.hub.Subscribe()
	defer cancel()

	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ping.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case line, ok := <-ch:
			if !ok {
				return
			}
			b, _ := json.Marshal(line)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/action/")
	var out string
	var err error
	switch name {
	case "restart-mediamtx":
		out, err = runCmd("launchctl", "kickstart", "-k", "gui/"+uid()+"/tv.interdependent.mediamtx")
	case "restart-cloudflared":
		out, err = runCmd("launchctl", "kickstart", "-k", "gui/"+uid()+"/tv.interdependent.cloudflared")
	case "open-site":
		out, err = runCmd("open", "https://interdependent.tv")
	case "open-public-stream":
		out, err = runCmd("open", "https://"+s.cfg.PublicHost+"/program/")
	case "open-local-hls":
		out, err = runCmd("open", "http://127.0.0.1:8888/program/")
	default:
		http.Error(w, "unknown action", 400)
		return
	}
	resp := map[string]any{"ok": err == nil, "output": out}
	if err != nil {
		resp["error"] = err.Error()
	}
	writeJSON(w, resp)
}

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
