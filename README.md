# Guest Pass for Continuum

`continuum.guest-pass` issues short-lived public links for tightly scoped media access. An authenticated Continuum operator can invite a friend to one show or movie without provisioning an account: each pass carries its own token, scope, expiry, and per-pass policy knobs (concurrency cap, watermark, PIN, IP/country allowlist, etc.).

## Category

Lives under **Sharing**. The Sharing category currently contains two plugins:

- [`continuum.guest-pass`](https://github.com/RXWatcher/continuum-plugin-guest-pass) (this) — one-to-one invites to a specific item or library slice.
- [`continuum.public-catalog`](https://github.com/RXWatcher/continuum-plugin-public-catalog) — a public-facing landing/advertising page for the library.

## Capabilities

| Type | ID | Purpose |
| --- | --- | --- |
| `http_routes.v1` | `guest-pass` | Admin UI and API at `/admin/*` and `/api/admin/*`, plus the public pass page at `/p/*`, public API at `/api/public/*`, and static assets at `/assets/*`. |
| `scheduled_task.v1` | `maintenance` | Cron `17 */6 * * *`. Prunes `guest_pass_events` rows older than `audit_retention_days`. |

## Dependencies

Standalone sharing layer. The plugin does not depend on other plugins for core operation:

- Catalog browsing in the admin UI goes through the Continuum host SDK (`ListLibraryMedia`).
- Final content-ID → media-file-ID resolution at pass-creation time reads `public.media_files` directly (a `SELECT` grant on that single table is required until the SDK exposes a resolver call).
- Guest playback flows through the host's scoped stream grants — the plugin never exposes broad library permissions to the public route.

Host: [`ContinuumApp/continuum`](https://github.com/ContinuumApp/continuum). SDK: [`ContinuumApp/continuum-plugin-sdk`](https://github.com/ContinuumApp/continuum-plugin-sdk).

## External services

- **Postgres**, in a dedicated `guest_pass` schema. The plugin owns its own tables (`guest_passes`, `guest_pass_grants`, `guest_pass_events`, app config) and runs its own migrations on startup via `golang-migrate`.

## Pass model

Each pass is created by an admin and stored with only a sha256 hash of its URL-safe token (32 bytes of entropy) — the plaintext token is shown to the operator exactly once. Optional PINs are hashed with bcrypt; legacy sha256 PIN hashes are upgraded in place on first successful verify.

A pass targets either a specific item or a library slice via `target_type` / `target_id` and carries the policy that governs the recipient's access:

- **Expiry.** Hard `expires_at`, plus an optional `valid_hours_after_first_open` window so the clock starts when the recipient actually opens the link. `Status` and `EffectiveExpiresAt` take the earlier of the two.
- **Usage caps.** `max_opens`, `max_plays`, `max_watch_minutes`, `max_concurrent_streams`, `max_devices`. Open and play counters are enforced inside the `UPDATE` statement so there is no TOCTOU race. Concurrent-stream slots are reserved transactionally via `guest_pass_grants` with a `FOR UPDATE` on the parent pass row, and can be released if a downstream step (host stream mint) fails.
- **Playback policy.** `max_resolution`, `allow_downloads`, `allow_direct_play`, `disable_seeking`, `session_grace_minutes`, `per_item_play_count`.
- **Identity binding.** `require_pin` + bcrypt-hashed PIN, `lock_to_first_ip` (binds to the first observed client IP), `ip_allowlist` (bare addresses or CIDR), `country_allowlist`, `geofence`.
- **Watermarking.** `watermark_mode`, `watermark_profile`, `watermark_logo_url`, with `{{ip}}` substitution available in profiles.
- **Lifecycle.** `revoked_at` is set by the admin endpoint and short-circuits `Status` to `revoked`. Status also rolls forward to `expired`, `open_limit_reached`, or `play_limit_reached` as caps are hit.

> Per-request client IP feeds `lock_to_first_ip`, `ip_allowlist`, audit rows, and the `{{ip}}` watermark token. It is read from the `X-Continuum-Client-IP` header injected by the host — the plugin deliberately does **not** fall back to `X-Forwarded-For` from arbitrary callers, since that would let guests spoof their own IP. Until the host stamps that header, IP-derived features are inert and audit rows record an empty IP.

## Audit

Every guest-pass interaction can be logged to `guest_pass_events` (pass id, event type, IP, user agent, free-form attrs JSON, timestamp). `RecordEvent` is best-effort — the public flow does not block on logging. `ListEvents` exposes the 200 most recent rows per pass to the admin UI.

The `maintenance` scheduled task runs every six hours and calls `PruneEvents(retentionDays)`, which deletes audit rows older than `audit_retention_days` (default 180). Retention should be tuned to match your privacy policy.

## Configuration

| Key | Required | Description |
| --- | --- | --- |
| `database_url` | yes | Postgres DSN for the `guest_pass` schema. The role only needs ownership of `guest_pass` plus `SELECT` on `public.media_files`. |
| `public_base_url` | no | Absolute URL used when returning share links. Empty returns plugin-relative paths; set this when Continuum sits behind a reverse proxy and links need an absolute external origin. Validated as an absolute URL on configure. |
| `audit_retention_days` | no | Days of audit history to keep. Defaults to 180; values below 1 fall back to the default. |

Example DSN:

```text
postgres://plugin_guest_pass:password@postgres:5432/continuum?search_path=guest_pass&sslmode=disable
```

Database setup:

```sql
CREATE ROLE plugin_guest_pass WITH LOGIN PASSWORD '<chosen>';
CREATE SCHEMA guest_pass AUTHORIZATION plugin_guest_pass;
GRANT CONNECT ON DATABASE continuum TO plugin_guest_pass;
GRANT USAGE ON SCHEMA public TO plugin_guest_pass;
GRANT SELECT ON public.media_files TO plugin_guest_pass;
```

Migrations under `internal/migrate/files` are applied automatically on startup; the operator only needs to create the schema and grant the connect role.

## Detailed docs

- [`docs/setup-debug-flows.md`](docs/setup-debug-flows.md) — setup checklist, route map, operational flows, and a debugging runbook covering proxy 404s, expiry, scope review, and audit retention.

## Build and release

```bash
make build
make test
```

CI builds linux-amd64 binaries on push to main via the reusable workflow in [RXWatcher/continuum-plugin-repository](https://github.com/RXWatcher/continuum-plugin-repository) and publishes them to the catalog at [`./binaries/`](https://github.com/RXWatcher/continuum-plugin-repository/tree/main/binaries).
