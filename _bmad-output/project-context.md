---
project_name: 'tv'
user_name: 'Chris'
date: '2026-05-12'
sections_completed: ['technology_stack', 'language_rules', 'framework_rules', 'testing_rules', 'quality_rules', 'workflow_rules', 'anti_patterns']
status: 'complete'
optimized_for_llm: true
source_document: 'tv_poc_plan.md v1.0 (Linear Channel Playback POC, Owner: Vik, 2026-05-12)'
rule_count: ~150
section_count: 7
decisions_resolved: '§17.1, 17.2, 17.4, 17.5, 17.6, 17.7 — Closed 2026-05-12; 17.3 — Gated pending Story 0.1 close (Vik+Chris)'
mvp_triage_resolutions_2026_05_12: 'D2 (owned content), D4 (ambient ch1), D6+D7 (Vik self-attestation), D12 (in-house FFmpeg), D13 (Chris owns encoding), D14 (apex viewer URLs + cdn subdomain for HLS), D16 (staggered encode), D18 (tv-masters R2 bucket)'
---

# Project Context for AI Agents

_This file contains critical rules and patterns that AI agents must follow when implementing code in this project. Focus on unobvious details that agents might otherwise miss._

---

## Technology Stack & Versions

> **Status — pre-bootstrap.** The repo currently contains the legacy live-streaming POC (static HTML + MediaMTX). The stack below is the *target* for the Linear Channel Playback POC (Epic TV-1, per `tv_poc_plan.md` v1.0). Until the Next.js scaffold lands, agents MUST NOT assume any of these dependencies exist.

### Monorepo & Build
- **Nx** — workspace manager. App lives at the workspace root or under `apps/tv/`. Confirm path on bootstrap.
- **pnpm** — package manager (implied by `pnpm seed` references in pivot doc §8.2).

### Runtime & Framework
- **Next.js 16** (App Router only — no Pages Router).
- **OpenNext on Cloudflare Workers** — server runtime. **NOT Node.js.**
  - No Node-only APIs (`fs`, `child_process`, `Buffer`, native modules).
  - No long-lived sockets. No `pg.Pool`, no `pg.Client`.
  - Web APIs only: `fetch`, `Request`, `Response`, `crypto.subtle`, `URL`.
  - RSCs that import a Node-only module produce a **silent bundle error at deploy time, not at compile time.** Verify every server-component import against the Cloudflare Workers runtime compatibility list.
- **React 19+** (whatever Next.js 16 ships with).

### Edge Runtime Contract (mandatory)
- **Every route under `app/api/` MUST export `export const runtime = 'edge'`.** Omission silently falls back to Node-compatible mode locally, then fails on Workers at deploy. This is a hard rule.
- **`/api/time` MUST additionally export `export const dynamic = 'force-dynamic'`** to prevent build-time static rendering of `Date.now()`.
- **`env` (Workers bindings) is read from the OpenNext platform context inside route handlers** — not from `process.env`, not from `NEXT_PUBLIC_*`. KV access is `env.KV.get(key)` / `env.KV.put(key, value, { expirationTtl })`. Binding names in `wrangler.toml` MUST match the property names destructured in handlers.

### Data Layer

**Neon Postgres (source of truth)**
- Access via **`@neondatabase/serverless` HTTP driver only.** Never `pg`, `pg.Pool`, or `pg.Client` — Workers cannot open TCP sockets.
- **Set `neonConfig.fetchConnectionCache = true`** to reuse connections across requests in the same isolate. Without it, cold-connection overhead accumulates at scale.
- **`db.transaction()` is HTTP-batched** (single round-trip), not an interactive transaction. Multi-statement interactive logic must run outside the Worker.

**Drizzle ORM**
- Schema lives at `db/schema.ts`. Postgres dialect (`drizzle-orm/pg-core`) only — never SQLite or MySQL builders.
- **Adapter import is `drizzle-orm/neon-http`**, never `drizzle-orm/node-postgres` in deployed code. Wrong import compiles locally and fails in production.
- **`drizzle-kit` is dev-only.** Never import its runtime helpers (`migrate()`, `pushSchema()`) from a deployed Worker — they pull in Node `fs`/`path`.
- **Migrations: `pnpm drizzle-kit migrate`. NEVER `drizzle-kit push` against the production Neon URL** — `push` is destructive and drops unrecognized columns.

**Workers KV (read-through cache)**
- TTLs are endpoint-specific and **load-bearing for correctness**, not just performance (see Caching Rules section, future Category 4).
- Tag-based purge on schedule edits is required; not optional.

**Cloudflare R2 (HLS storage)**
- HLS manifests + segments. Workers→R2 transfers are free; viewer egress is metered.
- **R2 bucket MUST have CORS configured for `interdependent.tv` as an allowed origin before any browser test on Sprint 1 Story 1.4.** HLS segments are fetched directly browser→R2 (not proxied through Workers); without CORS, every segment 404s in Chrome/Firefox. Safari is more permissive — testing only on Safari produces false positives.

### Video Player
- **`hls.js` >= 1.5 for non-Safari** — for `currentTime` seek reliability and stable `MANIFEST_PARSED` timing. **NOT for LL-HLS.** LL-HLS is explicitly out of scope (pivot §19).
- **`lowLatencyMode: false` MUST be set explicitly in hls.js config.** Enabling it on a non-LL-HLS manifest causes hls.js to attempt partial-segment loading that 404s against R2.
- **Native `<video>` on Safari** (HLS plays natively). NO DASH. NO `dash.js`.
- **Seek sequence is non-negotiable:**
  - hls.js path: `MANIFEST_PARSED` event → set `video.currentTime = offsetSec` → call `video.play()`. Never seek before parse. Never play before seek.
  - Safari native path: `loadedmetadata` event → set `currentTime` → call `play()`. Two distinct code paths; not interchangeable.
- **`hls.destroy()` MUST be called before reassigning the ref on channel switch or unmount.** No early-return guards that bypass teardown. Leaked instances exhaust the `<video>` `src` slot budget and cause silent black-screen regressions after a few channel switches.

### Authentication
- **WebAuthn (per auth pivot v2). NOT invoked on viewing path.** Only on contribution actions (PASS, RECOMMEND) — and those are OUT of scope for this epic.

### Critical Version Pins (do not upgrade independently)
- **`@opennextjs/cloudflare`** — pin to a version that explicitly supports Next.js 16.x in its changelog. Adapter does not track Next minor releases in lockstep. Verify before upgrading either package.
- **`@neondatabase/serverless`** (HTTP driver).
- **`hls.js` >= 1.5** (for seek reliability, not LL-HLS).
- **Drizzle** — `drizzle-orm/pg-core` and `drizzle-orm/neon-http`. No other dialects, no other adapters in deployed code.

### Engineering Culture Rules (encoded)
- **CI flakiness caused by real-time clock dependency is a failing defect, not a retry.** If a schedule-math test flakes because `Date.now()` was used instead of an injected clock, fix the clock injection; do not add retry logic. (See `currentProgram(now)` signature requirement in Category 4 / 7.)

### Legacy / Being Superseded (do NOT extend)
- `index.html` / `dual.html` — current Mux SRT live-streaming POC.
- `mediamtx/` — RTMP/SRT server + Cloudflare tunnel config. **The `mediamtx/` directory is dead infrastructure. Its `.yml` files reference Cloudflare tunnel credentials that are NOT the Workers/R2 deployment model. Do not read for any Cloudflare networking pattern.**
- `current_POC_stream.md` — gitignored personal notes; out of scope.
- `@mux/mux-video@0.21` (CDN-loaded in legacy player) — being replaced; do not extend.

