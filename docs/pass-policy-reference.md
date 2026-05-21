# Pass policy reference

Every knob on a pass, what it actually does, where it is enforced, and how it interacts with the others. The README gives the one-paragraph summary; this is the field-by-field reference.

The authoritative type is `store.Pass` (`internal/store/pass.go`). Validation and normalisation live in `internal/server/validation.go` and `internal/server/handlers_admin.go`.

## Identity / scope

| Field | Required | Notes |
| --- | --- | --- |
| `title` | yes | Free-form, shown to the recipient. Trimmed. Used in default watermark template (`{{title}}`). |
| `target_type` | yes | Only `media_file` is accepted today (`validTargetType`). Other values reject with HTTP 400 `bad_target_type`. |
| `target_id` | yes | For `media_file`, must parse as a positive integer when minting the stream grant. The admin UI fills this from the catalog search resolver. |
| `note` | no | Operator-only memo. Never shown to the recipient. |
| `created_by` | auto | Stamped from `X-Continuum-User-Id` at create time. Not user-supplied. |

## Expiry

| Field | Default | Behaviour |
| --- | --- | --- |
| `expires_at` | (computed) | Absolute RFC3339 timestamp. Parsed by `parseExpiry`. Must be in the future. |
| `expires_in_hours` | 24 | Convenience for clients that prefer relative expiry. Used only when `expires_at` is empty. Capped at one year (`24*365`). |
| `valid_hours_after_first_open` | 0 | When > 0, the pass also expires at `first_opened_at + N hours`. `EffectiveExpiresAt` returns the **earlier** of this and `expires_at`. |

A pass without `valid_hours_after_first_open` has a fixed wall-clock expiry. A pass with it has a maximum of `min(absolute, first-open + N)` — so an unopened pass still expires absolutely.

## Usage caps

All caps treat `0` as **unlimited**. Defaults vary.

| Field | Default | Enforced where | Notes |
| --- | --- | --- | --- |
| `max_opens` | 0 (∞) | `Store.RecordOpen` UPDATE | Counter bumped inside the `UPDATE … WHERE open_count < max_opens` so no TOCTOU. Returns `ErrOpenLimit`. |
| `max_plays` | 0 (∞) | `Store.RecordPlay` UPDATE | Same atomic pattern. Returns `ErrPlayLimit`. |
| `max_watch_minutes` | 0 (∞) | Host stream mint | Passed through as `MaxWatchMinutes` in the `ScopedStreamRequest`. Enforced by the host, not the plugin. |
| `max_concurrent_streams` | 1 | `Store.ReserveGrantSlot` (TX with `SELECT … FOR UPDATE` on parent pass) | Returns `ErrConcurrencyLimit`. Slot lifetime = pass `EffectiveExpiresAt`. |
| `max_devices` | 1 | `Store.RegisterDevice` (TX with `SELECT … FOR UPDATE`) | Returns `ErrDeviceLimit`. Counts unique `device_id` rows. |
| `per_item_play_count` | false | Plugin tags; host respects | Currently passed through metadata only; the plugin enforces total `max_plays`. |
| `session_grace_minutes` | 0 | Host stream grant | Plugin stores; host uses it when re-issuing tokens during a single session. |

### Device ID handling

`accessRequest.DeviceID` is what the SPA POSTs. If absent or whitespace, `fallbackDeviceID` synthesises one from `sha256(clientIP + "\n" + UA)`. That means:

