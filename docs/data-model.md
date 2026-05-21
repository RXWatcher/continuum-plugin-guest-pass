# Data model

Tables, indexes, transactional invariants, and the cross-schema reach. Source of truth is `internal/migrate/files/0001_init.up.sql`; this page narrates the choices.

All tables live in the plugin's own `guest_pass` schema. The connection role owns that schema (`CREATE SCHEMA guest_pass AUTHORIZATION plugin_guest_pass`) and additionally needs `SELECT` on `public.media_files`.

## Schema layout

```
guest_pass.
├── guest_passes          one row per share, ~35 columns
├── guest_pass_events     audit log, pruned by retention task
├── guest_pass_devices    per-pass device registry (UNIQUE(pass_id, device_id))
├── guest_pass_grants     short-lived concurrent-stream reservations
└── app_config            singleton row (id = 1)
```

The `app_config` table uses `CONSTRAINT app_config_singleton CHECK (id = 1)` plus `INSERT … ON CONFLICT DO NOTHING` to ensure one row.

## guest_passes

Wide row holding the pass itself. Sensitive columns:

- `token_hash TEXT NOT NULL UNIQUE` — sha256 hex of the 32-byte URL-safe token. UNIQUE blocks any chance of collision, though at 256-bit entropy this is astronomical.
- `pin_hash TEXT NOT NULL DEFAULT ''` — bcrypt hash of the PIN, or empty when no PIN is set. Legacy rows may contain a sha256 hex hash; the verify path detects this (prefix-test on `$2`) and the next successful verify upgrades it in place via `Store.RehashPIN`.

JSON columns store allowlists:

- `ip_allowlist`, `country_allowlist`, `geofence` are `JSONB` arrays. The plugin always reads through `json.Unmarshal`; nothing relies on Postgres JSON operators, so re-encoding via UPDATE is safe.

Counter columns (`open_count`, `play_count`) are `INTEGER NOT NULL DEFAULT 0`. Incremented inside an `UPDATE … RETURNING …` with the cap check inlined, so no separate read-then-write.

Indexes:

- `guest_passes_expires_at_idx ON (expires_at)` — for future cleanup queries; not currently used by the runtime path.
- `guest_passes_target_idx ON (target_type, target_id)` — for finding passes by their target (e.g. "all passes pointing at this movie").

## guest_pass_events

Audit log. Columns: `pass_id` (FK with `ON DELETE CASCADE`), `event_type`, `ip`, `user_agent`, `attrs JSONB`, `created_at`.

Event types follow the convention noted in `operations.md`:

- Bare verbs for success: `created`, `opened`, `revoked`, `grant_minted`.
- Prefix `rejected_<reason>` for `/open` rejections.
- Prefix `play_rejected_<reason>` for `/play` rejections.
- Prefix `<action>_rejected_<reason>` for verification-gate rejections inside `verifyPassAccess` (e.g. `play_rejected_ip_locked`).
- `grant_failed` carries `attrs.error` (host stream mint failure detail).

Indexes:

- `guest_pass_events_pass_id_idx ON (pass_id, created_at DESC)` — backs `ListEvents`.
- `guest_pass_events_created_at_idx ON (created_at)` — backs `PruneEvents`.

Retention: the `maintenance` scheduled task issues `DELETE FROM guest_pass_events WHERE created_at < NOW() - interval '<N> days'` every 6 hours. Default 180 days.

## guest_pass_devices

One row per `(pass_id, device_id)`. The UNIQUE constraint makes re-registration idempotent.

`RegisterDevice` runs in a transaction:

1. `SELECT 1 FROM guest_passes WHERE id = $1 FOR UPDATE` — serialises concurrent inserts for the same pass.
2. Fast path: `UPDATE … SET last_seen_at = NOW(), …` — if a row exists, just bump.
3. Slow path: `SELECT COUNT(*)` then `INSERT` if under `max_devices`. Otherwise return `ErrDeviceLimit`.

The `FOR UPDATE` on the parent pass row is the synchronisation point. Two concurrent registrations of two different device IDs for the same pass cannot both pass the count check.

## guest_pass_grants

