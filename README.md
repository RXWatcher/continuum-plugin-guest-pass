# Guest Pass for Continuum

`continuum.guest-pass` creates temporary public links for tightly scoped media
access. It is meant for short-lived sharing workflows where an authenticated
Continuum user grants access to a specific media item without creating a full
account for the recipient.

The plugin uses Continuum host APIs for scoped stream grants and keeps its own
audit trail in a dedicated Postgres schema.

## Features

- Creates temporary guest access links.
- Supports a configurable public base URL for reverse-proxied deployments.
- Records guest-pass audit events.
- Scheduled maintenance prunes old audit data.
- Strips and validates public request context through Continuum's scoped access
  model rather than exposing broad library permissions.

## Configuration

| Key | Required | Description |
|---|---|---|
| `database_url` | yes | Postgres DSN for the `guest_pass` schema. |
| `public_base_url` | no | Absolute URL used when returning share links. Empty returns relative plugin URLs. |
| `audit_retention_days` | no | How long to retain guest-pass audit rows. Defaults to 180 days. |

Example DSN:

```text
postgres://plugin_guest_pass:password@postgres:5432/continuum?search_path=guest_pass&sslmode=disable
```

## Database Setup

```sql
CREATE ROLE plugin_guest_pass WITH LOGIN PASSWORD '<chosen>';
CREATE SCHEMA guest_pass AUTHORIZATION plugin_guest_pass;
GRANT CONNECT ON DATABASE continuum TO plugin_guest_pass;
```

## Operations

- Put public guest-pass routes behind HTTPS.
- Set `public_base_url` when Continuum sits behind a reverse proxy and returned
  links need an absolute external origin.
- Keep audit retention aligned with your privacy policy.

## Build And Test

```bash
make build
make test
```
