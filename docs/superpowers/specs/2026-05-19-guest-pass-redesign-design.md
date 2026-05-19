# Guest pass UI redesign

## Summary

Redesign the `continuum-plugin-guest-pass` web UI across both surfaces:

- the admin console used by Continuum operators
- the guest-facing public pass page used by recipients

The redesign keeps the existing React/Vite stack, existing API contracts, and current feature set. The work is strictly presentational and interaction-focused. No backend schema or endpoint changes are part of this design.

The target outcome is:

- a polished internal admin tool optimized for fast pass creation, monitoring, and policy tuning
- a premium public handoff that feels intentional and trustworthy rather than exposing internal system details

## Goals

- Make new pass creation the fastest and clearest workflow in the admin surface.
- Preserve monitoring of recent passes without forcing operators into a separate page or mode.
- Keep advanced restrictions available but collapsed by default.
- Improve scanability, affordance, and action clarity across the admin UI.
- Replace the guest page's diagnostic tone with a focused media handoff and playback experience.
- Add stronger loading, empty, success, and error states without changing underlying behavior.

## Non-goals

- No backend API changes.
- No data model or permission changes.
- No change to pass policy semantics.
- No conversion to a wizard flow.
- No new third-party UI libraries for this pass.

## Current problems

### Admin surface

- The page is a single-column slab, so settings, creation, and monitoring compete equally for attention.
- The default form does not strongly separate fast-path fields from exception handling.
- Recent pass rows rely on ambiguous icon-only actions.
- Generic cards, pills, shadows, and typography make the interface feel like scaffolding rather than a product tool.
- Loading and empty states are plain text rather than composed UI states.

### Guest surface

- The page leads with internal pass details instead of the shared media and action.
- Raw target identifiers and policy counters are too prominent.
- PIN unlock and playback are functional but do not feel like a deliberate handoff.
- The player state does not clearly dominate once playback is available.

## Design direction

### Admin tone

The admin UI should feel operational, restrained, and fast. It stays dark, but with cleaner hierarchy and fewer generic container treatments. The visual language should support repeated use by operators rather than feel decorative.

### Guest tone

The guest page should feel like a premium presentation layer over the same permissions model. It should communicate trust, clarity, and purpose with stronger spacing, headline treatment, and media-led composition.

## Information architecture

### Admin page

Use a split-focus desktop layout:

- primary creation rail on the left
- secondary monitoring rail on the right

On smaller screens, collapse to a single-column stack with creation first, settings second, and monitoring third.

The admin layout is organized into these regions:

1. top header
2. creation rail
3. utility/settings panel
4. monitoring panel
5. transient feedback states

### Guest page

Use a single centered presentation surface with clear state transitions:

1. loading/opening
2. locked by PIN
3. ready to play
4. active playback
5. unavailable/error

## Component design

### Admin header

Keep the back link and refresh affordance, but increase hierarchy:

- product eyebrow
- page title
- one-line operational summary
- lightweight refresh action

Do not add secondary navigation.

### Creation rail

The creation rail is the dominant region. It is split into ordered sections that reflect the operator workflow:

1. media selection
2. pass basics
3. policy summary
4. advanced restrictions
5. create/share actions

#### Media selection

Media search becomes the anchor step. It should include:

- search field
- type filter
- results list
- selected media summary

The selected state should feel committed and easy to verify, using poster art when available plus concise metadata. The operator should not need to inspect raw IDs to trust the selection.

#### Presets

Templates remain near the top of the form but become clearly framed presets rather than equal-weight cards. They should read as accelerators for common pass types.

#### Pass basics

Visible by default:

- title
- note
- expires in hours
- max plays
- max resolution

These fields represent the fast path for most passes and should remain compact and easy to scan.

#### Policy summary

The live summary remains visible in the main flow and updates as the form changes. It should read as a compact entitlement preview for the selected media rather than a loose pile of decorative chips.

#### Advanced restrictions

Advanced options remain collapsed by default behind a dedicated secondary control. When expanded, they are grouped into sub-sections:

- playback limits
- access controls
- watermarking
- network and geography

The expanded state must feel structured, not like an unbounded dump of edge-case fields.

#### Create/share actions

After a successful create, the success state becomes a composed share block containing:

- generated share URL
- copy action
- print action

The share block should feel like the natural end state of the workflow.

### Utility/settings panel

Plugin settings move out of the main creation emphasis and into a compact utility panel. It should still be easy to edit, but it should not visually outrank pass creation.

Keep:

- public base URL
- audit retention days

### Monitoring panel

Recent passes become a stronger operational review surface.

#### Metrics

Keep summary counts for active, revoked, and opened passes, but make them slimmer and less card-heavy so they support the list rather than dominate it.

