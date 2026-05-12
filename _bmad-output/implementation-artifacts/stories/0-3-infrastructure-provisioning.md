---
story_id: 0.3
story_key: 0-3-infrastructure-provisioning
epic: 0
sprint: Pre-Sprint-1 Prep
title: Cloudflare + Neon + DNS infrastructure provisioning
status: ready-for-dev
priority: Must (blocker for all Sprint 1 stories)
owners:
  - Chris (end-to-end account ownership)
effort: M
dependencies: []
blocks: ["1.1", "1.2", "1.5", "4.4"]
risks: ["R1"]
project_context_refs:
  - "Cat 1 (Cloudflare Workers + OpenNext + Neon HTTP driver, edge runtime contract)"
  - "Cat 4 (k6 against deployed Workers preview, NEVER localhost)"
  - "Cat 7 #9 (export const runtime = 'edge')"
pivot_doc_refs: ["§5 ADR-001 (Cloudflare Workers)", "§5 ADR-002 (Neon Postgres)", "§14 scale path"]
decisions_resolved: ["D1-INFRA (added 2026-05-12 party-mode readiness review)", "Plan C analytics (2026-05-12 party — Mux Data SDK on R2 delivery, no Mux Video)"]
---

# Story 0.3: Cloudflare + Neon + DNS Infrastructure Provisioning

## User Story

As **Chris (account owner)**,
I want Cloudflare, Neon, and DNS fully provisioned with a deployable Workers preview environment,
So that every downstream Sprint 1+ story has a real target to deploy to and test against — not a localhost simulacrum.

## Context

This story was added 2026-05-12 during the implementation-readiness review (party-mode consensus, 6/6 vote for "new Story 0.3 vs. baking into 1.1/1.2 ACs vs. README"). Stories 1.1 (`pnpm drizzle-kit migrate` against Neon), 1.2 (Workers edge route handlers), 1.5 (Lighthouse on deployed Workers preview), 2.5 (k6 on deployed preview), and 4.4 (500-concurrent on deployed preview) ALL assume this infrastructure exists. Without it, every story's "deployed preview" AC is non-actionable.

Per project-context.md Cat 4 (load testing rule): k6 MUST target deployed Workers. `next dev` does not emulate Workers KV TTL, edge runtime constraints, or cold-start behavior. This story makes that target real.

## Acceptance Criteria

### Cloudflare account + Workers project

