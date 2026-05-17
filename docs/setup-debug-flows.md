# Guest Pass Setup, Debugging, And Flows

Plugin ID: `continuum.guest-pass`
Version documented: `0.1.0`

## Purpose

temporary public-link system for scoped media access.

## Runtime Dependencies

- Continuum plugin host
- Postgres schema for this plugin
- A public_base_url that reaches Continuum/plugin routes

## Setup Checklist

1. Create schema and configure database_url.
2. Set public_base_url to the externally reachable URL.
3. Install the plugin and open the admin route.
4. Create a short-lived test pass for a non-sensitive item.
5. Open the public /p/* link in a private browser session.

## Configuration Reference

- `database_url`
- `public_base_url`
- `audit_retention_days`

Use the plugin manifest/admin form as the source of truth for field validation and defaults. Keep database credentials scoped to the plugin schema unless a plugin explicitly needs read access to Continuum core tables.

## Exposed Routes

- `* /api/admin/* [admin]`
- `* /api/public/* [public]`
- `GET /assets/* [public]`
- `GET /p/* [public]`
- `GET /admin/* [admin]`

## Capabilities

- `http_routes.v1 (guest-pass) - Temporary public links for tightly scoped media access.`
- `scheduled_task.v1 (maintenance) - Prunes old guest-pass audit events.`

## Operational Flows

### Pass access

1. Admin creates a scoped pass.
2. The plugin stores token, scope, expiry, and audit state.
3. Guest opens /p/* without Continuum auth.
4. The plugin validates token/scope/expiry and serves only the allowed public surface.
5. Maintenance task expires old/audited records.

## How This Plugin Communicates

- Does not require another plugin for core operation.
- Reads/writes its own pass database.
- Uses public plugin routes for guests and admin routes for operators.

## Debugging Runbook

- If links 404 behind a proxy, verify public_base_url and route forwarding for /p/* and /api/public/*.
- If links never expire, check the maintenance scheduled task.
- If public access is broader than expected, review pass scope before sharing.
- Use audit retention settings to keep enough history for support without retaining forever.

## Log And Health Checks

- Start with Continuum Admin -> Plugins and confirm the installation is enabled.
- Check the plugin process logs around startup for manifest loading, migration, and route registration.
- Check scheduled task logs when a workflow depends on polling or reconciliation.
- Confirm the plugin routes are reachable through Continuum using the access level shown above.
- For database-backed plugins, verify the configured role can connect, create/migrate tables in its schema, and read/write expected rows.

## Common Failure Patterns

- Wrong installation ID selected in a portal or router setting after reinstalling a plugin.
- Plugin database URL points at the public schema instead of the dedicated plugin schema.
- Reverse proxy forwards the SPA route but not `/api/*`, `/api/v1/*`, `/assets/*`, or provider-specific public routes.
- Network checks are run from the operator laptop instead of from the Continuum/plugin runtime network.
- Secrets are regenerated during restart, invalidating signed URLs, encrypted fields, or login state.

## Verification After Changes

1. Restart or reload the plugin installation.
2. Open the plugin route or admin page in Continuum.
3. Exercise the smallest workflow that crosses a plugin boundary.
4. Confirm both the source plugin and destination plugin record the same request/session/login identifier.
5. Leave the scheduled reconciler enough time to run, then confirm terminal state or a useful error.
