# Debugging

Symptom-first runbook. Each section starts with the operator-visible complaint and walks to the root cause.

## "The pass says expired but I set it for tomorrow"

Almost always the **first-open window** clamping the effective expiry.

`EffectiveExpiresAt = min(expires_at, first_opened_at + valid_hours_after_first_open)`.

If `valid_hours_after_first_open` is set (UI label: "Valid for N hours after first open") and the recipient already opened the link, the clock that matters is the one anchored to `first_opened_at`. A 24-hour share with `valid_hours_after_first_open=2`, opened at 09:00, expires at 11:00 — not the next morning.

Check:

```sql
SELECT id, expires_at, valid_hours_after_first_open, first_opened_at
FROM guest_pass.guest_passes WHERE id = $1;
```

If `first_opened_at` is non-NULL and small `valid_hours_after_first_open` × hours has already elapsed, the pass is correctly "expired". Issue a new pass with a longer first-open window or `valid_hours_after_first_open = 0` (no first-open clamp).

## "The recipient says the page returns 404"

Five candidates, in order of likelihood:

1. **Wrong public URL.** `share_url` in the create response is built from `public_base_url`. If that's empty, the link is plugin-relative (`/p/<token>`) — fine when the operator copies from a browser tab that already has the right origin, broken when they paste it into chat. Set `public_base_url` in `Config` to the externally reachable origin.
2. **Reverse proxy not forwarding `/p/*`.** The host typically forwards all plugin routes, but a custom proxy in front of Silo may only forward `/api/*`. The pass page is served by the plugin SPA at `GET /p/{token}` (see `manifest.json` http_routes). Confirm it reaches the plugin.
3. **Token typo.** Tokens are URL-safe base64 of 32 random bytes (43 chars). One wrong character → `GetPassByToken` returns `ErrNotFound` → `publicNotFound` writes `{"status":"not_found"}` with 404.
4. **Pass revoked or in a non-active terminal state.** The preview endpoint returns 403 with `{"status":"revoked"|"expired"|…}` rather than 404 in those cases. So an actual 404 means the token never matched.
5. **Plugin not configured.** `requireStore` middleware returns 503 with `{"error":{"code":"not_configured"}}`. If the recipient is seeing 503 not 404, the plugin's `database_url` is missing.

## "Recipient hits a cap they shouldn't have"

Caps that surprise:

- **`max_devices`** defaults to **1**. Two browsers on the same machine count as two devices (different fallback hashes). To allow a phone + a TV, bump this.
- **`max_concurrent_streams`** defaults to **1**. A user who reloads mid-stream may briefly hold two slots: the old grant's `expires_at` is the pass's effective expiry (could be hours away), and the new request will be rejected with `concurrent_stream_limit_reached`. Releasing happens only on host stream-mint failure (`ReleaseGrantSlot`), not on graceful client disconnect.
- **`max_opens`** counts every preview-then-open transition. The SPA loads the preview first (no counter) and then calls `/open` only when the recipient actively engages. But refreshing post-PIN does call `/open` again.
- **`max_plays`** counts every successful `/play` call. Reseeking does not call `/play`; closing and reopening playback often does.

To diagnose, run `GET /api/admin/passes/{id}/events` and look for `play_rejected_*` or `rejected_*` events. The most recent rejection name tells you the gate that fired.

Releasing a stuck concurrent-stream slot manually:

```sql
DELETE FROM guest_pass.guest_pass_grants
WHERE pass_id = $1 AND expires_at > NOW();
```

(Filter further by `created_at` if the pass has both legitimate active and stale slots.)

## "IP allowlist locked me out after a VPN switch"

`lock_to_first_ip` is the more common offender: the first request that lands on `/api/public/passes/{token}/open` writes its IP into `guest_passes.first_ip`. Every subsequent request whose `X-Silo-Client-IP` doesn't match is rejected with `{"status":"ip_locked"}`. The lock is set on **first open**, not first preview.

Fix options:

- Reset the lock: `UPDATE guest_pass.guest_passes SET first_ip = '' WHERE id = $1;`. Next open will re-anchor.
- Issue a replacement pass without `lock_to_first_ip`. The original cannot be un-locked through the UI.

`ip_allowlist` is the explicit form. CIDR notation works (`Pass.AllowsIP` parses with `net.ParseCIDR`); bare addresses also work. Empty list = allow all. Whitespace is trimmed but no other normalisation is applied, so `192.168.1.1` and `192.168.001.001` are different entries.

## "Country allowlist rejects requests from inside the allowed country"

`requestCountry` reads `CF-IPCountry`, then `X-Geo-Country`, then `X-Country-Code` — the first non-empty wins, uppercased. If none of those headers reach the plugin, `requestCountry` returns `""` and `allowsCountry` returns `false` whenever the merged allowlist is non-empty (no header = no country = denied).