Short-lived reservations representing "this device is currently playing". One row per playback attempt that successfully passed the concurrency check.

`ReserveGrantSlot` is the only writer:

1. `BEGIN`.
2. `SELECT 1 FROM guest_passes WHERE id = $1 FOR UPDATE` — same parent-row lock used everywhere.
3. If `max_concurrent > 0`: `SELECT COUNT(*) FROM guest_pass_grants WHERE pass_id = $1 AND expires_at > NOW()`. If at cap, return `ErrConcurrencyLimit`.
4. `INSERT INTO guest_pass_grants (pass_id, device_id, expires_at)`.
5. `COMMIT`.

Grant rows have `expires_at = pass.EffectiveExpiresAt()`. A pass expiring in 6 hours has grant rows that "expire" in 6 hours. Live grants are counted by `expires_at > NOW()`; expired ones are inert but not auto-deleted.

`ReleaseGrantSlot` deletes by id, used only when the host stream mint fails after the slot was reserved.

There is no automatic cleanup of dead grant rows. If table size becomes an issue:

```sql
DELETE FROM guest_pass.guest_pass_grants WHERE expires_at < NOW() - interval '7 days';
```

Index: `guest_pass_grants_active_idx ON (pass_id, expires_at)` — supports the count query.

## app_config

Singleton row, two operator-tunable columns:

- `public_base_url TEXT` — used by `shareURL` to build external links. Empty = relative URLs.
- `audit_retention_days INTEGER` — feeds the maintenance task.

Read/write through `Store.GetAppConfig` / `Store.UpdateAppConfig`. `GetAppConfig` lazily inserts the row if missing. `UpdateAppConfig` does an upsert.

Manifest-supplied values are imported once via `ImportLegacyAppConfig`, which is a no-op when the row already differs from defaults. See `operations.md` for the asymmetry rationale.

## Cross-schema reach

The only query against `public.*` is in `internal/store/catalog.go`:

```sql
JOIN public.media_files mf ON (mf.content_id = r.content_id OR mf.episode_id = r.content_id)
WHERE mf.missing_since IS NULL
```

Selects the highest-resolution non-missing media file for each requested content id. Used by the admin catalog search to surface `playable=true/false` per item.

Required grant:

```sql
GRANT USAGE ON SCHEMA public TO plugin_guest_pass;
GRANT SELECT ON public.media_files TO plugin_guest_pass;
```

This is intentionally narrow — `SELECT` on exactly one table — and is documented as a temporary irreducibility until the host SDK exposes a "resolve playable file for content id" call. When that lands, the join can be replaced and the grant dropped.

## Transactional guarantees

Every write that needs to enforce a cap acquires `SELECT 1 FROM guest_passes WHERE id = $1 FOR UPDATE` first:

- Device registration (count + insert).
- Concurrent-stream reservation (count + insert).

The exception is `RecordOpen`/`RecordPlay`, which do not take a lock because they enforce the cap inside a single `UPDATE … WHERE open_count < max_opens` — Postgres serialises the row read implicitly under `READ COMMITTED`.

Consequence: even under thundering-herd "open this pass" attempts, the counter cannot overshoot. The first N succeed; the rest get `ErrOpenLimit` and a `rejected_open_limit_reached` audit row.

## Migrations

Single migration file: `internal/migrate/files/0001_init.up.sql` (and `…down.sql`). `golang-migrate` applies pending migrations on every `Configure`. The migration uses `CREATE TABLE IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS` so re-running is safe.

Future migrations should follow the same numbering convention (`0002_*.up.sql` / `0002_*.down.sql`). The runner reads from an embedded FS.

## What the plugin never writes

- Anything in `public.*` — the role doesn't have grants beyond `SELECT public.media_files`.
- Anything outside its own schema. If you see plugin writes elsewhere in your logs, something else is using the same role.

## Backup considerations

The plugin's data is independent from the host's: backing up `pg_dump --schema=guest_pass continuum > guest_pass.sql` captures every pass, audit row, and config row in isolation. Restoring is similarly self-contained.

`guest_pass_events` dominates volume; the rest is tiny.
