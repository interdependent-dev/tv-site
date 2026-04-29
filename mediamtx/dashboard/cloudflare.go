package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

var cfClient = &http.Client{Timeout: 7 * time.Second}

type CFStatus struct {
	Status struct {
		Indicator   string `json:"indicator"`
		Description string `json:"description"`
	} `json:"status"`
}

type CFIncidents struct {
	Incidents []struct {
		Name    string `json:"name"`
		Status  string `json:"status"`
		Impact  string `json:"impact"`
		Shortlink string `json:"shortlink"`
		Updates []struct {
			Body      string `json:"body"`
			CreatedAt string `json:"created_at"`
		} `json:"incident_updates"`
	} `json:"incidents"`
}

type CFZone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type CFZoneResp struct {
	Success bool     `json:"success"`
	Result  []CFZone `json:"result"`
	Errors  []any    `json:"errors"`
}

type CFAnalyticsResp struct {
	Success bool `json:"success"`
	Result  struct {
		Totals struct {
			Requests struct {
				All    int64 `json:"all"`
				Cached int64 `json:"cached"`
			} `json:"requests"`
			Bandwidth struct {
				All    int64 `json:"all"`
				Cached int64 `json:"cached"`
			} `json:"bandwidth"`
			Threats struct {
				All int64 `json:"all"`
			} `json:"threats"`
		} `json:"totals"`
	} `json:"result"`
}

func CollectCloudflare(ctx context.Context, cfg Config) map[string]any {
	out := map[string]any{
		"tokenSet":   cfg.CFAPIToken != "",
		"zoneName":   cfg.CFZoneName,
		"publicHost": cfg.PublicHost,
	}

	if s, err := cfStatus(ctx); err == nil {
		out["platformStatus"] = s.Status.Description
		out["platformIndicator"] = s.Status.Indicator
	}
	if inc, err := cfIncidents(ctx); err == nil {
		out["incidents"] = inc.Incidents
	}

	if cfg.CFAPIToken != "" {
		if zoneID, status, err := cfZoneID(ctx, cfg.CFAPIToken, cfg.CFZoneName); err == nil {
			out["zoneID"] = zoneID
			out["zoneStatus"] = status
			if a, err := cfAnalytics(ctx, cfg.CFAPIToken, zoneID); err == nil {
				totReq := a.Result.Totals.Requests.All
				cachedReq := a.Result.Totals.Requests.Cached
				hitPct := 0.0
				if totReq > 0 {
					hitPct = 100.0 * float64(cachedReq) / float64(totReq)
				}
				out["requests24h"] = totReq
				out["cachedRequests24h"] = cachedReq
				out["bandwidth24h"] = a.Result.Totals.Bandwidth.All
				out["threats24h"] = a.Result.Totals.Threats.All
				out["cacheHitPct"] = hitPct
			} else {
				out["analyticsError"] = err.Error()
			}
		} else {
			out["zoneError"] = err.Error()
		}
	}

	if expires, issuer, err := tlsExpiry(ctx, cfg.PublicHost); err == nil {
		out["tlsExpiry"] = expires
		out["tlsIssuer"] = issuer
	} else {
		out["tlsError"] = err.Error()
	}

	return out
}

func cfStatus(ctx context.Context) (CFStatus, error) {
	var s CFStatus
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.cloudflarestatus.com/api/v2/status.json", nil)
	resp, err := cfClient.Do(req)
	if err != nil {
		return s, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&s)
	return s, err
}

func cfIncidents(ctx context.Context) (CFIncidents, error) {
	var s CFIncidents
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.cloudflarestatus.com/api/v2/incidents/unresolved.json", nil)
	resp, err := cfClient.Do(req)
	if err != nil {
		return s, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&s)
	return s, err
}

func cfZoneID(ctx context.Context, token, name string) (string, string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.cloudflare.com/client/v4/zones?name="+name, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := cfClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	var out CFZoneResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	if !out.Success || len(out.Result) == 0 {
		return "", "", fmt.Errorf("zone %q not found (or token lacks access)", name)
	}
	return out.Result[0].ID, out.Result[0].Status, nil
}

func cfAnalytics(ctx context.Context, token, zoneID string) (CFAnalyticsResp, error) {
	var out CFAnalyticsResp
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/analytics/dashboard?since=-1440&until=0", zoneID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := cfClient.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

func tlsExpiry(ctx context.Context, host string) (time.Time, string, error) {
	d := &net.Dialer{Timeout: 4 * time.Second}
	conn, err := tls.DialWithDialer(d, "tcp", host+":443", &tls.Config{ServerName: host})
	if err != nil {
		return time.Time{}, "", err
	}
	defer conn.Close()
	chain := conn.ConnectionState().PeerCertificates
	if len(chain) == 0 {
		return time.Time{}, "", fmt.Errorf("no peer cert")
	}
	return chain[0].NotAfter, chain[0].Issuer.CommonName, nil
}