- Two browsers on the same machine usually have different UAs → different fallback device IDs → both count.
- Same browser, same IP across two pass opens → identical fallback device ID → counts as one.
- Empty `clientIP` (host hasn't stamped the header) → fallback IDs collapse for users with the same UA, undercounting devices.

`deviceID` is trimmed to 128 chars and the `(pass_id, device_id)` index makes re-registration a no-op (just bumps `last_seen_at`).

## Playback policy

| Field | Default | Effect |
| --- | --- | --- |
| `max_resolution` | `1080p` | Mapped to height via `resolutionHeight`: `480p→480`, `720p→720`, `1080p→1080`, `4k`/`2160p→2160`, else 0 (no cap). Passed to host as `MaxResolutionHeight`. |
| `allow_downloads` | false | Forwarded to host stream grant. Host gates download URLs. |
| `allow_direct_play` | false | Forwarded to host. When false, the host transcodes regardless of client capability. |
| `disable_seeking` | false | Forwarded to host. UI hint plus transcode mode. |

These are surfaced to the host through `ScopedStreamRequest`. The plugin does not gate playback itself — it grants the host a scoped token and lets the host serve.

## Identity binding

| Field | Default | Behaviour |
| --- | --- | --- |
| `require_pin` | false | When true, a non-empty `pin` is required at create time. Stored as bcrypt. |
| `pin` (input only) | — | Plaintext at create; never stored. Verified via `VerifyPIN`. Legacy sha256 hashes auto-upgraded to bcrypt on first match. |
| `lock_to_first_ip` | false | Latches `first_ip` from the first successful `/open` and rejects every subsequent request whose `clientIP(r)` differs. Reset by zeroing `first_ip`. |
| `ip_allowlist` | `[]` | List of bare addresses or CIDR ranges (`net.ParseCIDR`). Empty = allow all. Combined with `lock_to_first_ip` as conjunction (both must pass). |
| `country_allowlist` | `[]` | ISO 3166-1 alpha-2 codes. Source headers (in order): `CF-IPCountry`, `X-Geo-Country`, `X-Country-Code`. |
| `geofence` | `[]` | Unioned with `country_allowlist` as a second allowlist. Empty `country_allowlist` + non-empty `geofence` → only `geofence` applies. |

PIN verify is constant-time for the legacy sha256 path (`subtle.ConstantTimeCompare`); bcrypt has its own constant-time compare.

`splitList` is the parser for `ip_allowlist`, `country_allowlist`, `geofence` — comma/newline/tab separators, trims each token, drops empties. No further sanitisation.

## Watermarking

| Field | Default | Behaviour |
| --- | --- | --- |
| `watermark_mode` | `"none"` if both empty, `"all"` if profile set but no mode | Normalised by `normalizeWatermarkMode`. Accepts `none`, `visible`, `burned_in`, `forensic`, `all` (also `burnedin`, `burned-in` aliases). Unknown modes silently drop. Multiple modes can be combined with comma/space/plus. |
| `watermark_profile` | `""` | Text template with `{{token}}` placeholders. Defaults to `"Guest pass {{pass_id}} · {{ip}} · {{time}}"` when empty and no logo. |
| `watermark_logo_url` | `""` | URL to a logo. When set and `watermark_profile` is empty, no text template is auto-generated. |

Template substitutions:

- `{{pass_id}}` — numeric pass id.
- `{{title}}` — pass title.
- `{{ip}}` — `clientIP(r)` from `X-Continuum-Client-IP`.
- `{{device_id}}` — request device id (real or fallback).
- `{{subject}}` — literal `guest-pass:<id>`.
- `{{time}}` — current UTC time, RFC3339.

Final string is truncated to 160 characters. Substitution is **one-pass**, so a value containing `{{…}}` is not re-expanded.

## Lifecycle

| Field | Notes |
| --- | --- |
| `open_count` | Bumped by `RecordOpen` UPDATE. |
| `play_count` | Bumped by `RecordPlay` UPDATE. |
| `first_ip` | Latched on first `/open` if empty. |
| `first_opened_at` | `COALESCE(first_opened_at, NOW())` on first `/open`. |
| `revoked_at` | Set by `RevokePass`. One-way — `UPDATE … WHERE revoked_at IS NULL`. |
| `created_at` / `updated_at` | Standard audit timestamps. `updated_at` is touched on every state-changing path. |

## Status derivation

`Pass.Status(now)` returns one of:

1. `"revoked"` if `revoked_at IS NOT NULL`.
2. `"expired"` if `EffectiveExpiresAt() <= now`.
3. `"open_limit_reached"` if `max_opens > 0 && open_count >= max_opens`.
4. `"play_limit_reached"` if `max_plays > 0 && play_count >= play_count`.
5. `"active"` otherwise.

Status is computed on read; nothing persists it. There's no background job to "expire" a pass.

## Cross-field invariants

- `require_pin = true` without a non-empty `pin` is rejected at create time (`missing_pin`).
- `expires_at` must be future-dated (`expires_at must be in the future`).
- `expires_in_hours > 24*365` is rejected; the cap defends against typos creating effectively-permanent shares.
- `target_type` outside the allowlist is rejected — currently only `media_file`. Library-slice support is a future expansion.

## Field defaults summary

When the operator submits an otherwise-empty body with just `title`, `target_type`, `target_id`, the resulting pass has:

- 24-hour absolute expiry; no first-open clamp.
- All counts unlimited (`max_opens`, `max_plays`, `max_watch_minutes`).
- 1 concurrent stream, 1 device.
- `max_resolution=1080p`, no direct play, no downloads, no seek lock.
- No PIN, no IP lock, no IP allowlist, no country gate, no watermark.

The default is permissive on cosmetics, restrictive on concurrency — pick the recipient and accept they'll watch one stream at a time on one device.