1. **Cloudflare account exists** (or use Chris's existing). Account ID captured in `.dev.vars` or password manager.
2. **Workers project created** via `wrangler init` (or equivalent) targeting Next.js + OpenNext adapter. Project name documented in `wrangler.toml`.
3. **`wrangler.toml` committed to repo** with:
   - `main = "src/worker.ts"` (or OpenNext-generated entry).
   - `compatibility_date` set to today (2026-05-12) or later.
   - `compatibility_flags = ["nodejs_compat"]` (only if needed by OpenNext; verify).
   - `[[kv_namespaces]]` block declaring the `KV` binding with namespace ID.
   - `[[r2_buckets]]` block declaring HLS bucket binding (from Story 0.2).
   - `[vars]` for non-secret config; secrets via `wrangler secret put`.
4. **KV namespace provisioned** via `wrangler kv:namespace create KV`. Namespace ID captured in `wrangler.toml`.
5. **Workers preview deploys.** `wrangler deploy --env preview` (or equivalent) succeeds and returns a stable preview URL (e.g. `tv-poc-preview.<account>.workers.dev`).
6. **Smoke test:** `curl -I https://<preview-url>/api/healthz` (placeholder route) returns 200 OR 404 (proving the deployment serves traffic, even if route isn't built yet).

### Neon Postgres project + branch

7. **Neon account exists** (or use Chris's existing). Org ID + project ID captured.
8. **Neon project created** for `tv-poc`. Connection string for the default `main` branch captured.
9. **Neon dev branch created** (`dev` or `preview` named branch). Separate connection string. This is the target for Story 1.1's `drizzle-kit migrate` runs.
10. **Connection strings stored:**
    - Local dev: `.dev.vars` with `DATABASE_URL` (gitignored).
    - Workers preview: `wrangler secret put DATABASE_URL --env preview` with the dev-branch URL.
    - Workers production: `wrangler secret put DATABASE_URL --env production` with the main-branch URL (or deferred until production cutover).
11. **`neonConfig.fetchConnectionCache = true`** documented as a required Drizzle client init (carries through to Story 1.1's schema/client setup).
12. **Smoke test:** `psql $DATABASE_URL -c 'SELECT 1'` against the dev branch returns `1`.

### DNS (apex + subdomain)

13. **`interdependent.tv` apex** resolves to the Workers preview env (or production env, when promoted). Configured via Cloudflare Dashboard → DNS → CNAME or via Workers Routes. SSL cert valid (Cloudflare-managed Universal SSL).
14. **`cdn.interdependent.tv` subdomain** resolves to the R2 HLS bucket per Story 0.2 (custom domain feature on R2). Already covered by Story 0.2 AC 2 — cross-referenced here for completeness. SSL cert valid.
15. **`curl -I https://interdependent.tv/` returns 200** (or 404 if no route yet — proves the route binding works).
16. **DNS propagation confirmed** via `dig interdependent.tv` and `dig cdn.interdependent.tv` showing Cloudflare resolution.

### Mux Data account (analytics — QoE telemetry, free tier)

17a. **Mux Data account exists** (or Chris's existing Mux account reused — Mux Data is a separate product surface from Mux Video and lives under the same login). Free tier covers 100K plays/month — POC scale fits comfortably.
17b. **Mux Data Environment Key (`MUX_DATA_KEY`)** captured for the preview environment. **NOTE per Plan C consensus (2026-05-12):** Mux Data is the viewer-side QoE SDK only — `@mux/mux-data` — instrumented on the hls.js + Safari native player in Story 1.4. We are NOT adopting Mux Video as a delivery layer. Delivery stays R2 + FFmpeg.
17c. **`MUX_DATA_KEY`** stored in `.dev.vars` (gitignored) and `.dev.vars.example` (placeholder). Workers preview secret optional — the SDK runs client-side and reads the key from a public env var (`NEXT_PUBLIC_MUX_DATA_KEY` in Next.js App Router). Document this distinction in `docs/INFRASTRUCTURE.md`.

### Repo + secrets hygiene

17. **`.dev.vars` is gitignored.** Verified in `.gitignore`.
18. **`.dev.vars.example`** committed with placeholder keys (`DATABASE_URL=`, `CF_ACCOUNT_ID=`, `NEXT_PUBLIC_MUX_DATA_KEY=`, etc.). Developers copy it on clone.
19. **No secrets in `wrangler.toml` itself.** All secrets via `wrangler secret put`. Account ID + KV namespace IDs are not secret and can live in `wrangler.toml`. `NEXT_PUBLIC_MUX_DATA_KEY` is publishable (client-side) and lives in `.dev.vars` alongside non-secrets.

### Documentation

20. **`docs/INFRASTRUCTURE.md` (or `scripts/README.md` extension)** documents:
    - Cloudflare account ID + Workers project name.
    - Neon project ID + branch URLs (redacted).
    - DNS record summary.
    - One-command bootstrap for a new dev (`cp .dev.vars.example .dev.vars`, fill values, `pnpm install`, `pnpm dev`).
    - How to deploy a preview (`wrangler deploy --env preview`).

## Acceptance Tests / Verification

**Manual verification checklist:**

- [ ] `wrangler whoami` shows authenticated as Chris's account.
- [ ] `wrangler kv:namespace list` shows the `KV` namespace.
- [ ] `wrangler r2 bucket list` shows the HLS bucket (from 0.2).
- [ ] `dig interdependent.tv` returns a Cloudflare IP.
- [ ] `dig cdn.interdependent.tv` returns a Cloudflare IP.
- [ ] `curl -I https://interdependent.tv/` returns a 2xx, 3xx, or 4xx (NOT timeout / NXDOMAIN).
- [ ] `curl -I https://cdn.interdependent.tv/` returns a 2xx/3xx/4xx.
- [ ] `psql $DATABASE_URL -c 'SELECT 1'` works against the Neon dev branch.
- [ ] `.dev.vars` present locally; `.dev.vars.example` committed; both listed correctly in `.gitignore`.
- [ ] `wrangler deploy --env preview` succeeds and prints a preview URL.

**Carry-forward acceptance for Story 1.1:** The Neon dev branch is the target for `drizzle-kit migrate`.

**Carry-forward acceptance for Story 1.5 + 2.5 + 4.4:** The Workers preview URL is the target for Lighthouse + k6 runs.

## Dependencies

- **Upstream:** None — this is the unblocker for the entire technical track. Story 0.1 (asset inventory) and Story 0.2 (encoding pipeline) are parallel.
- **Downstream (blocks):**
  - 1.1 — Drizzle migrate needs Neon dev branch.
  - 1.2 — Workers edge routes need `wrangler.toml` + KV binding.
  - 1.5 — Cold-load baseline needs deployed Workers preview.
  - 2.5 — Concurrency soak test needs deployed Workers preview.
  - 4.4 — 500-concurrent load test needs deployed Workers preview.

## Risk Alignment (pivot §16)

| ID | Risk | Status | Mitigation |
|---|---|---|---|
| R1 | Asset encoding pipeline absent | Open / High | This story does NOT close R1 (Story 0.2 owns that). It DOES close the dependency-of-R1 question "where does the pipeline upload to" — `cdn.interdependent.tv` resolves to a real R2 bucket. |
| (new) Provisioning never happens | Implicit until this story | This story IS the closure. |

## Implementation Notes

- **Cloudflare Workers vs. Pages.** OpenNext for Next.js targets Workers, NOT Pages, as of Next.js 14+. Pivot ADR-001 references Workers. Use `@opennextjs/cloudflare` adapter; output Worker is what `wrangler deploy` ships.
- **Environments:** `wrangler.toml` supports `[env.preview]` and `[env.production]` sections. Use preview for everything in Sprints 1–3; promote to production for Sprint 4 demo or whenever the demo URL is needed publicly.
- **DNS gotchas:** Universal SSL provisioning can take 5–15 minutes after CNAME setup. Run AC 13/14 verifications AFTER waiting. If SSL is still "pending" after 30 min, check that the zone is on Cloudflare nameservers (not just orange-cloud proxied via another DNS provider).
- **R2 custom domain (`cdn.interdependent.tv`):** Configured in Cloudflare Dashboard → R2 → bucket → Settings → Custom Domain. This is Story 0.2's responsibility but the DNS record lives in the same zone as the apex — verify they don't collide.
- **Cost ceiling for POC:**
  - Workers Free: 100k req/day. Sprint 4's 500-concurrent 30-min run = ~30k req. Fits.
  - Neon Free: 0.5GB storage, autosuspend after 5min idle. Fits POC.
  - R2 Free: 10GB storage, 1M Class A ops/month, 10M Class B ops/month. Sprint 4 extrapolation needed (Story 4.4 AC 4).
  - **If any limit is hit during Sprint 4 load test:** documented as a known constraint, NOT a blocker. Upgrade to paid tier ($5/mo Workers, $0.04/GB-month R2) is a Chris call, not a story.
- **Secrets in CI (if added later):** Cloudflare API token + Neon API key go in CI secrets, not in repo. POC may not need CI deploy automation; manual `wrangler deploy` is acceptable.

## Definition of Done

- [ ] AC 1–20 met.
- [ ] All manual verification checks green.
- [ ] `docs/INFRASTRUCTURE.md` committed.
- [ ] Sprint-status.yaml entry `0-3-infrastructure-provisioning` flips to `done`.
- [ ] Stories 1.1, 1.2, 1.5, 4.4 unblocked.