Cloudflare in front: `CF-IPCountry` is set automatically. Other proxies: you must add a header transform that emits one of the three.

`country_allowlist` and `geofence` are unioned — both treated as allowlists. If the operator filled in `country_allowlist=["NL"]` and `geofence=["BE"]`, requests from NL or BE pass.

## "X-Silo-Client-IP is empty in all my audit rows"

Expected when the host has not stamped the header. The plugin deliberately does **not** read `X-Forwarded-For` from the request — see the comment in `internal/server/request_ctx.go`. Spoof-resistance: a guest cannot lie about their IP because the only header the plugin trusts is one the host injects on the gRPC-forwarded request.

Until the host adds the resolved-IP header:

- `lock_to_first_ip` will "lock" to the empty string — every subsequent request will match (empty == empty) and the policy is effectively a no-op.
- `ip_allowlist` will block everything unless the allowlist contains the empty string (don't do this).
- The `{{ip}}` watermark token renders as blank.

Decision: enable IP-derived features only after confirming non-empty IPs land in `guest_pass_events.ip` for known good test traffic.

## "PIN keeps prompting for a guest who knows the right PIN"

Most often the SPA is sending an empty `pin` field. The `accessRequest` body trims whitespace; an empty string after trim falls through to `PINMatch`, which calls `VerifyPIN`. For bcrypt-stored hashes (`$2…`), `bcrypt.CompareHashAndPassword(stored, "")` returns an error and `VerifyPIN` returns `PINVerifyFail`. The response is HTTP 401 `{"status":"pin_required"}`.

Less commonly: a legacy sha256 PIN hash (from before bcrypt was introduced) was matched and the background rehash failed (network blip, DB pool exhaustion). Next attempt should re-trigger the rehash. Force-reset by re-creating the pass.

## "I see two `guest_passes` rows with the same token hash"

Cannot happen: `token_hash` has a UNIQUE constraint (migration `0001_init.up.sql`). The token itself is 32 bytes of CSPRNG, so collisions are astronomically unlikely. If you see this, restore from backup — the schema is compromised.

## "Watermark text doesn't render the IP"

`renderWatermarkText` substitutes `{{ip}}` from the request's `clientIP(r)`. If the host hasn't stamped `X-Silo-Client-IP`, `clientIP` returns empty and the template ends up with a literal blank where the IP would be. See the "IP is empty" entry above.

Also: substitution is one-pass. A title containing `{{pass_id}}` will _not_ recursively expand if used inside the template — by design (the comment in `watermark.go` calls this out).

## Admin endpoint returns 401/403

- **401 `unauthenticated`** — the host did not forward `X-Silo-User-Id`. Check that the request is hitting the plugin via the host's authenticated admin route, not directly.
- **403 `forbidden`** — user is authenticated but `X-Silo-User-Role != "admin"`. Their host-side role needs to be admin.

## Catalog search returns empty or `playable=false`

`hSearchCatalog` does two steps:

1. Asks the host SDK (`ListLibraryMedia`) for items by title query.
2. Resolves each item to a preferred media-file-id via `Store.PlayableFileIDs`, which reads `public.media_files` directly.

Common reasons for `playable=false`:

- The role's grant on `public.media_files` is missing — every lookup returns no rows. Confirm with `\dp public.media_files` and re-grant if needed.
- All matching files have `missing_since IS NOT NULL` (the host's scanner flagged them as gone). They're filtered out by the resolver query.
- The host returns a `MediaID` that doesn't match either `content_id` or `episode_id` on any file row. Most likely a mismatch between host-side and plugin-side naming.

Empty search results (`items: []`) when titles obviously match means the host SDK call failed. Check plugin logs for `catalog_search_failed`.

## "Plugin won't configure: `database_url is required`"

The `Configure` RPC validates `database_url` before doing anything else. The error surfaces in the host's install form. Check that the install form actually has a value and the JSON shape is `{"value": "postgres://…"}` (not a bare string) — `runtime.go` extracts via `m["value"]` first, then falls back to `firstString(m)`.

## Inspecting state directly

Useful one-liners (substitute `$1`):

```sql
-- All recent activity for a pass
SELECT created_at, event_type, ip, user_agent, attrs
FROM guest_pass.guest_pass_events WHERE pass_id = $1
ORDER BY created_at DESC LIMIT 50;

-- Devices on a pass
SELECT device_id, first_ip, user_agent, created_at, last_seen_at
FROM guest_pass.guest_pass_devices WHERE pass_id = $1;

-- Active stream slots
SELECT id, device_id, created_at, expires_at
FROM guest_pass.guest_pass_grants
WHERE pass_id = $1 AND expires_at > NOW();

-- Effective status without going through the API
SELECT id, title, expires_at, first_opened_at, valid_hours_after_first_open,
       open_count, max_opens, play_count, max_plays, revoked_at
FROM guest_pass.guest_passes WHERE id = $1;
```
