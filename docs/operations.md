# Operations

Day-two runbook for operators running `continuum.guest-pass`. The README covers _what_ the plugin is and _how_ to install it; this page covers _running_ it: configuration changes, the maintenance task, observability, and lifecycle actions.

## Configuration surfaces

There are two places config lives, and the relationship is not symmetric:

1. **Manifest-supplied install config** — `database_url`, `public_base_url`, `audit_retention_days` arrive through the host's `Configure` RPC and land in `internal/runtime/runtime.go`. `database_url` is required; the others have defaults.
2. **Singleton DB row (`app_config`)** — `public_base_url` and `audit_retention_days` are also persisted in a one-row table inside the plugin's schema, and the admin `PATCH /api/admin/config` endpoint writes there.

On startup the plugin calls `Store.ImportLegacyAppConfig`. It seeds the DB row from the manifest **only if the DB row still holds defaults**. Once an operator has edited config through the admin UI, the manifest values are ignored on subsequent restarts. This is intentional: the operator's UI edit is the source of truth.

Practical consequences:

- Updating `public_base_url` in the host's plugin install form after the operator changed it in the UI does nothing. Edit it through the admin UI.
- Updating `audit_retention_days` in the manifest after the operator changed it in the UI does nothing either. Same fix.
- `database_url` is the exception — it is never persisted in `app_config` and always comes from the manifest. Rotating the password means editing the install config.

## Lifecycle of a pass

```
created → active → (revoked | expired | open_limit_reached | play_limit_reached)
```

The status string is derived on read by `Pass.Status(now)` (see `internal/store/pass.go`). It is **not** persisted. That means:

- A pass with `MaxOpens=5` and `OpenCount=5` will show `open_limit_reached` immediately on next read — no background job flips it.
- Reviving a "limit reached" pass is therefore impossible without admin-side schema surgery (and even then, the recipient experience would be broken — no UI for it).
- A revoked pass cannot be un-revoked. `RevokePass` is a one-way `UPDATE … WHERE revoked_at IS NULL`.

`EffectiveExpiresAt` is the **earlier** of `expires_at` and `first_opened_at + valid_hours_after_first_open`. This is the most common source of "why is my pass expired already?" — see `debugging.md`.

## Maintenance scheduled task

Capability: `scheduled_task.v1`, id `maintenance`, cron `17 */6 * * *` (every 6 hours at :17).

What it does: `Store.PruneEvents(retentionDays)` — a single `DELETE FROM guest_pass_events WHERE created_at < NOW() - interval`. Returns count and exits.

What it does **not** do:

- It does **not** prune `guest_passes`. Expired/revoked passes are kept for the audit trail forever (cheap; one row per pass).
- It does **not** prune `guest_pass_devices`. Device rows persist for the life of the pass.
- It does **not** prune `guest_pass_grants`. The grant rows are short-lived (their `expires_at` is the pass's effective expiry) and only consulted via `expires_at > NOW()`, so stale rows are inert. They still accumulate.

If you need to purge dead grants for size/cleanliness, run periodically:

```sql
DELETE FROM guest_pass.guest_pass_grants WHERE expires_at < NOW() - interval '30 days';
```

The retention default is 180 days. Values below 1 fall back to the default (defensive guard in `runtime.go` and `normalizeAppConfig`).

## Where logs come from

`writeInternal` is the only path that logs through `Deps.Logger` (an `hclog.Logger` wired from `cmd/.../main.go`). Best-effort audit writes (`RecordEvent` failures) are swallowed silently — the public flow does not block on the audit table. If you need to see why an audit row didn't land, add temporary logging there.

Plugin process logs go through the Continuum host's plugin-log channel. Look for `name=continuum-plugin-guest-pass` lines.

## Observability checklist

There is no `/metrics` endpoint. To observe a pass in flight:

1. Find the pass id in the admin UI list (or `SELECT id FROM guest_pass.guest_passes WHERE title = …`).
2. Inspect the audit trail with `SELECT event_type, ip, created_at FROM guest_pass.guest_pass_events WHERE pass_id = $1 ORDER BY created_at DESC LIMIT 200;` — or use the admin endpoint `GET /api/admin/passes/{id}/events`.
3. Cross-reference open/play counters: `SELECT open_count, max_opens, play_count, max_plays, first_ip, first_opened_at FROM guest_pass.guest_passes WHERE id = $1;`.
4. Active concurrent streams: `SELECT COUNT(*) FROM guest_pass.guest_pass_grants WHERE pass_id = $1 AND expires_at > NOW();`.

Event-type naming convention: successful actions are bare verbs (`opened`, `created`, `revoked`, `grant_minted`). Rejections are prefixed (`rejected_*` for `/open` flow, `play_rejected_*` for `/play` flow, `*_rejected_<reason>` for verification gates).

## Rotating database credentials

1. Edit the role's password in Postgres.
2. Update the `database_url` in the plugin install config in the Continuum admin UI.
3. The host re-invokes `Configure`, which builds a new pool, swaps it in atomically (`poolPtr.Swap`), and closes the old one. No restart required.

Migrations run in `Configure` (`migrate.Run(ctx, cfg.DatabaseURL)`); they are idempotent (`golang-migrate` with file source). Re-configuring is safe.

## Data growth profile

- `guest_passes`: roughly one row per share. Indexed on `expires_at` and `(target_type, target_id)`. Wide row (\~30 columns) but small in absolute terms.
- `guest_pass_events`: dominant table. Multiple rows per pass interaction. Pruned by retention.
- `guest_pass_devices`: bounded by `max_devices` per pass. Effectively negligible.
- `guest_pass_grants`: one row per playback attempt that successfully reserved a slot. Pruned only by the manual query above; otherwise grows linearly with usage.

If `guest_pass_events` size becomes a problem, lower `audit_retention_days`. The maintenance task is cheap (single bulk `DELETE` with `created_at` filter).

## When to restart vs. reconfigure

- Schema migration changes (new migration file in `internal/migrate/files`) → restart not needed, just re-`Configure`. But in practice the operator does this by upgrading the plugin binary, which the host handles.
- Embedded SPA changes → require a new binary (the SPA is `go:embed`-ed via `web/embed.go`).
- Config-only changes → use the admin UI `PATCH /api/admin/config`, or the host install form for `database_url`. No restart.

## Host header dependency

Read `debugging.md` (the `X-Continuum-Client-IP` section) before enabling any IP-based policy. The plugin deliberately does not fall back to `X-Forwarded-For`; if the host has not stamped the resolved client IP, every IP-derived policy is inert and audit rows record empty IPs.
