package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Iface struct {
	Name      string   `json:"name"`
	MAC       string   `json:"mac"`
	MTU       int      `json:"mtu"`
	Up        bool     `json:"up"`
	Addrs     []string `json:"addrs"`
	LinkSpeed string   `json:"linkSpeed"`
	Media     string   `json:"media"`
}

type PortStatus struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Proto     string `json:"proto"`
	Open      bool   `json:"open"`
	LatencyMs int64  `json:"latencyMs"`
}

type Ping struct {
	Target    string  `json:"target"`
	OK        bool    `json:"ok"`
	LatencyMs float64 `json:"latencyMs"`
	Loss      float64 `json:"lossPct"`
}

func CollectNetwork(ctx context.Context, cfg Config) map[string]any {
	out := map[string]any{}

	out["interfaces"] = listInterfaces()
	out["lanIPs"] = primaryLANs()
	out["publicIP"] = publicIP(ctx)
	out["dnsLookup"] = dnsLookup(ctx, cfg.PublicHost)
	out["ports"] = probePorts(ctx, cfg.Ports)
	out["pings"] = []Ping{
		ping(ctx, defaultGateway()),
		ping(ctx, "1.1.1.1"),
		ping(ctx, "8.8.8.8"),
		ping(ctx, cfg.PublicHost),
	}
	return out
}

func listInterfaces() []Iface {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []Iface
	for _, i := range ifs {
		if strings.HasPrefix(i.Name, "lo") || strings.HasPrefix(i.Name, "utun") || strings.HasPrefix(i.Name, "awdl") || strings.HasPrefix(i.Name, "llw") || strings.HasPrefix(i.Name, "anpi") || strings.HasPrefix(i.Name, "ap") {
			continue
		}
		addrs, _ := i.Addrs()
		var as []string
		for _, a := range addrs {
			as = append(as, a.String())
		}
		speed, media := ifconfigDetail(i.Name)
		out = append(out, Iface{
			Name:      i.Name,
			MAC:       i.HardwareAddr.String(),
			MTU:       i.MTU,
			Up:        i.Flags&net.FlagUp != 0,
			Addrs:     as,
			LinkSpeed: speed,
			Media:     media,
		})
	}
	return out
}

var mediaRE = regexp.MustCompile(`media:\s*([^\n]+)`)
var statusRE = regexp.MustCompile(`status:\s*(\w+)`)

func ifconfigDetail(name string) (speed, media string) {
	out, err := exec.Command("ifconfig", name).CombinedOutput()
	if err != nil {
		return "", ""
	}
	if m := mediaRE.FindStringSubmatch(string(out)); len(m) > 1 {
		media = strings.TrimSpace(m[1])
	}
	if m := statusRE.FindStringSubmatch(string(out)); len(m) > 1 {
		if media != "" {
			media += " (" + m[1] + ")"
		} else {
			media = m[1]
		}
	}
	return speed, media
}

func primaryLANs() []string {
	var out []string
	ifs, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, i := range ifs {
		if i.Flags&net.FlagLoopback != 0 || i.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if ip4 := ipnet.IP.To4(); ip4 != nil && !ip4.IsLoopback() && !ip4.IsLinkLocalUnicast() {
				out = append(out, ip4.String())
			}
		}
	}
	return out
}

func publicIP(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "dig", "+short", "+time=2", "+tries=1", "myip.opendns.com", "@resolver1.opendns.com")
	if out, err := cmd.Output(); err == nil {
		ip := strings.TrimSpace(string(out))
		if ip != "" {
			return ip
		}
	}
	cli := &http.Client{Timeout: 3 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.ipify.org", nil)
	resp, err := cli.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return strings.TrimSpace(string(b))
}

func dnsLookup(ctx context.Context, host string) map[string]any {
	start := time.Now()
	r := net.Resolver{}
	addrs, err := r.LookupHost(ctx, host)
	dur := time.Since(start).Milliseconds()
	out := map[string]any{"host": host, "ms": dur}
	if err != nil {
		out["error"] = err.Error()
	} else {
		out["addrs"] = addrs
	}
	return out
}

func probePorts(ctx context.Context, ports []PortSpec) []PortStatus {
	out := make([]PortStatus, len(ports))
	for i, p := range ports {
		start := time.Now()
		proto := p.Proto
		if proto == "" {
			proto = "tcp"
		}
		ps := PortStatus{Name: p.Name, Host: p.Host, Port: p.Port, Proto: proto}
		if proto == "udp" {
			// UDP is connectionless, so we can't "dial" it meaningfully.
			// Trick: try to bind locally. If bind fails with "address in use",
			// something is already listening on that port → "open".
			// (Only meaningful when probing 127.0.0.1.)
			addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(p.Host, strconv.Itoa(p.Port)))
			if err == nil {
				conn, err := net.ListenUDP("udp", addr)
				if err != nil {
					ps.Open = true // bind failed → port is taken
				} else {
					conn.Close() // we got it → nothing was listening
				}
			}
		} else {
			d := net.Dialer{Timeout: 700 * time.Millisecond}
			conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(p.Host, strconv.Itoa(p.Port)))
			if err == nil {
				conn.Close()
				ps.Open = true
			}
		}
		ps.LatencyMs = time.Since(start).Milliseconds()
		out[i] = ps
	}
	return out
}

var pingRE = regexp.MustCompile(`time=([\d.]+)\s*ms`)
var pingLossRE = regexp.MustCompile(`(\d+(?:\.\d+)?)% packet loss`)

func ping(ctx context.Context, target string) Ping {
	p := Ping{Target: target}
	if target == "" {
		return p
	}
	cmd := exec.CommandContext(ctx, "ping", "-c", "2", "-W", "1500", "-q", target)
	out, err := cmd.CombinedOutput()
	s := string(out)
	if err == nil {
		p.OK = true
	}
	if m := pingRE.FindStringSubmatch(s); len(m) > 1 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			p.LatencyMs = v
		}
	}
	if m := pingLossRE.FindStringSubmatch(s); len(m) > 1 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			p.Loss = v
			if v >= 100 {
				p.OK = false
			}
		}
	}
	return p
}

func defaultGateway() string {
	out, err := exec.Command("route", "-n", "get", "default").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "gateway:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
		}
	}
	return ""
}