#### Pass rows

Each row is divided into three zones:

- identity
- current state
- actions

Identity includes:

- pass title
- compact media metadata

Current state includes:

- status
- expiry
- usage progress
- critical restrictions such as PIN or IP lock

Actions should no longer rely on ambiguous icon-only buttons. At minimum, destructive and high-frequency actions must be explicit through icon-plus-text or labeled affordances.

Target actions:

- copy link
- duplicate
- revoke
- view activity

#### Activity expansion

Activity remains inline under the row, but it should read as an event history rather than a debug block. Keep timestamps and IP context where available.

### Guest presentation surface

The guest page should lead with the shared media and the next user action.

Primary content:

- pass title
- media art if available
- note when present
- expiry
- dominant CTA

Secondary content:

- resolution
- device limit
- watch time
- play limit

Do not foreground raw internal identifiers such as `target_type:target_id`.

### PIN unlock state

If a PIN is required, the page opens in a focused unlock state before revealing the full ready-to-play layout. The unlock step should feel intentional and trustworthy, not like a form fragment inserted above diagnostics.

### Playback state

Once playback is available, the player becomes the dominant element. Supporting pass details remain available but visually demoted so the video owns the page.

## Layout rules

### Desktop admin

- Use a two-column layout with a wider creation rail and narrower monitoring rail.
- Settings sit within the secondary side of the page, not inside the primary creation block.
- Maintain clear section boundaries through spacing, border rhythm, and surface contrast rather than repeated heavy cards.

### Mobile admin

- Collapse to one column.
- Preserve workflow order: creation first, settings second, monitoring third.
- Avoid horizontal compression of dense action groups.

### Guest page

- Keep a single centered layout.
- Let the hero content breathe with larger spacing and stronger typography than the admin UI.
- Keep entitlement metadata below the main action hierarchy.

## Visual system

### Typography

- Replace the current default sans stack with `Geist` as the primary web font, with a clean system sans fallback if the font import fails.
- Use a clearer scale between operational labels, section titles, and guest-facing headlines.
- Keep the admin side clean and sans-only.

### Color

- Preserve a dark admin surface with one restrained accent.
- Reduce generic gray-on-gray sameness by improving contrast between shell, section, input, and hover surfaces.
- Keep success and error states integrated with the palette rather than looking pasted on.

### Surfaces

- Reduce dependence on generic bordered cards and pill badges.
- Use lighter grouping and more deliberate spacing for the admin interface.
- Give the guest page a more premium framed surface with stronger separation between invitation and playback contexts.

### Icons

- Existing icon dependency can remain for this pass to stay scoped.
- Icon use should be less dominant in high-risk actions; text clarity takes priority.

## Interaction design

### Admin

- Hover and pressed states should feel tactile but restrained.
- Action clarity matters more than ornamental motion.
- Expanding advanced options should preserve context and not jump the layout unexpectedly.
- Selecting media should visibly confirm the current target.

### Guest

- The primary CTA should remain obvious in locked, ready, and playback-adjacent states.
- Error and unavailable states should stay direct and readable.

## State design

### Loading

Replace plain text loading copy with shaped placeholders or structured loading blocks in:

- media search results
- recent passes list
- guest pass opening state

### Empty

Provide composed empty states for:

- no media matches
- no recent passes
- no activity for a pass

Each empty state should indicate the next meaningful action.

### Success

Creating a pass should produce a focused share state rather than a generic status banner.

### Error

Errors remain inline and direct. Keep the current behavior of surfacing backend error messages where available.

## Accessibility

- Preserve visible keyboard focus states.
- Ensure icon-driven actions have readable labels or accessible names.
- Maintain sufficient contrast in both admin and guest themes.
- Keep form labels above inputs and preserve clear validation messaging.

## Implementation notes

- Keep the work inside `web/src/main.tsx` and `web/src/styles.css` unless a small helper extraction improves readability.
- Preserve current API calls and state shape.
- Reuse current helper functions where possible.
- Avoid importing new UI frameworks or animation libraries.

## Testing and verification

Before implementation is called complete:

- run the existing frontend tests in `web/`
- run a production build in `web/`
- manually inspect the rendered admin and guest flows if a local browser run is available

Testing focus:

- no regressions in media search interaction
- no regressions in pass creation
- no regressions in activity expansion
- no regressions in guest PIN unlock and playback launch flows

## Recommended implementation order

1. restructure admin layout and section hierarchy
2. redesign creation rail and success state
3. redesign monitoring list and action affordances
4. redesign guest locked, ready, and playback states
5. polish loading, empty, success, and error states
6. run tests and build verification
