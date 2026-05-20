# Guest Pass Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the guest pass admin and public guest-pass UI so pass creation is faster, monitoring is clearer, and the guest handoff feels premium without changing backend behavior.

**Architecture:** Keep the existing React/Vite single-entry frontend and current API calls. Extract a small presentation helper module for label and summary logic that can be tested in isolation, then reshape `main.tsx` and `styles.css` around a split-focus admin layout and a media-led guest page.

**Tech Stack:** React 19, TypeScript, Vite, Vitest, CSS

---

## File map

- Modify: `web/src/main.tsx`
  - Reorganize admin and guest page JSX, success states, action labels, and loading/empty rendering.
- Modify: `web/src/styles.css`
  - Replace the flat single-column styling with the redesigned admin workspace, guest handoff surface, and responsive states.
- Create: `web/src/presentation.ts`
  - Hold pure formatting helpers for pass row summaries, guest entitlement facts, and share-state text.
- Create: `web/src/presentation.test.ts`
  - Cover the new presentation helpers with focused Vitest cases.

## Task 1: Add presentation helpers and test seams

**Files:**
- Create: `web/src/presentation.ts`
- Create: `web/src/presentation.test.ts`
- Modify: `web/src/main.tsx`

- [ ] **Step 1: Write the failing helper tests**

```ts
import { describe, expect, it } from "vitest";
import {
  buildGuestFacts,
  buildPassRowMeta,
  buildPassRowTags,
  formatUsageStat,
} from "./presentation";

describe("buildPassRowMeta", () => {
  it("prefers readable expiry text over raw target ids", () => {
    expect(
      buildPassRowMeta({
        effective_expires_at: "2026-05-20T12:00:00Z",
        max_resolution: "1080p",
        target_type: "media_file",
        target_id: "42",
      }),
    ).toContain("Expires");
  });
});

describe("buildPassRowTags", () => {
  it("surfaces key restrictions only when enabled", () => {
    expect(
      buildPassRowTags({
        require_pin: true,
        lock_to_first_ip: true,
        allow_downloads: false,
        allow_direct_play: false,
      }),
    ).toEqual(["PIN", "IP lock"]);
  });
});

describe("formatUsageStat", () => {
  it("renders unlimited caps with the infinity symbol", () => {
    expect(formatUsageStat("Plays", 2, 0)).toBe("Plays 2/∞");
  });
});

describe("buildGuestFacts", () => {
  it("returns guest-facing entitlement facts without raw target identifiers", () => {
    expect(
      buildGuestFacts({
        max_resolution: "4k",
        max_devices: 2,
        max_watch_minutes: 180,
        max_plays: 1,
      }),
    ).toEqual([
      ["Resolution", "4k"],
      ["Devices", "2"],
      ["Watch time", "180 min"],
      ["Play limit", "1"],
    ]);
  });
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `pnpm test presentation.test.ts`

Expected: FAIL with import or export errors for the new helper functions.

- [ ] **Step 3: Write the minimal presentation helpers**

```ts
export function formatUsageStat(label: string, current: number, limit: number): string {
  return `${label} ${current}/${limit > 0 ? String(limit) : "∞"}`;
}

export function buildPassRowMeta(pass: {
  effective_expires_at: string;
  max_resolution: string;
}): string[] {
  return [`Expires ${formatShortDate(pass.effective_expires_at)}`, pass.max_resolution];
}

export function buildPassRowTags(pass: {
  require_pin: boolean;
  lock_to_first_ip: boolean;
  allow_downloads: boolean;
  allow_direct_play: boolean;
}): string[] {
  const tags: string[] = [];
  if (pass.require_pin) tags.push("PIN");
  if (pass.lock_to_first_ip) tags.push("IP lock");
  if (pass.allow_downloads) tags.push("Downloads");
  if (pass.allow_direct_play) tags.push("Direct play");
  return tags;
}

