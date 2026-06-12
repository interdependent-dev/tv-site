/* ── SEASON 1 — the 12-event timeline ─────────────────────────────────
   Single source of truth for the guide's Season strip, the lower-third
   season banner, and /timeline. Source of record: the Studio 1 Event
   Timeline & PA Playbook (Studio-Event-Timeline PDF v1).

   KILL SWITCH: set SEASON_ENABLED to false and the Season strip and
   the season ticker banner remove themselves; /timeline keeps working.

   Dates are Pacific. `start`/`end` are YYYY-MM-DD calendar days
   (inclusive); null start = unscheduled (TBD); null end with a start =
   open-ended (runs until the next milestone). `status` is the PDF's
   Set / Proposed flag. */

window.SEASON_ENABLED = true;

window.SEASON = {
  number: 1,
  name: 'Old Hollywood Reborn',
  events: [
    { n: 1,  slug: 'signing-day',      short: 'SIGNING DAY',       name: 'Screenplay Signing Day',
      date: 'Fri Jun 19',              start: '2026-06-19', end: '2026-06-19', status: 'set',
      tag: 'Opens the cycle',
      produces: 'The cycle opens — screenplays optioned by first-look Readers.' },
    { n: 2,  slug: 'production-draft', short: 'PRODUCTION DRAFT',  name: 'Production Draft',
      date: 'Jun 20–22',               start: '2026-06-20', end: '2026-06-22', status: 'set',
      tag: 'The readathon weekend',
      produces: 'Championship Weekend — scripts read and championed; director adoption opens.' },
    { n: 3,  slug: 'option-pool',      short: 'OPTION-POOL PARTY', name: 'Option-Pool Party',
      date: 'Jul 10–12',               start: '2026-07-10', end: '2026-07-12', status: 'proposed',
      tag: 'EP soft-circle',
      produces: 'Executive Producers soft-circle — 10% deposits committed to the Studio.' },
    { n: 4,  slug: 'director-reveal',  short: 'DIRECTOR REVEAL',   name: 'Director Reveal',
      date: 'Fri Jul 24',              start: '2026-07-24', end: '2026-07-24', status: 'proposed',
      tag: 'The auteur is named',
      produces: 'Directors attach; casting opens; .tv auditions go live.' },
    { n: 5,  slug: 'cast-party',       short: 'CAST PARTY',        name: 'Cast Party',
      date: 'Sat Sep 12',              start: '2026-09-12', end: '2026-09-12', status: 'proposed',
      tag: 'Announce + celebrate the actors',
      produces: 'Cast announced; headshots and audition tapes go live.' },
    { n: 6,  slug: 'last-call',        short: 'LAST CALL',         name: 'Last Call',
      date: 'Thu Sep 17',              start: '2026-09-17', end: '2026-09-17', status: 'set',
      tag: 'Crew up',
      produces: 'Department heads announced; crew attachment deadline.' },
    { n: 7,  slug: 'lock-party',       short: 'LOCK PARTY',        name: 'Lock Party',
      date: 'Fri Sep 18',              start: '2026-09-18', end: '2026-09-18', status: 'set',
      tag: 'Script · budget · schedule locked',
      produces: 'Shooting script, budget, and schedule locked; cap table opens for EPs.' },
    { n: 8,  slug: 'greenlight',       short: 'GREENLIGHT DINNER', name: 'Greenlight Dinner',
      date: 'Tue Sep 22',              start: '2026-09-22', end: '2026-09-22', status: 'set',
      tag: 'Flagship · on the equinox',
      produces: 'Directors present, actors table-read — then EPs exercise their options.' },
    { n: 9,  slug: 'preview-dailies',  short: 'PREVIEW DAILIES',   name: 'Preview Production Dailies',
      date: 'Sep 23 → ~Oct 22',        start: '2026-09-23', end: '2026-10-22', status: 'set',
      tag: 'Previews shoot',
      produces: 'Preview production shoots; daily stills released.' },
    { n: 10, slug: 'preview-premiere', short: 'PREVIEW PREMIERE',  name: 'Preview Premiere',
      date: 'Oct 23–25',               start: '2026-10-23', end: '2026-10-25', status: 'set',
      tag: 'Audience go/no-go',
      produces: 'Previews open; the audience votes go/no-go — trailers drop, Plots tickets on sale.' },
    { n: 11, slug: 'feature-dailies',  short: 'FEATURE DAILIES',   name: 'Feature Production Dailies',
      date: 'Oct 26 →',                start: '2026-10-26', end: null, status: 'set',
      tag: 'Survivors only',
      produces: 'Feature production shoots; live camera feed for subscribers.' },
    { n: 12, slug: 'premiere',         short: 'PREMIERE!',         name: 'Premiere!',
      date: 'Release season · TBD',    start: null, end: null, status: 'proposed',
      tag: 'The closing episode',
      produces: 'The final features premiere — the closing episode of the series.' },
  ],
};

/* Per-event state at `nowMs`: 'past' | 'now' | 'next' | 'future'.
   Calendar days are Pacific (UTC-7 in season); a day ends at
   midnight PT = 07:00 UTC the next day. */
window.seasonStates = function (nowMs) {
  const dayStartUtc = function (iso) {
    const p = iso.split('-');
    return Date.UTC(+p[0], +p[1] - 1, +p[2], 7);          // 00:00 PT
  };
  const dayEndUtc = function (iso) {
    const p = iso.split('-');
    return Date.UTC(+p[0], +p[1] - 1, +p[2] + 1, 7);      // 24:00 PT
  };

  const states = window.SEASON.events.map(function (ev) {
    if (!ev.start) return 'future';                        // TBD
    const s = dayStartUtc(ev.start);
    const e = ev.end ? dayEndUtc(ev.end) : null;
    if (nowMs >= s && (e === null || nowMs < e)) return 'now';
    if (e !== null && nowMs >= e) return 'past';
    return 'future';
  });

  // Promote the first scheduled future event to 'next'.
  for (let i = 0; i < states.length; i++) {
    if (states[i] === 'future' && window.SEASON.events[i].start) {
      states[i] = 'next';
      break;
    }
  }
  return states;
};

/* "TODAY" / "TOMORROW" / "IN N DAYS" for a YYYY-MM-DD start (PT). */
window.seasonCountdown = function (iso, nowMs) {
  if (!iso) return '';
  const p = iso.split('-');
  const startUtc = Date.UTC(+p[0], +p[1] - 1, +p[2], 7);
  const days = Math.ceil((startUtc - nowMs) / 86400000);
  if (days <= 0) return 'TODAY';
  if (days === 1) return 'TOMORROW';
  return 'IN ' + days + ' DAYS';
};
