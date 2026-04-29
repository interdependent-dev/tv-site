package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

//go:embed static
var staticFS embed.FS

type Config struct {
	Addr           string
	MediaMTXAPI    string
	MediaMTXLog    string
	CloudflaredLog string
	DashboardLog   string
	CFAPIToken     string
	CFZoneName     string
	GHRepo         string
	PublicHost     string
	AutoOpenPath   string
	Ports          []PortSpec
}

type PortSpec struct {
	Name  string
	Host  string
	Port  int
	Proto string
}

func main() {
	log.SetFlags(log.Ltime)

	here, _ := os.Getwd()
	parent := filepath.Dir(here)

	cfg := Config{
		Addr:           env("DASHBOARD_ADDR", ":9998"),
		MediaMTXAPI:    env("MEDIAMTX_API", "http://127.0.0.1:9997"),
		MediaMTXLog:    env("MEDIAMTX_LOG", filepath.Join(parent, "mediamtx.log")),
		CloudflaredLog: env("CLOUDFLARED_LOG", filepath.Join(parent, "cloudflared.log")),
		DashboardLog:   env("DASHBOARD_LOG", filepath.Join(here, "dashboard.log")),
		CFAPIToken:     os.Getenv("CF_API_TOKEN"),
		CFZoneName:     env("CF_ZONE_NAME", "interdependent.dev"),
		GHRepo:         env("GH_REPO", "interdependent-dev/tv-site"),
		PublicHost:     env("PUBLIC_HOST", "live.interdependent.dev"),
		AutoOpenPath:   env("AUTO_OPEN_PATH", "program"),
		Ports: []PortSpec{
			{"MediaMTX API", "127.0.0.1", 9997, "tcp"},
			{"HLS", "127.0.0.1", 8888, "tcp"},
			{"WebRTC/WHEP signal", "127.0.0.1", 8889, "tcp"},
			{"SRT Ingest", "127.0.0.1", 8890, "udp"},
			{"WebRTC media", "127.0.0.1", 8189, "udp"},
			{"Dashboard", "127.0.0.1", 9998, "tcp"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := NewLogHub()
	hub.Add("mediamtx", cfg.MediaMTXLog)
	hub.Add("cloudflared", cfg.CloudflaredLog)
	hub.Add("dashboard", cfg.DashboardLog)
	hub.Start(ctx)

	health := NewHealthMonitor()
	health.Start(ctx, hub)

	go RunWatcher(ctx, cfg)

	srv := NewServer(cfg, hub, health)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("shutting down")
		cancel()
		srv.Shutdown(context.Background())
	}()

	log.Printf("mission control → http://localhost%s", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Println(err)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
