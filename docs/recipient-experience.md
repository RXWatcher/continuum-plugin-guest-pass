# Recipient experience

What the guest (the person who received a share link) actually sees, and how the operator can predict or diagnose recipient-side failures without ever holding the recipient's device.

## What the recipient gets

A URL of the form `<public_base_url>/p/<token>` — for example `https://watch.example.com/p/abc…43-char-token…xyz`.

There is no recipient account, no signup, no login. The token is the credential. Anyone with the URL can attempt to open the pass; per-pass policy decides whether they succeed.

## What happens when the recipient opens the link

The browser hits `GET /p/{token}` and is served the embedded SPA (a one-page React app with a `<base href>` injected from the host's mount-path header). The SPA then:

1. Calls `GET /api/public/passes/{token}` (preview). This **does not** count as an "open" — it only fetches the pass shape so the SPA knows whether to render the PIN prompt, the play button, or an "expired" notice.
2. If the pass needs a PIN, prompts for it. The PIN is included in the next request body.
3. Calls `POST /api/public/passes/{token}/open` with `{pin, device_id}`. This is the request that increments `open_count`, latches `first_ip`/`first_opened_at`, registers the device, and gates on identity policy. Returns the decorated pass on success.
4. On Play tap, calls `POST /api/public/passes/{token}/play` with `{pin, device_id}`. This increments `play_count`, reserves a concurrent-stream slot, asks the host SDK to mint a scoped stream grant, and returns the playback URL.

The recipient never sees `/api/admin/*` — that's bearer-gated by the host's admin identity headers.

## Statuses surfaced to the recipient

The SPA renders different UI states based on the `status` string returned by the API:

| Status | What the SPA shows | Trigger |
| --- | --- | --- |
| `active` | Normal flow — play button, pass info | Pass is valid, no caps hit |
| `revoked` | "This pass has been revoked" | Admin called `/revoke` |
| `expired` | "This pass has expired" | `EffectiveExpiresAt() <= now` |
| `open_limit_reached` | "Maximum opens reached" | `open_count >= max_opens` |
| `play_limit_reached` | "Maximum plays reached" | `play_count >= max_plays` |
| `device_limit_reached` | "Too many devices" | New device id when at `max_devices` |
| `concurrent_stream_limit_reached` | "Too many active streams" | Concurrent slot reservation failed (HTTP 429) |
| `ip_not_allowed` | "Access denied" | `ip_allowlist` non-empty and IP not in list |
| `ip_locked` | "Access denied" (locked to first IP) | `lock_to_first_ip` and IP differs from `first_ip` |
| `country_not_allowed` | "Access denied" (geographic) | Country allowlist non-empty, request country missing or not allowed |
| `pin_required` | PIN prompt (also for bad PIN) | `require_pin` and missing/wrong PIN (HTTP 401) |
| `not_found` | Generic 404 page | Token doesn't match any pass |
| `not_configured` | "Service unavailable" (503) | Plugin has no DB pool (operator skipped config) |

Note that `ip_locked` and `ip_not_allowed` look identical to the recipient ("access denied") so they can't tell which gate fired. The operator distinguishes by checking the audit trail.

## What the recipient cannot do

- Cannot see who shared the pass beyond what the operator put in `title`.
- Cannot see other passes, even on the same host.
- Cannot enumerate by guessing tokens (43 chars of base64 entropy; sha256 hash storage).
- Cannot bypass watermarking — the host bakes it in before streaming.
- Cannot share the link transitively in a useful way if `lock_to_first_ip` is set (first opener wins).
- Cannot use the pass past `EffectiveExpiresAt` even if mid-stream — the host stream grant expires at the same time.

## Things operators forget to tell recipients

**The PIN.** The token is in the URL but the PIN is out-of-band. Send it via a different channel (SMS for an emailed link, Signal for an SMS link). The recipient cannot recover a PIN from the operator without a manual reset (currently means: revoke and re-issue).

**The "valid N hours after first open" clock.** A recipient who saves a link to read later may not realise that the clock starts the moment they tap it. If you want a 24-hour window starting at first open, set `valid_hours_after_first_open=24` and `expires_at` further out.

**Device limits.** If you say "watch on your TV" but `max_devices=1` and they previewed it once on their phone (which already registered the phone's fallback device id), the TV will get `device_limit_reached`. For TV + phone, set `max_devices=2`.

**Country gates.** A recipient on a corporate or mobile VPN often appears as a different country. `country_allowlist=["NL"]` will reject a Dutch recipient routed through a US datacentre VPN.

## Recipient-visible audit footprint

The plugin records `opened` events to `guest_pass_events` with the recipient's IP, user agent, and best-fit device id. This is visible to the admin via the events list (200 most recent per pass). The recipient is not told the audit exists; depending on the operator's privacy posture they may want to communicate it.

Audit IPs depend on the host stamping `X-Continuum-Client-IP`. If that's not happening yet, recipient IPs in audit rows will be empty.

## Recovering a stuck recipient

Common operator workflow when a recipient says "it's broken":

1. Look up the pass in the admin UI; check the events list.
2. The most recent event is almost always `rejected_<reason>` or `play_rejected_<reason>`. The reason tells you which gate fired.
3. Address the gate per the matching section in `debugging.md`, or issue a fresh pass with relaxed constraints.

There is no operator action that resumes a recipient's in-flight session — fixes apply to the next attempt only.
