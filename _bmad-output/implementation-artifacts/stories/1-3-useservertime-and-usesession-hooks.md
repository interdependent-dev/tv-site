---
story_id: 1.3
story_key: 1-3-useservertime-and-usesession-hooks
epic: 1
sprint: Sprint 1 — Schedule and single-channel playback
title: useServerTime and useSession React hooks
status: ready-for-dev
priority: Must
owners: ["Amelia (impl)"]
effort: M
dependencies: ["1.2"]
blocks: ["1.4"]
risks: ["R3", "R6"]
project_context_refs:
  - "Cat 2 (Zod LocalStorage validation, branded EpochMs type)"
  - "Cat 3 (useServerTime + useSession contracts)"
  - "Cat 4 (clock injection for tests)"
  - "Cat 7 #2 (schedule math sacred), #20 (CI flakiness rule)"
pivot_doc_refs: ["§10.2 (useServerTime)", "§10.3 (useSession + SessionV1 schema)"]
---

# Story 1.3: useServerTime and useSession React Hooks

## User Story

As **the `<ChannelPlayer>` (Story 1.4)** and **the Resume Modal (Story 3.1)**,
I want two contract-defined hooks (`useServerTime` and `useSession`) with deterministic, testable behavior,
So that schedule math has a single source of authoritative time and viewer session state has a single source of validated LocalStorage state.

## Context

These hooks are architectural primitives, not utilities. Their contracts are part of the public interface of the codebase (project-context.md Cat 3). Get them wrong now and every downstream client component carries the bug.