### Cross-references (rules that surface here but live primarily elsewhere)
- **Schedule math (`Date.now()` ban, `currentProgram(now)` signature)** → Category 7 (Critical Don't-Miss Rules)
- **Test environment (edge-runtime pool, Neon branches, Playwright for media, Miniflare for KV TTL, clock injection, k6 against deployed Workers)** → Category 4 (Testing Rules)
- **KV TTLs as correctness invariants** → Category 4 (Testing) + future Caching Rules block
- **`POST /api/events` always returns 204** → Category 7 (Critical Don't-Miss Rules)
- **`loop_duration_sec` denormalization invariant** → Category 3 (Framework-Specific / Data Layer Rules)

---

## Critical Implementation Rules

### Language-Specific Rules

**TypeScript strictness**
- `tsconfig.json` MUST set: `"strict": true`, `"noUncheckedIndexedAccess": true`, `"isolatedModules": true`, `"target": "ES2022"`.
- **`verbatimModuleSyntax` is NOT recommended.** OpenNext wraps Next.js's SWC; the dual-pass build has caused breakage on re-export patterns (`export type * from ...`) in edge runtimes. Achieve the same discipline via ESLint `@typescript-eslint/consistent-type-imports` (error level) + `isolatedModules: true`. If `verbatimModuleSyntax` works cleanly for your OpenNext version, you may opt-in — but treat it as build-pipeline-sensitive and smoke-test on every OpenNext bump.

**Trust-boundary validation (Zod at every inbound edge)**
- **All inbound data crossing a trust boundary MUST be validated through a Zod schema before being typed.** Trust boundaries:
  - API route responses (client side: every `fetch` of `/api/...`)
  - Workers KV reads (server side: an entry may be from a previous schema version)
  - LocalStorage reads (client side: `itv.session.v1` may be from a previous app version)
- `JSON.parse(localStorage.getItem(...))` returns `any`. **Never `as SessionV1`** on it. Flow through `SessionSchema.parse(...)`.
- **`satisfies` over `as` for narrowing return shapes.** `as` compiles even when a field is missing; `satisfies` validates. `as` is banned for narrowing API response objects.

**Module boundaries (RSC / Client / Edge)**
- **`"use client"` and `"use server"` directives MUST be the first non-comment line of the file**, before all imports. Placement after imports silently breaks the boundary.
- **Detection rule:** any file referencing `window`, `document`, `navigator`, `localStorage`, `sessionStorage`, or `addEventListener` MUST have `"use client"`. The RSC compiler errors at *build*, not at lint — this rule prevents that build failure.
- **`import type { ... }` is required for type-only imports** crossing client/server boundaries. ESLint `consistent-type-imports` enforces it.
- **Named exports throughout.** Default exports are reserved for Next.js App Router special files only: `page.tsx`, `layout.tsx`, `loading.tsx`, `error.tsx`, `not-found.tsx`, `template.tsx`, `default.tsx`.

**Timestamps, units, and serialization contract**
- **API payloads use millisecond-number timestamps (`*Ms` suffix), not `Date` objects.** Drizzle `timestamp` columns return `Date`; route handlers MUST `.getTime()` them before responding. AI agents forget where "the boundary" is — convert as soon as the value leaves Drizzle.
- **Time arithmetic uses raw `number` math against `Date.now()` and `performance.now()` only.** No `date-fns`, `moment`, `dayjs`, `Temporal` polyfill. `Intl.DateTimeFormat` for display formatting only.
- **`Date.now()` is BANNED in schedule math.** See Category 7 (`currentProgram(nowMs: number)` injectable-clock contract).
- **Bigint serialization:** `playback_events.id` is `bigint`. `JSON.stringify` throws on bigint. Convert with `String(id)` — never `Number(id)` (53-bit precision loss).

**Type discipline for unit confusion**
- **Branded time-unit types ARE REQUIRED.** The codebase mixes UTC epoch ms (server anchor), local-clock ms, and video-element seconds. Distinct types are mandatory:
  ```ts
  type EpochMs = number & { readonly __brand: 'EpochMs' };
  type VideoSeconds = number & { readonly __brand: 'VideoSeconds' };
  ```
  Without these, the multiply-by-1000 error is a silent bug class. Conversion sites become named and auditable.
- **ID branding (`ChannelId`, `AssetId`, `SessionId`) is explicitly deferred.** Use `string` for IDs throughout. Revisit post-POC if an ID-confusion bug surfaces. This deferral is intentional, not an omission.

**Async discipline at the edge**
- **No floating Promises.** Every async call is either `await`-ed or passed to `ctx.waitUntil()`. Workers terminate the isolate when the response Promise resolves; floating Promises (e.g., `void promise.then(...)`) are killed mid-flight.
- **Use `ctx.waitUntil(promise)` (via OpenNext's platform context) for fire-and-forget work** — the `/api/events` insert is the canonical case.
- **`async`/`await` only — no `.then()` chains.** Mixed style produces footguns at the await boundary.
- **`Promise.all` for independent reads, sequential `await` for dependent reads.** Series-awaiting independent DB + KV calls is a measurable p95 regression at 500-concurrent.
- **No top-level `await` at module scope** outside explicit startup files (`instrumentation.ts` or similar). A top-level await in a hot-path module blocks Worker init on every cold start.

**Error handling at the edge**
- **`unknown` in catch blocks.** No `any`. TypeScript defaults catches to `unknown` since 4.4 — do not relax.
- **Narrow errors through dedicated helpers** (`isNetworkError(e)`, `isCacheError(e)`, `isAbortError(e)`) so error-path logic is independently unit-testable. No ad-hoc `e.code === '...'` checks scattered through routes.
- **No try/catch added solely to silence lints.** Only legitimate swallow point is `POST /api/events` per pivot §9.5.
- **Catch at the route boundary, not inside helpers.** Helpers throw; the route handler decides the HTTP shape.

**State shape**
- **Discriminated unions for state with mutually-exclusive modes**, not optional fields with runtime checks. Example: `type PlayerMode = { kind: 'live'; ... } | { kind: 'vod-detour'; ... } | { kind: 'returning'; ... }`. Boolean flags (`isVod`, `isReturning`) are banned — they admit invalid combinations the compiler can't catch. Detailed shape lives in Category 3.
- **API response types use `field: T | undefined`, not `field?: T`**, for fields that can be absent. The `?:` form allows the key to be missing entirely (real scenario for stale KV cache hits from older schema versions); `| undefined` forces exhaustive narrowing.

**React + DOM specifics**
- **Every `useEffect` that constructs an `Hls` instance MUST return `() => hls.destroy()`** cleanup. Strict-mode double-mount and channel-switch both leak the instance otherwise. (Restated from Category 1 in the React-hook context — distinct anti-pattern.)
- **`encodeURIComponent` for all dynamic path segments in client-side `fetch` calls.** `fetch(\`/api/channels/${slug}/manifest\`)` is fine for `ch-cinema` but silently 404s for any future slug with special chars.

**Bundle hygiene (Workers size constraints)**
- **No `lodash`, `ramda`, `moment`, `date-fns`, `dayjs`, `axios` in deployed code.** Workers bundles have a 1 MB compressed limit (Free tier) / 10 MB (Paid). Native JS + `fetch` cover every need.
- **No polyfills.** Workers runtime is modern; target ES2022.
- **`structuredClone` for KV payloads: BANNED.** Use `JSON.stringify` / `JSON.parse` — `structuredClone` is not guaranteed across all OpenNext serialization paths.

### Framework-Specific Rules

#### Next.js 16 App Router patterns

**Route structure (per pivot §10.1, updated by D14 resolution 2026-05-12)**
- The viewer surface lives under a route group: `app/(tv)/`. The route group is invisible in the URL.
- Canonical routes:
  - `/` → `app/(tv)/page.tsx` — resolves last/default channel, renders Player + Guide, hosts the Resume Modal.
  - `/[slug]` → `app/(tv)/[slug]/page.tsx` — direct channel URL, **apex path, no `/watch/` prefix.** Channel slugs are `channel-1`, `channel-2`, ..., `channel-5`.
- **Slug validation is mandatory in `[slug]/page.tsx`** to prevent the apex `[slug]` route from catching non-channel paths. Validate against the channels table at request time; return `notFound()` if no match.
- Special files used: `layout.tsx` (provider tree), `loading.tsx` (brand frame skeleton), `error.tsx` (full-screen "Stand by" + auto-retry), `not-found.tsx` (channel slug 404 + any non-channel apex hit).
- **Resume modal mounts on `/` ONLY**, per pivot §10.6. `/[slug]` MUST NOT show it. Putting the modal in `layout.tsx` violates this.
- Parallel routes, intercepting routes, route handlers under non-`/api` paths: NOT used.

**URL = state contract**
- `interdependent.tv/channel-1` is the canonical, shareable, refreshable URL for a channel. Opening it from a new tab MUST work identically to switching to it from the guide.
- Channel switches use Next.js `router.push('/${slug}')`. NOT a full reload. NOT `window.location.assign`.
- Editorial channel names live in `channels.name` (e.g., "INTERDEPENDENT Cinema"), NOT in the slug. The slug is `channel-N` and remains stable; the editorial name can change.

**Route handler shape (under `app/api/`)**
- Every handler exports `export const runtime = 'edge'` (Cat 1 rule).
- Handler body:
  1. Validate request params with Zod (Cat 2 trust-boundary rule).
  2. Read-through KV cache via `cached(key, tag, ttl, () => fetchFromDB())` helper.
  3. Return `Response.json(payload, { headers: { 'Cache-Control': '...' } })` with explicit `Cache-Control` per pivot §12.1.
- **NEVER** return a bare `NextResponse.json` without explicit Cache-Control. CDN caching is part of the correctness contract, not a perf decoration.

#### Server Components vs Client Components

- **Default is Server Component.** Mark with `"use client"` only when the component reads DOM (`window`, `localStorage`), uses state/effects, or wraps a third-party client lib (hls.js, mux-video, etc.).
- **Provider components (`ServerTimeProvider`, `SessionProvider`, any context provider) are client components.** They live in `app/(tv)/_components/providers/` and are imported into `layout.tsx`.
- **Props crossing the RSC → Client boundary MUST be plain serializable values.** No functions, no `Date` objects, no `Map`/`Set`, no Drizzle row objects with embedded `Date`. Convert to `*Ms` numbers + plain objects on the server side.
- **`async` Server Components are encouraged for initial data fetches** (e.g., `app/(tv)/page.tsx` may `await` `/api/channels` server-side and hand the result down as a serializable prop, eliminating a client waterfall). Keep the same data flowing to the client provider so it can re-fetch on revalidation.

#### Drizzle query patterns

**Schema location**
- `db/schema.ts` at the repo root (or `apps/tv/db/schema.ts` if Nx is multi-app). Single source of truth.
- Drizzle table builders use `drizzle-orm/pg-core` only (Cat 1).

**Repository layer**
- Query functions live in `db/queries/<entity>.ts` — one function per use case, named for the use case, NOT for the SQL (`getPublishedChannelsWithCurrentNext()`, not `selectChannelsJoinAssets()`).
- Each query returns plain serializable objects (timestamps as `*Ms` numbers, bigints as strings — see Cat 2).
- Route handlers MUST import query functions, NEVER `db` directly. Routes orchestrate; queries execute.

**Denormalization invariant (pivot §8.3)**
- `channels.loop_duration_sec` is a denormalized cache of `SUM(asset.duration_sec)` over the channel's `channel_assets` rows.
- **Any code path that mutates `channel_assets` MUST recompute and write `loop_duration_sec` back to the `channels` row in the same logical operation.** This is app-code-enforced (no DB constraint in the POC). Forgetting it silently breaks the schedule math for every viewer until the row is manually corrected.
- Recommended pattern: wrap playlist-edit operations in a helper (`updateChannelPlaylist(channelId, items)`) that handles both writes atomically. Direct `db.update(channelAssets)` outside this helper is banned.

**Migration flow**
- `pnpm drizzle-kit generate` to author a migration; `pnpm drizzle-kit migrate` to apply (Cat 1).
- Seed script is a separate `pnpm seed` Node script (runs OUTSIDE the Worker; can use `pg` or anything Node-native). Seed asserts pivot §8.3 invariants before exit.

#### Workers KV access patterns

**Helper: `cached(key, tag, ttl, loader)`**
- All read-through caching MUST flow through one helper. Inline `env.KV.get(...)` + JSON.parse + fallback-to-DB scattered through route handlers is banned. The helper enforces:
  - JSON serialization symmetry (Cat 2 `structuredClone` ban).
  - Tag-keyed index for invalidation (writes `tag:<tag> → [keys...]` set on every put).
  - TTL is mandatory — no infinite-TTL caches in the POC.
- TTLs by endpoint (pivot §12.1, restated for hard reference):

  | Endpoint | TTL |
  |---|---|
  | `/api/time` | 1s |
  | `/api/channels` | 30s |
  | `/api/channels/[slug]/manifest` | 5 min |
  | `/api/channels/[slug]/schedule?...&toMs=now+` | 60s |
  | `/api/channels/[slug]/schedule?fromMs=future` | 1h |

**Tag-based purge**
- Schedule edit flow: app code (1) writes new `channel_assets`, (2) recomputes `loop_duration_sec` (above invariant), (3) calls `purgeTag('channel:${slug}')` + `purgeTag('channels:list')`.
- Purge propagation across the KV global edge is **eventually consistent (~60s)**. Acceptable per pivot §12.3. Documented, not bug.

#### Hook contracts (project-specific)

These hooks are first-class architectural primitives, not utilities. Their contracts are part of the public interface of the codebase.

**`useServerTime()`** (pivot §10.2)
- Returns `{ getServerNow(): EpochMs, ready: boolean, offsetMs: number }`.
- On mount: single `fetch('/api/time')`, sets `offsetMs = serverTsMs - Date.now()`.
- Subsequent reads: `Date.now() + offsetMs`.
- Re-syncs every 5 minutes in the background.
- **On `visibilitychange → visible` after >60s hidden, re-sync immediately.** (Browsers throttle background tabs; offset may drift while hidden.)
- On detected drift >500ms (computed via a re-sync), warn to `console.warn` and accept the new offset.

**`useSession()`** (pivot §10.3)
- Reads `itv.session.v1` from LocalStorage on mount (single read).
- Exposes `{ session, updateSession, clearSession }`.
- **`updateSession` writes are throttled to 1 per 2 seconds.**
- **Final write on `visibilitychange → hidden` AND `beforeunload`** (some browsers fire only one). Both listeners required; not a choice.
- Schema-validate every read with Zod (Cat 2 trust-boundary rule). If validation fails, treat as cold entry — do not crash.

**`usePlaybackPosition({ manifest, getServerNow })`**
- Pure function of inputs; no internal time source.
- Returns `{ currentAsset, offsetSec, playlistIndex }`.
- **Schedule math uses `((elapsed % L) + L) % L` (double-mod), per pivot §7.2.** Single-mod is wrong for clock-skew or future anchors. Encoded in Category 7.

#### Player module contract (pivot §10.4)

- **Two distinct code paths.** Detect via `'canPlayType' in HTMLVideoElement.prototype && video.canPlayType('application/vnd.apple.mpegurl')` for native HLS support.
- **`PlayerMode` discriminated union** (Cat 2):
  ```ts
  type PlayerMode =
    | { kind: 'live'; channelSlug: string }
    | { kind: 'vod-detour'; assetId: string; resumeMs: number; returnToSlug: string }
    | { kind: 'returning'; channelSlug: string; countdownMs: number };
  ```
- **On `ended` event in `live` mode:** advance to `playlist[(index + 1) % playlist.length]`, set `currentTime = 0`, play. Skip manifest re-fetch (already in memory).
- **Manifest background refresh every 60s.** If the asset that *should* be playing has changed (schedule edit), gracefully switch (tear down hls.js, reload, seek).
- **Error retry policy:** 3 attempts with exponential backoff (1s / 2s / 4s). After 3 failures, render `<PlayerError>` with "Try another channel" CTA. No silent black screens.

#### Provider tree composition

- `app/(tv)/layout.tsx` mounts `<ServerTimeProvider>` ONCE wrapped around `<SessionProvider>` wrapped around `{children}`. Providers do NOT re-mount per route.
- Provider files live in `app/(tv)/_components/providers/`.
- **Server-rendered initial state can be passed as a prop to the provider** (e.g., `<ServerTimeProvider initialOffsetMs={...}>`) to eliminate the first-mount fetch waterfall. Optional but recommended.

#### Styling approach

- **Open decision.** Not specified in pivot doc. Surfaced for Category 5 (Code Quality & Style) or §17 open-decisions list. Recommendation: Tailwind for velocity + Workers-bundle compatibility, but flagging not deciding here.

### Testing Rules

#### Test environment matrix (mandatory)

| Test type | Runner / Environment | Why |
|---|---|---|
| API route handlers (`app/api/**`) | Vitest with `@edge-runtime/vm` pool OR `@cloudflare/vitest-pool-workers` (Miniflare) | Edge runtime has no Node globals; jsdom/Node test pool silently passes code that throws on Workers. |
| Pure hook logic (`useServerTime` offset math, `useSession` LocalStorage R/W, `usePlaybackPosition` schedule math) | Vitest with `jsdom` | No media APIs touched; jsdom is sufficient and fast. |
| Player + media lifecycle (`MANIFEST_PARSED`, `ended`, `currentTime` seek, hls.js teardown) | **Playwright** (Vitest browser mode or standalone) | jsdom has no `MediaSource`; `<video>` never fires `timeupdate`/`canplay`/`ended`. jsdom for media tests produces false negatives. |
| Golden user flows (cold load → playback, channel switch, resume modal branches) | Playwright E2E | Real browser; real HLS; real LocalStorage. |
| Load (500 concurrent, ±2s sync) | k6 against deployed Cloudflare Workers preview env | `next dev` does not emulate Workers KV TTL, edge runtime constraints, or cold-start behavior. Loading localhost is meaningless. |

- **Importing any `app/api/**` module in a Node-runtime test (default jsdom pool) is BANNED.** Use the edge-runtime pool.

#### Database testing (Neon)

- **No fetch-layer mocks of `@neondatabase/serverless`.** Mock libraries (`pg-mem`, `jest-pg-mock`) intercept at the `pg.Client` protocol layer that Neon doesn't use; they validate fiction.
- **Each CI run creates an ephemeral Neon branch** via Neon's branch API. The branch's `DATABASE_URL` is injected into the test env. Branch is torn down at workflow end. `DATABASE_URL` in CI points to a branch, NEVER to `main`.
- Local dev: `postgres:16-alpine` container is acceptable; apply schema via `drizzle-kit push` (the only legitimate use of `push` — see Cat 1).
- **Drizzle query tests run against a real Postgres** (branch or container), exercising window functions and JSON aggregation that mocks cannot represent.

#### Clock injection (load-bearing for correctness)

- **`currentProgram(now: number)` signature is mandatory.** The function MUST accept the clock as a parameter; no internal `Date.now()` calls. (Restated from Cat 7 for visibility; this is the rule that makes schedule tests possible at all.)
- Test-side clock control:
  - **Vitest hook/unit tests:** `vi.setSystemTime(new Date('2026-05-12T...'))`.
  - **Playwright E2E:** `page.clock.setFixedTime(...)` and `page.clock.runFor(...)`. Real-time waits (`page.waitForTimeout`) for behavior assertions are banned — they're flaky-by-design.
- **CI flakiness caused by real-time clock dependency is a failing defect, not a retry.** No `retry: 3` configs in `vitest.config.ts` or `playwright.config.ts` to paper over clock issues. Fix the clock injection.

#### Workers KV testing

- **Plain `Map` mocks of KV are BANNED for any test that exercises TTL or tag-purge semantics.** TTL contracts (1s / 30s / 5min / 60s / 1h, pivot §12.1) are correctness invariants for the ±2s sync gate.
- **Use Miniflare's KV implementation** (`@cloudflare/vitest-pool-workers`). It enforces TTL expiry.
- **Tag-purge integration test is mandatory:** write key with tag → fire `purgeTag()` → assert next read is a miss. This test MUST exist before Sprint 4 demo and MUST run in CI.

#### Player + media testing (Playwright)

- **`<ChannelPlayer>` tests assert the seek sequence explicitly:**
  - hls.js path: assert `currentTime` is set AFTER `MANIFEST_PARSED` and BEFORE `play()`.
  - Safari path: assert `currentTime` is set in `loadedmetadata`, BEFORE `play()`.
- **Memory-leak test:** switch channels 10 times in sequence, assert exactly one active `<video>` element, assert no detached HTMLMediaElement instances in the heap (Playwright + Chrome DevTools Protocol).
- **Time-sync test:** open two browser contexts on the same channel, assert their `currentTime` within ±2s after 30s of playback. This is the G3 acceptance gate; it MUST be an automated test, not a manual demo.
- **Resume modal branches:** all five scenarios from pivot §11.2 MUST have Playwright tests. The decision matrix is the spec.

#### Load testing (k6)

- **`tests/load/` directory** holds k6 scripts. First-class CI artifacts, versioned with the code.
- **k6 target MUST be a deployed Cloudflare Workers preview environment** (`wrangler deploy --env staging` or Pages preview branch). NEVER localhost or `next dev`.
- **Warm-up: one request per channel before ramp** to populate KV. Skipping warm-up produces meaningless first-30s metrics dominated by KV-miss → DB latency.
- **Acceptance gate (pivot §3 G3 + Sprint 4 §13):**
  - 500 concurrent sessions sustained for 30 min.
  - p95 cold-start ≤ 2.5s.
  - KV hit rate ≥ 98% after warm-up.
  - Postgres CPU < 30%.
- A Sprint 4 demo is NOT declared complete until k6 passes this gate against a deployed preview.

#### Test data factories

- **`db/queries/test-factories.ts` (or `db/test/factories.ts`)** generates test channels with known epoch anchors and known playlist durations. Pure functions; deterministic output.
- **Factories MUST assert pivot §8.3 invariants before returning:** `loop_duration_sec === SUM(asset.duration_sec)`, no zero-duration assets, non-null `hls_url`. Tests built on invalid factories are worse than no tests.
- **No shared mutable state across tests.** Each test gets its own factory-built fixture. `beforeAll` writes that bleed across tests are banned.

#### Coverage philosophy

- **No blanket coverage threshold.** Risk-based.
- **Coverage REQUIRED:**
  - Schedule math (`currentProgram`, double-mod, all §7.4 edge cases). 100%.
  - Resume modal decision matrix (pivot §11.2). One test per row of the matrix.
  - `loop_duration_sec` denormalization invariant on every playlist-edit code path.
  - `POST /api/events` returns 204 on Postgres failure (the swallow contract).
  - All five §3 acceptance gates (G1–G5).
- **Coverage NOT REQUIRED:** trivial Server Components, layout files, pure render-only chrome (icons, badges).

#### Test discipline

- **Test files colocate with code: `Foo.tsx` + `Foo.test.tsx` in the same directory.** Exception: Playwright E2E lives under `tests/e2e/`.
- **One assertion per behavioral expectation.** Multi-assert tests where the first failure masks the second are banned.
- **AAA structure** (Arrange / Act / Assert) with blank-line separators. Helpers extracted into the same file's `describe.helpers` block, not a shared utils import.
- **No mocking of internal modules.** If a unit needs mocking to test, the unit is too coupled — refactor. Mock only at the network/HLS/IO boundary.

#### Deferred testing patterns

- **Pact / consumer-driven contract testing:** deferred. Single consumer (the .tv frontend), single producer (the .tv API routes), same monorepo — Pact's value lands when consumer and producer ship independently. Revisit when `.studio` or `BACKSTAGE` consumes these APIs.
- **Snapshot tests:** limited use. Acceptable for stable serialized outputs (e.g., channel manifest JSON shape against a versioned fixture). Banned for React component rendered output — too brittle.
- **Mutation testing (Stryker):** not in POC scope.

### Code Quality & Style Rules

#### File and folder organization

**Per pivot §10.1, enforced:**

```
app/
├── (tv)/                       # route group; invisible in URL
│   ├── layout.tsx              # providers
│   ├── page.tsx                # /  (default channel resolution + resume modal)
│   ├── watch/[slug]/page.tsx   # /watch/[slug]
│   └── _components/            # tv-surface-only components
│       ├── ChannelPlayer/
│       │   ├── index.tsx       # the component
│       │   ├── usePlaybackPosition.ts
│       │   └── PlayerChrome.tsx
│       ├── ChannelGuide/
│       ├── ResumeModal/
│       └── providers/
└── api/
    ├── time/route.ts
    ├── channels/route.ts
    ├── channels/[slug]/manifest/route.ts
    ├── channels/[slug]/schedule/route.ts
    └── events/route.ts

db/
├── schema.ts
├── queries/<entity>.ts
└── test/factories.ts
```

- **Component folders use `index.tsx` for the public component**, with sub-pieces as siblings. Reach-into imports (`import { Foo } from '@/.../ChannelPlayer/PlayerChrome'`) are allowed for tightly-coupled siblings; cross-tree reach-ins are not.
- **`_components/` (underscore prefix) is Next.js's convention for "not a route segment."** Don't use it for shared code that other routes need — that goes in `components/` at the repo root.

#### Path aliases

- `@/` → repo root (`/`) — i.e., `@/db/schema`, `@/app/(tv)/_components/...`.
- **No deeper aliases** (`@/db`, `@/api`, `@/lib` as separate aliases). Single `@/` keeps imports greppable. Multiple aliases produce inconsistency under AI generation.
- Configured in `tsconfig.json` `compilerOptions.paths` and mirrored in `next.config.ts`. Both must agree.

#### Naming conventions

| Construct | Convention | Example |
|---|---|---|
| React components | PascalCase | `ChannelPlayer`, `ResumeModal` |
| Hooks | camelCase, `use` prefix | `useServerTime`, `usePlaybackPosition` |
| Files (components) | PascalCase | `ChannelPlayer.tsx`, `index.tsx` |
| Files (hooks, utils, queries) | camelCase | `usePlaybackPosition.ts`, `getChannels.ts` |
| Files (route segments) | Next.js convention | `page.tsx`, `layout.tsx`, `route.ts` |
| Route URL segments | kebab-case (channels follow `channel-N` literal pattern) | `/channel-1`, `/api/channels` |
| Database tables / columns | snake_case | `channel_assets`, `loop_duration_sec` |
| TypeScript types / interfaces | PascalCase | `ChannelManifest`, `PlayerMode` |
| Constants | UPPER_SNAKE | `DEFAULT_RETRY_LIMIT`, `SESSION_KEY` |
| Zod schemas | PascalCase + `Schema` suffix | `ChannelManifestSchema`, `SessionSchema` |

- **Schema slug values follow kebab-case** (`ch-cinema`, `ch-backstage`), matching URL conventions. Migrations enforce.

#### Comments

- **Default to writing no comments.** Well-named identifiers are the documentation.
- **Add a comment ONLY when the WHY is non-obvious** — a hidden constraint, a subtle invariant, a workaround for a specific bug, behavior that would surprise a reader.
- **DO NOT explain WHAT the code does** (`// loop over channels` next to a `for` loop). Named identifiers do this.
- **DO NOT reference the current task, fix, or callers** (`// used by ChannelGuide`, `// added for resume modal`, `// fix for #123`). PR descriptions hold this context; comments rot.
- One-line comments only. No multi-paragraph docstrings. JSDoc is acceptable on PUBLIC types where TS inference doesn't surface intent (e.g., the discriminator field of `PlayerMode`).
- **Exception:** Drizzle schema columns MAY have a one-line `//` comment when the column name doesn't fully convey semantics (`license_state` — 'owned' | 'licensed' | 'placeholder').

#### Emojis

- **No emojis in code, comments, identifiers, or commit messages.** Project-wide rule. (UI strings rendered to users are out of scope here — those are an editorial decision.)

#### No time estimates

- **No time estimates in stories, ADRs, project-context, or any planning artifact.** BMAD convention; AI-paced development invalidates human-pace estimates.
- Stories use S / M / L effort labels only (pivot §13 already follows this).
- Sprint scope is bounded by acceptance criteria, not by hours.

#### Editing discipline

- **Prefer editing existing files to creating new ones.**
- **Do NOT create documentation files (`*.md`, `README.md`) unless explicitly requested.** The pivot doc + this file + BMAD-generated planning artifacts cover the spec.
- **Do NOT create planning, decision, or analysis documents unless explicitly requested.** Work from conversation context, not intermediate scratch files.

#### Linter & formatter

- **ESLint flat config (`eslint.config.mjs`)** — Next.js 16+ uses flat config.
- Required rule sets:
  - `@typescript-eslint/recommended-type-checked`
  - `eslint-plugin-react/recommended`
  - `eslint-plugin-react-hooks` (exhaustive-deps as ERROR, not warn)
  - `@typescript-eslint/consistent-type-imports` (error)
  - `@typescript-eslint/no-floating-promises` (error)
  - `@typescript-eslint/no-explicit-any` (error)
  - `eslint-plugin-drizzle` (if available for your Drizzle version; safety against accidental `db.delete()` without `where`)
- **Prettier** with project default + `printWidth: 100`. No tabs. 2-space indent.
- **CI gate:** `pnpm lint && pnpm typecheck` MUST pass before any PR merges. No `--no-verify` commits.

#### Styling approach

- **Tailwind CSS 4** (utility-first; small bundle; Workers-friendly).
- **Global stylesheet** for tokens (font, color variables, the velvet-rope brand palette) — kept lean, ≤ 100 lines.
- **CSS-in-JS libraries (styled-components, Emotion) are BANNED** in deployed code. They add bundle weight and RSC compatibility friction.
- Component-level styles are Tailwind class strings. Conditional classes via `clsx` (lightweight; ~200 bytes).
- Tailwind config lives at `tailwind.config.ts`. The brand palette + custom fonts are configured there, not redefined per component.

#### Import ordering

Enforced by Prettier or `eslint-plugin-import`:
1. Node built-ins (rare, but `node:crypto` etc. if ever needed at build time)
2. External packages (`react`, `next`, `drizzle-orm`, `hls.js`, etc.)
3. `@/` aliased imports
4. Relative imports (`./`, `../`)
5. Type-only imports (`import type { ... }`)
6. CSS / static asset imports

#### Commit messages

- Conventional Commits style: `<type>(<scope>): <subject>`. Types: `feat`, `fix`, `chore`, `refactor`, `docs`, `test`, `style`, `build`, `ci`, `perf`. Scopes: route name, component name, or `db`, `infra`, `tests`.
- Subject ≤ 70 chars, imperative mood, lowercase.
- Body explains WHY, not WHAT. Reference ADRs or pivot sections (`per pivot §7.2`) when relevant.
- **No Co-Authored-By unless the user explicitly asks for it.**

### Development Workflow Rules

#### BMAD agent roles (who does what)

This project is authored under BMAD conventions. AI agents working in this repo MUST understand role boundaries:

| Code | Agent | Role | Triggered when |
|---|---|---|---|
| Mary | Business Analyst | Requirements discovery, evidence-based findings | Pre-PRD, market/domain research |
| John | Product Manager | PRD authorship, story prioritization | PRD edits, story acceptance |
| Sally | UX Designer | UX specs, decision matrices | Resume modal copy, error states |
| Winston | System Architect | ADRs, stack decisions, edge-runtime constraints | Architecture changes, stack edits |
| Amelia | Senior Software Engineer | Implementation, test-first | Story execution |
| Murat | Test Architect | NFR assessment, test strategy, CI design | Risk-based test design, load test gates |
| Paige | Technical Writer | Documentation discipline | When docs change |
| Sophia | Storyteller | Narrative, demo scripts | Sprint 4 demo doc |

- The pivot doc (`tv_poc_plan.md`) is the canonical source of truth for Epic TV-1. ALL agents read it before acting on this surface.
- Sprint story files are authored by the SM (Scrum Master) workflow from BMAD, AFTER the §17 open decisions are resolved (see below).

#### ADR discipline

- **ADRs are immutable once accepted.** Edits to a "Proposed" ADR are fine; once status is "Accepted," changes require a NEW ADR that supersedes the old one. The new ADR explicitly states "Supersedes ADR-XXX" and the old ADR is updated to "Superseded by ADR-YYY" — but its content is preserved verbatim.
- Pivot §5 lists ADR-001 through ADR-007. All are status "Proposed" pending §17 decisions.
- ADRs live in `docs/adr/` (one file per ADR, numbered, immutable filename) once we exit the pivot-doc-as-spec phase.
- **AI agents do NOT silently retract or modify an ADR.** Surface the conflict; propose a superseding ADR; wait for human sign-off.

#### §17 Decision Gate (blocking Sprint 1)

Per pivot §17, seven decisions blocked Sprint 1. Status after 2026-05-12 party-mode resolution (Chris as casting vote, panel: John, Mary, Winston, Victor):

| # | Decision | Resolution | Status | Owner |
|---|---|---|---|---|
| 17.1 | Simulated live vs. true live | **Simulated** (ADR-001 → Accepted) | Closed | Vik / PM |
| 17.2 | Anonymous viewing vs. WebAuthn gating | **Anonymous on viewing path** (ADR-005 → Accepted); WebAuthn only on contribution actions | Closed | PM + auth-pivot v2 |
| 17.3 | POC content source | **(a) Owned masters required.** Sprint 1 cannot start until Vik confirms ≥5 cleared, encodable assets in writing. If <5, options: delay Sprint 1 OR re-scope to 3 channels. Public-domain placeholder is NOT acceptable demo content. | **Gated — pending asset inventory** | Vik |
| 17.4 | VOD detour vs. live-only on resume | **VOD detour** (ADR-006 → Accepted). Editorially named "Catch up" or similar in player chrome. | Closed | Vik |
| 17.5 | Ads in POC | **None.** Analytics events designed for creator-economy model (completion rate, channel loyalty, per-creator retention), NOT for ad-targeting. | Closed | PM |
| 17.6 | Channel guide scope | **Internal only — 5 INTERDEPENDENT channels.** Federated creator channel from THE LOT considered and deferred (Victor's flip rejected; revisit post-POC). | Closed | PM |
| 17.7 | Geographic / geo-fencing | **None for POC.** Cross-cutting addendum: **add `countryCode` (from `CF-IPCountry` Workers header) to `playback_events` payload** — geo-aware analytics is NOT geo-fencing; capture the data now. | Closed | PM / legal |

- **Six of seven decisions Closed.** 17.3 is **Gated** pending Vik's asset inventory confirmation. Sprint 1 cannot kick off while 17.3 is Gated.
- **ADR status updates:** ADR-001, ADR-005, ADR-006 move to "Accepted, 2026-05-12." Remaining proposed ADRs progress with their own decisions.
- **Schema additions accepted from party-mode cross-cutting items (file as Sprint 1 prep):**
  - `channels.channelType` discriminator (`'loop' | 'live'`) — anticipates true-live channel without a future migration.
  - `assets.expiresAt` (nullable timestamp), `assets.territories` (nullable text array) — anticipates licensed content without a future migration.
  - `playback_events.countryCode` (nullable text) — geo-aware analytics per 17.7.
- **Pivot doc supersession:** flag for v1.1 once Vik closes 17.3.

#### Branching strategy

- **Trunk-based with short-lived feature branches.**
- Branch naming: `<type>/<scope>-<short-description>` — examples: `feat/channel-player-seek`, `fix/manifest-cache-purge`, `chore/eslint-flat-config`.
- Branches MUST be ≤ 200 changed lines OR ≤ 3 days from creation to merge — whichever comes first. Long-lived branches drift; AI-paced development makes this drift faster.
- `main` is the only protected branch. Force-push to `main` is BANNED unless the user explicitly authorizes it.
- **Never push to `main` without a PR.** Even single-line typo fixes go through PR review for the audit trail.

#### Commit hygiene

- Conventional Commits (per Cat 5).
- Commits MUST be atomic — one logical change per commit. WIP commits (`wip`, `oops`, `fix typo`) are squashed before PR merge or interactively rebased before PR open.
- **NEVER skip hooks (`--no-verify`, `--no-gpg-sign`)** unless the user explicitly asks for it. If a hook fails, investigate and fix the root cause.
- **NEVER use `git push --force` against `main`.** Force-push to a feature branch IS acceptable when the branch is yours and the rewrite is small (interactive rebase, cleaning history before review).
- **`git add -A` and `git add .` are discouraged.** Stage specific paths to avoid accidentally including secrets or build artifacts.

#### Files that MUST NOT be committed

- `.env`, `.env.local`, `.env.production` — any env file with secrets.
- `mediamtx/cloudflared.log`, `mediamtx/*.plist`, `mediamtx/cloudflared-config.yml` — legacy infra; tunnel credentials live here.
- `current_POC_stream.md` — already gitignored; personal notes.
- `mediamtx.zip` — already gitignored; binary archive.
- Cloudflare `wrangler.toml`-adjacent secret files (`.dev.vars`).
- Neon connection strings in plain text anywhere outside `wrangler.toml` env-var bindings.

`.gitignore` enforces these; agents adding new categories of secrets MUST update `.gitignore` first.

#### Environment variables / Workers bindings

- **`wrangler.toml`** is the source of truth for Workers bindings (KV, R2, env vars).
- Local dev secrets live in `.dev.vars` (gitignored). Staging/prod secrets are set via `wrangler secret put`.
- Required bindings:
  - `KV` — Workers KV namespace.
  - `R2_BUCKET` — R2 bucket binding (if Workers ever proxy R2 reads; for POC, browser fetches direct).
  - `DATABASE_URL` — Neon HTTP connection string (secret).
- **Binding names in `wrangler.toml` MUST match the property names destructured in route handlers** (Cat 1 rule, restated for workflow visibility).

#### Deployment pipeline

The progression is mandatory; no skipping stages:

1. **Local (`pnpm dev`)** — Next.js dev server. Useful for component work; NOT representative of Workers runtime behavior. Do not declare a story done based on `pnpm dev` alone.
2. **Wrangler preview (`pnpm wrangler dev`)** — local Workers emulator with Miniflare. KV TTLs, edge runtime, env bindings all behave correctly. THIS is the local checkpoint for backend work.
3. **Staging preview deploy (`wrangler deploy --env staging`)** — real Workers, real KV, real Neon branch. Where k6 load tests run (Cat 4). Each PR triggers one of these (Pages preview branch, automatic).
4. **Production (`wrangler deploy --env production`)** — `main` branch CI deploys here.

**Migration ordering rule:** any code change that depends on a new column or table MUST be preceded by a deployed migration. The pattern:
- Step 1: Author migration + deploy schema change to staging.
- Step 2: Verify schema on staging.
- Step 3: Merge code that uses the new schema; deploy.

Never combine schema migration + code that depends on it into a single deploy. Roll-forward is acceptable; roll-back is not (no destructive `down` migrations in POC — write a compensating forward migration if you need to back out).

#### PR review

- Every PR has at least one reviewer (human or `/ultrareview`-style automated). Self-merge banned.
- PR template includes:
  - Summary (≤ 3 bullets).
  - Pivot doc reference (`Implements pivot §X.Y` or `Addresses ADR-NNN`).
  - Test plan: bulleted checklist of how to verify locally + on staging.
  - Risk flags: any §16 risk this change touches or mitigates.
- **No squash-merges by default** — preserve commit history for `git blame` archaeology. Exceptions: branches with WIP-style commits that should be flattened.
- **AI-authored PRs are reviewed with the same rigor as human PRs.** Co-Authored-By trailers MUST NOT lower the review bar.

#### Story workflow

- Stories are generated by the BMAD SM workflow from the epic file + accepted ADRs (after §17).
- Story files live in `_bmad-output/planning-artifacts/sprint-N/story-N.M.md`.
- Story status flow: Draft → Ready → In Progress → Review → Done.
- A story is **Done** only when ALL of: AC met, tests green, lint+typecheck green, deployed to staging, PR merged. Local-only "done" is not done.

#### Risk register maintenance

- Pivot §16 lists R1–R10. Status updates flow into the same register; new risks discovered during implementation get appended (R11+) with the same shape.
- Risks at status "Open / High" block sprint exit. Epic TV-1 DoD requires no Open/High risks at Sprint 4 exit.

#### Documentation discipline

- **The pivot doc + this project-context.md + ADRs + story files are the canonical body of project documentation.** Do not create parallel docs that restate the same content.
- README.md is one paragraph: what the project is, where to start, where the spec lives. No deep documentation lives in README.
- **Demo doc (pivot §13 Sprint 4 Story 4.6) is the only narrative doc** allowed for external stakeholders. Authored by Sophia in collaboration with Paige.

### Critical Don't-Miss Rules

> **If you forget the rest of this document, remember these.** These rules either repeat the most load-bearing items from prior categories (intentional redundancy — they're worth saying twice) or capture mental-model traps that the architecture's success depends on.

#### 1. "Live" is simulated, NOT true live

- This codebase serves **simulated linear TV from VOD assets on a deterministic UTC-anchored loop.** No RTMP ingest. No live encode. No LL-HLS.
- Every viewer's browser independently computes the same `(currentAsset, offsetSec)` from `(server_now, epochAnchor, playlist)`. No server-side coordination per viewer.
- Glossary at pivot §19 is the spec. The word "live" in this codebase is shorthand for the simulated mechanism unless explicitly prefixed *true live*.
- True-live encoding is **post-POC** (pivot §14.4). If an agent reaches for RTMP, MediaMTX, Mux Live, or hls.js LL-HLS features, push back. Those are scale-sprint work, not now.

#### 2. Schedule math is sacred — and is a pure function

- The formula is exact: `positionInLoop = ((elapsedSec % L) + L) % L` (double-mod). Single-mod is wrong for clock-skew or future anchors.
- `currentProgram(now: number, channel: Channel)` MUST accept the clock as a parameter. **`Date.now()` is BANNED inside the function.** Without this signature, schedule tests are flaky-by-design.
- Time units do not mix: `epochAnchorMs` is milliseconds, `loopDurationSec` and `offsetSec` are seconds, `video.currentTime` is seconds. The unit-confusion bug class is real — see the branded types in Cat 2 (`EpochMs`, `VideoSeconds`).
- **UTC anchor only.** Never localize. Display formatting in the viewer's local timezone happens at the render boundary, never inside the math.

#### 3. Anonymous viewing on entry — NO auth on the viewing path

- Per pivot ADR-005 and PRD NFR-03: no passwords, no OTP, no magic links, no WebAuthn challenge on the way to playback.
- ENTER → straight into the player. No modal. No interstitial.
- WebAuthn (per auth pivot v2) is invoked ONLY on contribution actions (PASS, RECOMMEND, save preferences). Those are OUT of scope for Epic TV-1.
- If an agent adds an auth check to `/`, `/watch/[slug]`, or any API route under `/api/channels` or `/api/time`, that is a violation. The only auth-aware route in the foreseeable horizon is `/api/channels` (when Taste Score lands post-POC, per pivot §14.6).

#### 4. LocalStorage is the only viewer state — server is stateless per-viewer

- `itv.session.v1` is the single LocalStorage key. JSON blob, versioned with `v: 1`, Zod-validated on read.
- The server stores **nothing** per viewer. No cookies. No session tokens. No server-side LocalStorage equivalent.
- `/api/events` writes analytics but is fire-and-forget (see rule #6). Analytics are NOT viewer state.
- Cross-device resume is **impossible without auth** and is explicitly deferred. Do not attempt to synthesize it.
- LocalStorage cleared → graceful degradation to cold entry. Not an error.

#### 5. The autoplay constraint — viewer click is the unmute trigger

- Browser autoplay policies BLOCK silent autoplay unless the video starts `muted`. Pivot risk R10 (Closed) confirms.
- The player MUST start with `muted` attribute on the `<video>` element. The first viewer click anywhere (or the canonical "tap to unmute" zone) unmutes.
- Do not chase this with workarounds. Do not request user-gesture exceptions. Do not log it as a bug. It is the contract.

#### 6. `POST /api/events` returns 204 — ALWAYS

- Pivot §9.5 contract is explicit: "Always. Even on internal failure."
- Postgres write failures MUST be caught, logged to `console.error`, and swallowed. NEVER bubble to the client.
- An agent writing a naive `await db.insert(...)` without try/catch produces a 500 to the client on any analytics failure, which BLOCKS PLAYBACK because the client fires `start` events at playback init. This is the single most likely bug to crash the demo.
- The route handler returns `new Response(null, { status: 204 })`. No body. No error object. No retry hint.

#### 7. `loop_duration_sec` denormalization invariant (pivot §8.3)

- `channels.loop_duration_sec` is a denormalized cache of `SUM(asset.duration_sec)` for that channel's `channel_assets` rows.
- **ANY code path that mutates `channel_assets` MUST recompute and write `loop_duration_sec` back to the channel row in the same logical operation.**
- App-code enforced (no DB constraint in POC). Forgetting silently breaks the schedule math for EVERY viewer until manually corrected.
- Wrap playlist edits in `updateChannelPlaylist(channelId, items)`. Direct `db.update(channelAssets)` outside this helper is banned.

#### 8. HLS segments fetch directly from R2 — CORS is mandatory

- Browser fetches segments directly from R2; **Workers do NOT proxy them**.
- **R2 bucket MUST have a CORS policy allowing `interdependent.tv` as origin.** Missing CORS causes every segment 404 in Chrome/Firefox. Safari is more permissive — testing only on Safari produces FALSE POSITIVES.
- This rule has no compile-time signal and no local repro on `pnpm dev`. It bites at Sprint 1 Story 1.4. Verify CORS before declaring that story done.

#### 9. `export const runtime = 'edge'` on EVERY API route

- Omission silently falls back to Node-compatible mode locally, then fails at Workers deploy time.
- `/api/time` MUST also export `export const dynamic = 'force-dynamic'` to prevent build-time static rendering of `Date.now()`.
- Cat 1 rule, restated because this is the single most common Next.js 16 + OpenNext mistake.

#### 10. Resume modal decision matrix is the spec (pivot §11.2)

- The matrix has five rows. Each row maps a session state to an exact (copy, button-1, button-2) tuple. **One Playwright test per row.**
- The matrix is exhaustive. If a session state doesn't match any row, fall through to "Welcome back" + "Browse channels" single-CTA (row 5). Do not invent new states.
- Modal mounts on `/` ONLY (pivot §10.6). Mounting it in `layout.tsx` violates the contract because it would show on `/watch/[slug]` too.
- Session age > 24h: skip modal, autoplay live. Not configurable.

#### 11. VOD detour pattern (ADR-006)

- "Continue" plays the asset standalone as VOD with `currentTime = lastPositionSec`. When the asset `ended`s, navigate to `/watch/[returnChannelSlug]` and resume LIVE.
- Player chrome shows "VOD — returning to *Channel* live in M:SS" badge when <30s remain.
- User can exit detour early via "Go live now" CTA.
- "Continue" does NOT mean "rewind the live channel." Rewinding a simulated-live channel is meaningless — every viewer's position is computed from wall-clock time.
- Every channel asset is dual-purpose: live programming + on-demand seekable. The asset stays reachable by ID even after it cycles off the channel.

#### 12. hls.js teardown — destroy before reassign

- Every `useEffect` that constructs `Hls` MUST return `() => hls.destroy()`.
- Every channel switch MUST call `hls.destroy()` on the previous instance BEFORE constructing the new one.
- Leaked hls.js instances exhaust the `<video>` `src` slot budget and produce silent black-screen regressions after a handful of channel switches.
- Memory-leak test is required (Cat 4): switch channels 10 times, assert one active `<video>`, zero detached `HTMLMediaElement` instances.

#### 13. Two player code paths — hls.js vs. native HLS — NEVER interchangeable

- Detect via `video.canPlayType('application/vnd.apple.mpegurl')` — **NOT user-agent sniffing**, which is brittle and bans iPad-Safari-pretending-to-be-Chrome edge cases.
- hls.js seek sequence: `MANIFEST_PARSED` → `currentTime = offsetSec` → `play()`.
- Safari seek sequence: `loadedmetadata` → `currentTime = offsetSec` → `play()`.
- These are distinct lifecycle moments. Code that uses one event with the other path silently fails.

#### 14. Schedule edits propagate ~60s — eventually consistent

- Pivot §15: "Schedule edit visibility — eventually consistent. ≤60s globally."
- This is the documented consistency model, NOT a bug.
- Manifest background refresh is 60s; that's the propagation cadence.
- If an agent tries to make schedule edits propagate instantly (websockets, SSE, etc.), push back. The cost/complexity is not warranted at POC scale.

#### 15. The legacy code is NOT a reference

- `index.html`, `dual.html`, `mediamtx/`, `current_POC_stream.md`, `@mux/mux-video@0.21` — these belong to the live-streaming POC that the Linear Channel Playback POC supersedes.
- Agents scanning for "how does this project work" will find the legacy player and incorrectly conclude patterns from it apply. They do not.
- The `mediamtx/` directory contains Cloudflare TUNNEL credentials — that is NOT the Workers+R2 deployment model. Do not reference it for any Cloudflare networking pattern.

#### 16. Bundle size is a correctness constraint

- Workers bundles: 1 MB compressed (Free tier) / 10 MB (Paid).
- A single `import _ from 'lodash'` pushes a Free-tier deploy past the limit and the deploy 5xx's, not gracefully degrades.
- No `lodash`, `ramda`, `moment`, `date-fns`, `dayjs`, `axios`, `styled-components`, `emotion`. (Cat 2 + Cat 5.)
- Bundle size is checked at deploy time. If a PR's bundle delta is unexpectedly large, that's a review-stop.

#### 17. ±2s time-sync tolerance is an acceptance gate (G3)

- Two viewers on the same channel at the same moment MUST see content within ±2s of each other (pivot §15, §3 G3).
- This is a HARD GATE, not a goal. Sprint 1 exit requires it. Sprint 4 load test re-validates at 500 concurrent.
- Beyond ±2s the simulated-live illusion breaks; viewers notice asynchrony.
- Achieved via segment alignment + `useServerTime` offset re-sync + correct double-mod math. Any one missing breaks the gate.

#### 18. The §17 decision gate (status: 6 of 7 closed; 17.3 gated on Vik)

- **Closed (resolved 2026-05-12):** 17.1 (simulated), 17.2 (anonymous viewing), 17.4 (VOD detour), 17.5 (no ads), 17.6 (internal only — 5 channels), 17.7 (no geo-fencing).
- **Gated:** 17.3 (POC content source) — Sprint 1 cannot kick off until Vik confirms ≥5 cleared, encodable, owned assets in writing. Public-domain placeholder is NOT acceptable demo content per Chris's call.
- Story files for Sprint 1 MAY be drafted in parallel with the 17.3 inventory exercise — but they MUST NOT be marked Ready until Vik's confirmation. Conditional drafts referencing "5 channels at 4–8h loop each" are acceptable provided 17.3 closes before kickoff.
- See Category 6 §17 table for full resolutions and accepted schema additions.

#### 19. ADRs are immutable once accepted

- A "Proposed" ADR is mutable. An "Accepted" ADR is not.
- Changing an accepted ADR requires a new superseding ADR with a clear "Supersedes ADR-NNN" reference. The old ADR's content stays intact.
- AI agents do not silently retract or edit accepted ADRs. They surface the conflict and propose a new ADR.

#### 20. CI flakiness from real-time clocks is a defect, not a retry

- A schedule test that flakes near asset boundaries is a sign the function uses `Date.now()` instead of an injected clock. Fix the injection.
- No `retry: 3` in `vitest.config.ts` or `playwright.config.ts` to paper over clock issues.
- "Re-run the failing test" is not a remediation strategy in this codebase.

#### 21. The pivot doc is the canonical source — RFC 2119 keywords are literal

- `tv_poc_plan.md` v1.0 (or whichever version is current) is the authoritative spec.
- MUST / SHOULD / MAY in that doc are treated per RFC 2119 — literal, normative.
- This project-context.md repeats and amplifies the most load-bearing rules. The pivot doc is the source.
- When the pivot doc and this file appear to conflict, the pivot doc wins UNLESS the conflict is a clarification or implementation detail this file has authority over (e.g., the test-environment matrix in Cat 4).
- If a real conflict emerges, surface it. Do not silently choose.

---

## Usage Guidelines

**For AI Agents (read this first, every time):**

- Read this file before implementing ANY code in this project.
- The pivot document at `/Users/interdependent/Downloads/tv_poc_plan.md` (or wherever it lives in the repo after import) is the canonical spec. This file repeats its load-bearing rules and adds implementation-level constraints.
- When a rule and a request appear to conflict, surface the conflict — do not silently choose. Most "conflicts" are misreadings; a few are genuine and need a human decision.
- When in doubt, prefer the more restrictive option. "Did I encode this restriction?" → yes, you did.
- The §17 decision gate blocks Sprint 1 story authorship. Do NOT write story files until those decisions are closed.
- Critical Don't-Miss Rules (Category 7) are intentionally redundant with prior categories. If you only have time to read one section, read that one.

**For Humans (maintenance guidance):**

- This file is generated from the pivot document and three rounds of party-mode review (Winston/Amelia/Murat). Updates happen when:
  - The pivot doc version bumps (e.g., v1.0 → v1.1 after §17 decisions land).
  - A new ADR supersedes an old one — update the relevant cross-references in this file.
  - An implementation pattern emerges that agents repeatedly get wrong — encode it.
  - A rule becomes obvious (everyone knows it now) — delete it. Lean wins.
- Quarterly review for staleness. Items most likely to age badly:
  - Specific version pins (Next.js 16, OpenNext, hls.js 1.5 — bump notes).
  - The legacy/superseded boundary (will be irrelevant once `index.html`/`mediamtx/` are deleted).
  - Open §17 decisions (close them out and remove the gate).
- Do NOT let this file balloon. The goal is leverage per token, not completeness.

**Provenance:**

- v1.0 — 2026-05-12. Generated via `bmad-generate-project-context` from `tv_poc_plan.md` v1.0. Categories 1 & 2 included Advanced Elicitation + Party Mode reviews with Winston, Amelia, and Murat. Categories 3–7 drafted directly from the pivot doc + prior-round consensus.
