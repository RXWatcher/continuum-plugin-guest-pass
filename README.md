# continuum-plugin-guest-pass

Temporary public links for tightly scoped Continuum media access.

## Capabilities

| Capability | Notes |
|---|---|
| `http_routes.v1` | Admin API/UI and public guest pass pages. |
| `scheduled_task.v1` | Prunes old audit events. |

## Current scope

This first version ships an embedded Vite/React SPA for the admin and guest screens. It manages pass creation, validation, expiry, revocation, open/play limits, PINs, first-IP locking, IP allowlists, country/geofence checks, device limits, concurrent scoped-grant limits, and audit events.

For `media_file` targets, it asks Continuum's RuntimeHost for a narrowly scoped guest stream grant and passes through watch-time limits, resolution caps, direct-play/download flags, seeking policy, and watermark policy.

Watermarks support visible browser overlays, burned-in ffmpeg overlays, forensic token metadata, or all three. Text templates can include pass id, IP, device id, subject, title, and time. Logo overlays are supported by URL for visible playback and by local absolute file path for burned-in playback.

## Install

```sql
CREATE ROLE plugin_guest_pass LOGIN PASSWORD '<chosen>';
CREATE SCHEMA guest_pass AUTHORIZATION plugin_guest_pass;
GRANT CONNECT ON DATABASE continuum TO plugin_guest_pass;
```

Configure `database_url` with `search_path=guest_pass`.

## Build & test

```bash
make build
make test
```