Three load-bearing rules from project-context.md:
- **`Date.now()` is BANNED in schedule math** (Cat 7 #2). Schedule math consumes `getServerNow()` from this hook.
- **Schedule math signature: `currentProgram(now: number, channel: Channel)`** (Cat 4 clock injection). Story 1.4 lives this rule; this story enables it.
- **LocalStorage reads go through Zod** (Cat 2 trust-boundary). Stale `itv.session.v1` from a previous app version must not crash the client.

## Acceptance Criteria

### `useServerTime` (pivot §10.2)

1. File: `app/(tv)/_components/providers/useServerTime.ts` (hook) + `ServerTimeProvider.tsx` (provider). Both are client components (`"use client"`).
2. Returns: `{ getServerNow(): EpochMs, ready: boolean, offsetMs: number }`.
3. `EpochMs` branded type imported from `lib/types/time.ts` (Cat 2 type discipline).
4. **On mount:** single `fetch('/api/time')`; on response, set `offsetMs = serverTsMs - Date.now()` and `ready = true`.
5. **Subsequent reads:** `getServerNow()` returns `(Date.now() + offsetMs) as EpochMs`.
6. **Re-syncs every 5 minutes** via `setInterval`.
7. **Re-syncs immediately on `visibilitychange → visible` after >60s hidden.** Browsers throttle background tabs; the offset may drift while hidden.
8. **On detected drift >500ms** (computed via a re-sync), `console.warn` and accept the new offset.
9. Initial `getServerNow()` calls before `ready` return `Date.now() as EpochMs` (graceful fallback; UI should respect `ready` to avoid premature schedule math).
10. **Optional server-rendered initial state:** `<ServerTimeProvider initialServerTsMs={...}>` prop accepted to eliminate the first-mount fetch waterfall.

### `useSession` (pivot §10.3)

11. File: `app/(tv)/_components/providers/useSession.ts` + `SessionProvider.tsx`. Client components.
12. Returns: `{ session: SessionV1 | null, updateSession(patch: Partial<SessionV1>): void, clearSession(): void }`.
13. `SessionV1` type and `SessionSchema` (Zod) live at `lib/types/session.ts`.
14. **On mount:** single read of `localStorage.getItem('itv.session.v1')`. Parsed via `SessionSchema.parse(...)`. On parse failure: `clearSession()` (treat as cold entry; do not crash). LocalStorage disabled / not available: silent no-op; `session` stays `null`.
15. **`updateSession(patch)`:** merges patch into current session, throttles writes to **1 per 2 seconds**.
16. **Final write on `visibilitychange → hidden` AND `beforeunload`.** BOTH listeners required (some browsers fire only one). Cancels any pending throttled write; writes immediately.
17. **`clearSession()`:** removes the LocalStorage key; sets `session` to `null`.
18. **`sessionId` lifecycle:** if missing on first read, generate a UUID (via `crypto.randomUUID()`) and persist immediately. Stable across page loads.
19. **`mode` field discriminated union:** `'live' | 'vod-detour'` (Cat 2 + Cat 7 #11).

### Tests (Cat 4 — Vitest + jsdom for these hooks; clock injection is mandatory)

20. **`useServerTime` tests:**
   - Offset calculation: mock `fetch('/api/time')` to return `serverTsMs = 1000000000000`; mock `Date.now()` to return `1000000000500` via `vi.setSystemTime`; assert `offsetMs === -500`, `getServerNow() === 1000000000000`.
   - Re-sync interval: advance fake timers by 5 min; assert second `fetch` fires.
   - Drift detection: re-sync returns offset differing by >500ms; assert `console.warn` called.
   - `visibilitychange → visible` after 90s hidden: assert immediate re-sync.

21. **`useSession` tests:**
   - Cold mount: LocalStorage empty; assert `session === null`; assert new `sessionId` generated and persisted.
   - Warm mount: pre-populate valid SessionV1; assert parsed and returned.
   - Invalid LocalStorage: pre-populate non-conformant JSON; assert `clearSession()` called and `session === null` (no throw).
   - Throttling: call `updateSession` 5× within 1 second; assert exactly 1 LocalStorage write.
   - Visibility-hidden flush: call `updateSession`, fire `visibilitychange → hidden` before throttle expires; assert immediate write.
   - LocalStorage disabled (mock `localStorage` throws on `getItem`): assert no crash, `session === null`.

## Dependencies

- **Upstream:** 1.2 (needs `/api/time`).
- **Downstream:** 1.4 (player consumes both hooks), 3.1 + 3.3 (Resume modal + edge cases consume `useSession`).

## Risk Alignment (pivot §16)

| ID | Risk | Mitigation in this story |
|---|---|---|
| R3 | Clock skew on viewer device | Re-sync every 5 min + on `visibilitychange → visible` after 60s hidden + drift detection. AC 6–8. |
| R6 | LocalStorage cleared by viewer | Graceful: `session === null` on read; sessionId auto-regenerates. Already Closed in epic risk register; this story enforces. |

## Implementation Notes

- **NO `Date.now()` outside this hook's offset-update path.** The hook IS the abstraction. Other code reads `getServerNow()`.
- **Branded `EpochMs` type:**
  ```ts
  export type EpochMs = number & { readonly __brand: 'EpochMs' };
  ```
  Constructed via `getServerNow()` return; never bare-cast.
- **Throttle implementation:** use a small custom throttle (Cat 5 — no `lodash`). 5 lines:
  ```ts
  function createThrottle(fn: () => void, ms: number) {
    let lastRun = 0, pending: ReturnType<typeof setTimeout> | null = null;
    return () => { /* schedule or skip */ };
  }
  ```
- **`crypto.randomUUID()`** is available in modern browsers and Workers. No polyfill needed.
- **Schema forward-compat (per D3 additive-evolution + D2-VOD-URL consensus, 2026-05-12):** SessionSchema in this story includes Sprint-1 launch fields, but Story 1.4 / 2.4 call sites write additional optional fields (`lastAssetHlsUrl`, `lastAssetTitle`) that Story 3.1 formally consumes. To prevent strip-on-read, either (preferred) declare those fields here already as `z.string().nullable().optional()` (pure declaration; no behavior reads them until Sprint 3), OR apply `.passthrough()` to the schema object so unknown keys persist on parse. Story 3.1 tightens the schema (`resumeModalShown: z.boolean().default(false)`, etc.) when its consumer code lands — additive, non-breaking.
- **Test setup:** use `@testing-library/react` `renderHook` with the provider as wrapper. `vi.useFakeTimers()` + `vi.setSystemTime()` for clock control. Per Cat 7 #20 — CI flakiness from real-time clocks is a defect, not a retry.

## Definition of Done

- [ ] AC 1–21 met.
- [ ] Tests green (clock-injected, no real-time).
- [ ] Hook contracts cross-referenced in project-context.md Cat 3.
- [ ] Sprint-status.yaml `1-3-useservertime-and-usesession-hooks` → done.
