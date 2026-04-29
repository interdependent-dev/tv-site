package main

import (
	"context"
	"log"
	"os/exec"
	"time"
)

func RunWatcher(ctx context.Context, cfg Config) {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	wasReady := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		c, cancel := context.WithTimeout(ctx, 2*time.Second)
		paths, err := FetchPaths(c, cfg.MediaMTXAPI)
		cancel()
		if err != nil {
			wasReady = false
			continue
		}
		ready := false
		for _, p := range paths.Items {
			if p.Name == cfg.AutoOpenPath && p.Ready {
				ready = true
				break
			}
		}
		if ready && !wasReady {
			log.Printf("publisher detected on /%s — opening dashboard", cfg.AutoOpenPath)
			_ = exec.Command("open", "http://localhost"+cfg.Addr).Start()
		}
		wasReady = ready
	}
}