export function buildGuestFacts(pass: {
  max_resolution: string;
  max_devices: number;
  max_watch_minutes: number;
  max_plays: number;
}): Array<[string, string]> {
  return [
    ["Resolution", pass.max_resolution],
    ["Devices", pass.max_devices > 0 ? String(pass.max_devices) : "∞"],
    ["Watch time", `${pass.max_watch_minutes > 0 ? pass.max_watch_minutes : "∞"} min`],
    ["Play limit", pass.max_plays > 0 ? String(pass.max_plays) : "∞"],
  ];
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `pnpm test presentation.test.ts`

Expected: PASS with 4 passing tests.

- [ ] **Step 5: Wire the helpers into `main.tsx`**

```ts
import {
  buildGuestFacts,
  buildPassRowMeta,
  buildPassRowTags,
  formatUsageStat,
} from "./presentation";
```

- [ ] **Step 6: Commit the helper seam**

```bash
git add web/src/main.tsx web/src/presentation.ts web/src/presentation.test.ts
git commit -m "refactor: add guest pass presentation helpers"
```

## Task 2: Restructure the admin workspace around creation and monitoring

**Files:**
- Modify: `web/src/main.tsx`
- Test: `web/src/presentation.test.ts`

- [ ] **Step 1: Add a focused test for row tag composition needed by the admin list**

```ts
it("includes download and direct-play affordances when enabled", () => {
  expect(
    buildPassRowTags({
      require_pin: false,
      lock_to_first_ip: false,
      allow_downloads: true,
      allow_direct_play: true,
    }),
  ).toEqual(["Downloads", "Direct play"]);
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `pnpm test presentation.test.ts`

Expected: FAIL because the current helper does not yet include the expected tag order or values.

- [ ] **Step 3: Update the helper implementation to satisfy the admin row design**

```ts
export function buildPassRowTags(pass: {
  require_pin: boolean;
  lock_to_first_ip: boolean;
  allow_downloads: boolean;
  allow_direct_play: boolean;
}): string[] {
  const tags: string[] = [];
  if (pass.require_pin) tags.push("PIN");
  if (pass.lock_to_first_ip) tags.push("IP lock");
  if (pass.allow_downloads) tags.push("Downloads");
  if (pass.allow_direct_play) tags.push("Direct play");
  return tags;
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `pnpm test presentation.test.ts`

Expected: PASS with the new tag case green.

- [ ] **Step 5: Rebuild the admin page structure in `main.tsx`**

```tsx
<main className="app-shell">
  <header className="topbar">...</header>

  {error && <div className="alert">{error}</div>}

  <section className="workspace">
    <div className="creation-rail">
      {created && <ShareResult created={created} />}
      <form className="panel creation-panel" onSubmit={createPass}>...</form>
    </div>

    <aside className="monitoring-rail">
      <form className="panel utility-panel" onSubmit={saveConfig}>...</form>
      <section className="panel pass-list">...</section>
    </aside>
  </section>
</main>
```

- [ ] **Step 6: Convert the pass list rows from icon-only actions to explicit actions**

```tsx
<div className="row-actions">
  {pass.share_url && (
    <button type="button" onClick={() => void copy(absoluteURL(pass.share_url))}>
      <Copy size={15} />
      <span>Copy link</span>
    </button>
  )}
  <button type="button" onClick={() => duplicate(pass)}>
    <Copy size={15} />
    <span>Duplicate</span>
  </button>
  <button type="button" onClick={() => void loadEvents(pass.id)}>
    <Activity size={15} />
    <span>Activity</span>
  </button>
  <button className="danger-button" type="button" onClick={() => revoke(pass.id)}>
    <Trash2 size={15} />
    <span>Revoke</span>
  </button>
</div>
```

- [ ] **Step 7: Replace raw row metadata with guest-pass-specific summaries**

```tsx
<div className="pass-row-copy">
  <div className="row-title">{pass.title}</div>
  <div className="row-meta">
    {buildPassRowMeta(pass).map((item) => (
      <span key={item}>{item}</span>
    ))}
  </div>
  <div className="metrics">
    <span>{pass.status}</span>
    <span>{formatUsageStat("Opens", pass.open_count, pass.max_opens)}</span>
    <span>{formatUsageStat("Plays", pass.play_count, pass.max_plays)}</span>
    {buildPassRowTags(pass).map((item) => (
      <span key={item}>{item}</span>
    ))}
  </div>
</div>
```

- [ ] **Step 8: Rework the success state into a share-result block**

```tsx
function ShareResult({ created }: { created: CreateResponse }) {
  const shareURL = absoluteURL(created.share_url);
  return (
    <section className="success share-result">
      <div>
        <p className="eyebrow">Share link ready</p>
        <h2>{created.pass.title}</h2>
        <code>{shareURL}</code>
      </div>
      <div className="share-actions">
        <button type="button" onClick={() => void copy(shareURL)}>
          <Copy size={16} />
          <span>Copy link</span>
        </button>
        <button type="button" onClick={() => printInvite(created)}>
          <Printer size={16} />
          <span>Print invite</span>
        </button>
      </div>
    </section>
  );
}
```

- [ ] **Step 9: Commit the admin structure changes**

```bash
git add web/src/main.tsx web/src/presentation.ts web/src/presentation.test.ts
git commit -m "feat: redesign guest pass admin workspace"
```

## Task 3: Redesign the guest-facing pass experience

**Files:**
- Modify: `web/src/main.tsx`
- Modify: `web/src/presentation.ts`
- Test: `web/src/presentation.test.ts`

- [ ] **Step 1: Add a failing test for guest entitlement facts with unlimited values**

```ts
it("formats unlimited guest facts cleanly", () => {
  expect(
    buildGuestFacts({
      max_resolution: "1080p",
      max_devices: 0,
      max_watch_minutes: 0,
      max_plays: 0,
    }),
  ).toEqual([
    ["Resolution", "1080p"],
    ["Devices", "∞"],
    ["Watch time", "∞ min"],
    ["Play limit", "∞"],
  ]);
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `pnpm test presentation.test.ts`

Expected: FAIL if the helper still returns numeric zeroes instead of guest-friendly unlimited values.

- [ ] **Step 3: Update the guest helper logic**

```ts
export function buildGuestFacts(pass: {
  max_resolution: string;
  max_devices: number;
  max_watch_minutes: number;
  max_plays: number;
}): Array<[string, string]> {
  return [
    ["Resolution", pass.max_resolution],
    ["Devices", pass.max_devices > 0 ? String(pass.max_devices) : "∞"],
    ["Watch time", pass.max_watch_minutes > 0 ? `${pass.max_watch_minutes} min` : "∞ min"],
    ["Play limit", pass.max_plays > 0 ? String(pass.max_plays) : "∞"],
  ];
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `pnpm test presentation.test.ts`

Expected: PASS with the unlimited case green.

- [ ] **Step 5: Replace the guest page diagnostics-first layout with a media-led handoff**

```tsx
<main className="guest-shell">
  <section className="guest-panel">
    <div className="guest-hero">
      {pass && <MediaThumb item={{ title: pass.title, poster_url: playback?.logo_url ?? "" }} />}
      <div className="guest-copy">
        <p className="eyebrow">Continuum guest pass</p>
        <h1>{pass?.title ?? "Guest Pass"}</h1>
        {pass?.note && <p className="guest-note">{pass.note}</p>}
        {pass && <p className="guest-expiry">Available until {formatDate(pass.effective_expires_at)}</p>}
      </div>
    </div>
    ...
  </section>
</main>
```

- [ ] **Step 6: Split the guest states into focused blocks**

```tsx
{status === "loading" && <GuestLoadingState />}

{needsPin && pass && (
  <form className="pin-panel" onSubmit={unlockPass}>
    <label>
      Access PIN
      <input value={pin} onChange={(event) => setPin(event.target.value)} autoFocus />
    </label>
    <button className="primary" type="submit">Unlock pass</button>
  </form>
)}

{pass && !playback?.stream_url && !needsPin && (
  <>
    <dl className="guest-facts">
      {buildGuestFacts(pass).map(([label, value]) => (
        <div key={label}>
          <dt>{label}</dt>
          <dd>{value}</dd>
        </div>
      ))}
    </dl>
    <button className="primary guest-cta" type="button" onClick={play} disabled={pass.status !== "active"}>
      <ExternalLink size={17} />
      <span>Start playback</span>
    </button>
  </>
)}
```

- [ ] **Step 7: Make the playback state visually dominant**

```tsx
{playback?.stream_url && (
  <section className="playback-stage">
    <div className="player-shell">
      <video controls autoPlay src={new URL(playback.stream_url, window.location.origin).toString()} />
      ...
    </div>
    <dl className="guest-facts guest-facts-compact">...</dl>
  </section>
)}
```

- [ ] **Step 8: Commit the guest-page restructure**

```bash
git add web/src/main.tsx web/src/presentation.ts web/src/presentation.test.ts
git commit -m "feat: redesign guest pass public handoff"
```

## Task 4: Replace the flat styling with the new admin and guest visual system

**Files:**
- Modify: `web/src/styles.css`
- Modify: `web/src/main.tsx`

- [ ] **Step 1: Add the font import and new root tokens**

```css
@import url("https://fonts.googleapis.com/css2?family=Geist:wght@400;500;600;700;800&display=swap");

:root {
  --font-body: "Geist", ui-sans-serif, system-ui, sans-serif;
  --radius: 1rem;
  --radius-lg: 1.5rem;
  --shell: #111318;
  --surface-soft: #181b22;
  --surface-strong: #1f2430;
  --accent-strong: #8fb4ff;
}
```

- [ ] **Step 2: Run a focused build after the font/token change**

Run: `pnpm build`

Expected: PASS. This guards against accidental CSS or JSX syntax breakage before the larger style rewrite.

- [ ] **Step 3: Add the split-focus admin layout styles**

```css
.workspace {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(22rem, 0.95fr);
  gap: 1.5rem;
  align-items: start;
}

.creation-rail,
.monitoring-rail {
  display: grid;
  gap: 1rem;
}

.creation-panel,
.utility-panel,
.pass-list {
  border-radius: var(--radius-lg);
  padding: 1.25rem;
}
```

- [ ] **Step 4: Add the new admin component styling for rows, actions, and share result**

```css
.share-result {
  display: grid;
  gap: 1rem;
}

.row-meta,
.share-actions,
.row-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.danger-button {
  border-color: color-mix(in srgb, var(--destructive) 40%, var(--border));
  color: color-mix(in srgb, var(--destructive) 20%, var(--foreground));
}
```

- [ ] **Step 5: Add guest handoff and playback styling**

```css
.guest-hero {
  display: grid;
  grid-template-columns: 120px minmax(0, 1fr);
  gap: 1rem;
  align-items: start;
}

.guest-facts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

.playback-stage {
  display: grid;
  gap: 1rem;
}
```

- [ ] **Step 6: Add loading and empty-state styling hooks**

```css
.loading-block,
.empty-block {
  border: 1px dashed var(--border);
  border-radius: var(--radius);
  padding: 1rem;
  color: var(--muted-foreground);
}

.skeleton-row {
  height: 4rem;
  border-radius: calc(var(--radius) - 2px);
  background: linear-gradient(90deg, var(--surface) 0%, var(--surface-hover) 50%, var(--surface) 100%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite linear;
}
```

- [ ] **Step 7: Add responsive collapse rules**

```css
@media (max-width: 1080px) {
  .workspace {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .guest-hero,
  .guest-facts,
  .row-actions,
  .share-actions {
    grid-template-columns: 1fr;
    flex-direction: column;
  }
}
```

- [ ] **Step 8: Commit the visual system pass**

```bash
git add web/src/styles.css web/src/main.tsx
git commit -m "style: refresh guest pass admin and guest UI"
```

## Task 5: Add composed loading and empty states, then verify the frontend

**Files:**
- Modify: `web/src/main.tsx`
- Modify: `web/src/styles.css`
- Test: `web/src/presentation.test.ts`

- [ ] **Step 1: Add shaped loading and empty-state JSX**

```tsx
{mediaLoading && (
  <div className="loading-block" aria-live="polite">
    <div className="skeleton-row" />
    <div className="skeleton-row" />
  </div>
)}

{!loading && passes.length === 0 && (
  <div className="empty-block">
    <strong>No guest passes yet.</strong>
    <p>Create a pass from the panel on the left to start sharing media.</p>
  </div>
)}

{eventLoading !== pass.id && (eventsByPass[pass.id]?.length ?? 0) === 0 && (
  <div className="empty-block">
    <strong>No activity yet.</strong>
    <p>This pass has not been opened or played.</p>
  </div>
)}
```

- [ ] **Step 2: Run the focused Vitest suite**

Run: `pnpm test presentation.test.ts`

Expected: PASS with all helper tests green.

- [ ] **Step 3: Run the full frontend test suite**

Run: `pnpm test`

Expected: PASS with `passTools.test.ts`, `mountPath.test.ts`, and `presentation.test.ts` green.

- [ ] **Step 4: Run the production build**

Run: `pnpm build`

Expected: PASS with Vite production assets emitted successfully.

- [ ] **Step 5: Review the diff before final reporting**

Run: `git status --short`

Expected: only the intended frontend files appear modified.

- [ ] **Step 6: Commit the final polish and verification-ready state**

```bash
git add web/src/main.tsx web/src/styles.css web/src/presentation.ts web/src/presentation.test.ts
git commit -m "feat: finish guest pass UI redesign"
```

## Self-review checklist

- Spec coverage:
  - Admin split layout is covered in Tasks 2 and 4.
  - Fast-path creation flow and share-result state are covered in Task 2.
  - Monitoring clarity and action labeling are covered in Task 2.
  - Guest locked, ready, and playback states are covered in Task 3.
  - Loading, empty, and success states are covered in Task 5.
- Placeholder scan:
  - No `TBD`, `TODO`, or deferred implementation notes remain.
- Type consistency:
  - `buildPassRowMeta`, `buildPassRowTags`, `formatUsageStat`, and `buildGuestFacts` are defined once and reused consistently across the plan.
