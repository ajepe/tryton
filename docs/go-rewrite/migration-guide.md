# Talon Migration Guide

This guide covers moving a database from Python `trytond` to the Talon Go
server with minimal downtime and no data loss.

## 1. Pre-flight checks

1. Record the running versions:
   ```bash
   trytond --version
   psql -c "SELECT name, state, latest_version FROM ir_module_module ORDER BY name"
   ```
2. Audit installed modules. Talon ships core module support; third-party
   modules must be ported before cutover:
   ```sql
   SELECT name FROM ir_module_module WHERE state = 'activated';
   ```
3. Take a full backup:
   ```bash
   ./cutover/snapshot.sh mydb /var/backups/tryton
   ```
4. Confirm the PostgreSQL server version is supported (13+).
5. Run the compatibility contract checks in `talon/docs/compatibility-contract.md`.

## 2. Staging verification

> **Important:** install and activate every Python module **before** starting
> Talon against the database. Talon's schema sync creates tables for its ported
> models; if it runs first, later Python module activation can fail because the
> table already exists with Talon's simplified schema. Talon never drops
> tables, so install modules first.

1. Restore a production copy into a staging database.
2. Start Talon against the staging database:
   ```bash
   talon -database mydb_staging \
         -uri "postgres:///mydb_staging?host=/var/run/postgresql" \
         -data-dir /var/lib/talon \
         -web-dir /opt/talon/web \
         -admin-password "$(openssl rand -base64 24)"
   ```
3. Run the conformance suite against Python `trytond` and Talon:
   ```bash
   cd talon
   go run ./cmd/conformance \
     -python http://staging-python:8000 \
     -go http://staging-talon:8001 \
     -db mydb_staging \
     -login admin -password "$TALON_ADMIN_PASSWORD" \
     -report /tmp/conformance.json
   ```
4. Compare row counts and sequences:
   ```bash
   ./cutover/verify.sh mydb_staging mydb_talon_check
   ```
5. Exercise workflows manually: login, menus, list views, forms, on_change,
   buttons, attachments, reports.

## 3. Cutover window

1. Announce maintenance and stop application traffic.
2. Stop Python `trytond` workers.
3. Wait for in-flight transactions to finish:
   ```sql
   SELECT count(*) FROM pg_stat_activity
   WHERE datname = 'mydb' AND state <> 'idle';
   ```
4. Take the final backup with `cutover/snapshot.sh`.
5. Start Talon against the production database:
   ```bash
   systemctl start talon
   ```
6. Verify health:
   ```bash
   curl -fsS http://localhost:8000/ >/dev/null
   ./cutover/verify_rpc.sh http://localhost:8000 mydb admin "$TALON_ADMIN_PASSWORD"
   ```
7. Re-enable traffic (reverse proxy / load balancer).
8. Keep Python `trytond` installed but stopped for the rollback window.

## 4. Post-cutover

- Watch logs for `access denied`, `ConcurrencyException` and RPC errors.
- Confirm bus notifications reach clients.
- Confirm the scheduler runs due jobs (`ir_cron` next_call advances).
- Verify filestore growth in `-data-dir` and that attachments open.
- After the rollback window closes, remove the Python deployment.

## 5. Compatibility notes

- The JSON-RPC protocol, tagged codecs, context handling and error shapes are
  unchanged, so existing clients keep working.
- Tryton 8 uses Argon2id hashes in `res_user.password_hash`; Talon reads and
  writes the same format (bcrypt is still accepted for older databases).
- Tryton 8 search calls take `(domain, offset, limit, order, fields)` for
  `search_read` and an order list of `[field, "ASC"|"DESC"]` pairs; Talon
  accepts both the 7.x and 8.x forms.
- Common methods (`common.server.version`, `common.db.list`,
  `common.authentication.services`) are served from `/rpc/`; `common.db.login`
  and `common.db.logout` from `/{db}/rpc/`.
- Talon reads and writes the same tables; no schema migration is required for
  core models.
- `_timestamp` optimistic concurrency is preserved for writes.
