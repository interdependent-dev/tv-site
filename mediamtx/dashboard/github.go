package main

import (
	"context"
	"encoding/json"
	"os/exec"
)

type GHRun struct {
	DatabaseID   int64  `json:"databaseId"`
	Name         string `json:"name"`
	DisplayTitle string `json:"displayTitle"`
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	HeadBranch   string `json:"headBranch"`
	HeadSha      string `json:"headSha"`
	URL          string `json:"url"`
	Event        string `json:"event"`
}

func CollectGitHub(ctx context.Context, cfg Config) map[string]any {
	out := map[string]any{"repo": cfg.GHRepo}

	cmd := exec.CommandContext(ctx, "gh", "run", "list",
		"--repo", cfg.GHRepo,
		"--limit", "10",
		"--json", "databaseId,name,displayTitle,status,conclusion,createdAt,updatedAt,headBranch,headSha,url,event",
	)
	stdout, err := cmd.Output()
	if err != nil {
		out["error"] = "gh CLI not authed or unreachable: " + err.Error()
		return out
	}
	var runs []GHRun
	if err := json.Unmarshal(stdout, &runs); err != nil {
		out["error"] = err.Error()
		return out
	}
	out["runs"] = runs

	pagesCmd := exec.CommandContext(ctx, "gh", "api",
		"repos/"+cfg.GHRepo+"/pages",
	)
	if pagesOut, err := pagesCmd.Output(); err == nil {
		var pages map[string]any
		if json.Unmarshal(pagesOut, &pages) == nil {
			out["pages"] = pages
		}
	}

	return out
}
