package main

import (
	"context"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type SystemStats struct {
	Now           time.Time      `json:"now"`
	Hostname      string         `json:"hostname"`
	OS            string         `json:"os"`
	Arch          string         `json:"arch"`
	GoVersion     string         `json:"goVersion"`
	NumCPU        int            `json:"numCPU"`
	LoadAvg       []float64      `json:"loadAvg"`
	CPUUserPct    float64        `json:"cpuUserPct"`
	CPUSysPct     float64        `json:"cpuSysPct"`
	CPUIdlePct    float64        `json:"cpuIdlePct"`
	MemTotalBytes int64          `json:"memTotalBytes"`
	MemUsedBytes  int64          `json:"memUsedBytes"`
	MemFreeBytes  int64          `json:"memFreeBytes"`
	Disk          DiskInfo       `json:"disk"`
	Uptime        string         `json:"uptime"`
	Launchd       map[string]any `json:"launchd"`
	StartedAt     time.Time      `json:"startedAt"`
}

type DiskInfo struct {
	Mount      string `json:"mount"`
	UsedBytes  int64  `json:"usedBytes"`
	TotalBytes int64  `json:"totalBytes"`
	FreeBytes  int64  `json:"freeBytes"`
}

var startedAt = time.Now()

func CollectSystem(ctx context.Context) SystemStats {
	host, _ := exec.Command("hostname", "-s").Output()
	upt, _ := exec.Command("uptime").Output()

	cpuU, cpuS, cpuI := topCPU(ctx)
	memT, memU, memF := vmStat(ctx)

	s := SystemStats{
		Now:           time.Now(),
		Hostname:      strings.TrimSpace(string(host)),
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		GoVersion:     runtime.Version(),
		NumCPU:        runtime.NumCPU(),
		LoadAvg:       loadAvg(),
		CPUUserPct:    cpuU,
		CPUSysPct:     cpuS,
		CPUIdlePct:    cpuI,
		MemTotalBytes: memT,
		MemUsedBytes:  memU,
		MemFreeBytes:  memF,
		Disk:          diskUsage(),
		Uptime:        strings.TrimSpace(string(upt)),
		Launchd: map[string]any{
			"mediamtx":    LaunchdRunning("tv.interdependent.mediamtx"),
			"cloudflared": LaunchdRunning("tv.interdependent.cloudflared"),
			"dashboard":   LaunchdRunning("tv.interdependent.dashboard"),
		},
		StartedAt: startedAt,
	}
	return s
}

func loadAvg() []float64 {
	out, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		return nil
	}
	parts := strings.Fields(strings.Trim(string(out), "{}\n "))
	if len(parts) < 3 {
		return nil
	}
	la := make([]float64, 0, 3)
	for _, p := range parts[:3] {
		if v, err := strconv.ParseFloat(p, 64); err == nil {
			la = append(la, v)
		}
	}
	return la
}

var topRE = regexp.MustCompile(`CPU usage:\s*([\d.]+)% user,\s*([\d.]+)% sys,\s*([\d.]+)% idle`)

func topCPU(ctx context.Context) (user, sys, idle float64) {
	cmd := exec.CommandContext(ctx, "top", "-l", "1", "-n", "0")
	out, err := cmd.Output()
	if err != nil {
		return
	}
	if m := topRE.FindStringSubmatch(string(out)); len(m) > 3 {
		user, _ = strconv.ParseFloat(m[1], 64)
		sys, _ = strconv.ParseFloat(m[2], 64)
		idle, _ = strconv.ParseFloat(m[3], 64)
	}
	return
}

func vmStat(ctx context.Context) (total, used, free int64) {
	totOut, err := exec.CommandContext(ctx, "sysctl", "-n", "hw.memsize").Output()
	if err == nil {
		total, _ = strconv.ParseInt(strings.TrimSpace(string(totOut)), 10, 64)
	}
	psize := int64(16384) // mac default
	if pOut, err := exec.CommandContext(ctx, "sysctl", "-n", "hw.pagesize").Output(); err == nil {
		if v, err := strconv.ParseInt(strings.TrimSpace(string(pOut)), 10, 64); err == nil {
			psize = v
		}
	}
	vmOut, err := exec.CommandContext(ctx, "vm_stat").Output()
	if err != nil {
		return
	}
	pages := map[string]int64{}
	for _, line := range strings.Split(string(vmOut), "\n") {
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.TrimSuffix(val, ".")
		if v, err := strconv.ParseInt(val, 10, 64); err == nil {
			pages[key] = v
		}
	}
	freePages := pages["Pages free"] + pages["Pages speculative"]
	free = freePages * psize
	used = total - free
	return
}

func diskUsage() DiskInfo {
	out, err := exec.Command("df", "-k", "/").Output()
	if err != nil {
		return DiskInfo{}
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return DiskInfo{}
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 9 {
		return DiskInfo{}
	}
	total, _ := strconv.ParseInt(fields[1], 10, 64)
	used, _ := strconv.ParseInt(fields[2], 10, 64)
	free, _ := strconv.ParseInt(fields[3], 10, 64)
	return DiskInfo{
		Mount:      fields[len(fields)-1],
		TotalBytes: total * 1024,
		UsedBytes:  used * 1024,
		FreeBytes:  free * 1024,
	}
}

func LaunchdRunning(label string) bool {
	out, err := exec.Command("launchctl", "list", label).Output()
	if err != nil {
		return false
	}
	s := string(out)
	if !strings.Contains(s, label) {
		return false
	}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "\"PID\" =") {
			return !strings.Contains(line, "= 0;") && !strings.Contains(line, "= -1;")
		}
	}
	return strings.Contains(s, "\"PID\"")
}

func uid() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return u.Uid
}
