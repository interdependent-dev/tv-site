package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := Config{
		Addr:        ":0",
		MediaMTXAPI: "http://127.0.0.1:1", // unreachable so external calls fail fast
		PublicHost:  "live.invalid",
		GHRepo:      "owner/repo",
		Ports:       []PortSpec{{"Test", "127.0.0.1", 9, "tcp"}},
	}
	return NewServer(cfg, NewLogHub(), NewHealthMonitor())
}

func TestRouteIndexServesEmbeddedHTML(t *testing.T) {
	srv := newTestServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "MISSION CONTROL") {
		t.Error("index.html body should contain MISSION CONTROL")
	}
}

func TestRouteConfigJSON(t *testing.T) {
	srv := newTestServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/config", nil)
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("/api/config = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"publicHost", "ports", "ghRepo"} {
		if !strings.Contains(body, want) {
			t.Errorf("config response missing %q: %s", want, body)
		}
	}
}

func TestRouteHealthEmptyOk(t *testing.T) {
	srv := newTestServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/health/stream", nil)
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("/api/health/stream = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"overall":"ok"`) {
		t.Errorf("empty health should report overall=ok, got %s", rec.Body.String())
	}
}

func TestRouteHealthWithIncident(t *testing.T) {
	srv := newTestServer(t)
	srv.health.process(LogLine{
		Source: "mediamtx",
		Text:   "ERR something fatal",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/health/stream", nil)
	srv.http.Handler.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, `"overall":"err"`) {
		t.Errorf("health with err incident should report overall=err, got %s", body)
	}
	if !strings.Contains(body, "Mediamtx error") {
		t.Errorf("health response should include incident title, got %s", body)
	}
}

func TestRouteLogsHistoryFiltered(t *testing.T) {
	srv := newTestServer(t)
	srv.hub.Publish(LogLine{Source: "a", Text: "from-a"})
	srv.hub.Publish(LogLine{Source: "b", Text: "from-b"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/logs/history?source=a", nil)
	srv.http.Handler.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "from-a") {
		t.Error("filtered history should include from-a")
	}
	if strings.Contains(body, "from-b") {
		t.Error("filtered history should NOT include from-b")
	}
}

func TestRouteActionRejectsGet(t *testing.T) {
	srv := newTestServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/action/restart-mediamtx", nil)
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != 405 {
		t.Errorf("GET on /api/action/* should be 405, got %d", rec.Code)
	}
}

func TestRouteUnknownActionRejected(t *testing.T) {
	srv := newTestServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/action/nope", nil)
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Errorf("unknown action POST should be 400, got %d", rec.Code)
	}
}
