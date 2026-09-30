# Talon Operations Runbook

## Service management

```bash
systemctl start talon
systemctl stop talon
systemctl status talon
journalctl -u talon -f
```

Direct invocation:

```bash
talon \
  -host 0.0.0.0 -port 8000 \
  -database mydb \
  -uri "postgres://talon:secret@localhost/mydb?sslmode=require" \
  -data-dir /var/lib/talon \
  -web-dir /opt/talon/web \
  -admin-login admin -admin-password "$TALON_ADMIN_PASSWORD"
```

Flags:

| Flag | Purpose |
|------|---------|
| `-host`, `-port` | Listen address |
| `-database` | Database name used in RPC paths |
| `-uri` | PostgreSQL connection string (also used for LISTEN/NOTIFY) |
| `-data-dir` | Filestore root (`{data-dir}/{database}/ab/cd/<sha256>`) |
| `-web-dir` | Built web client (`sao-ts/dist`) served with gzip/cache headers |
| `-admin-login`, `-admin-password` | Upsert an administrator on startup |
| `-version` | Print version |

## Health checks

- `GET /` returns the web client (or a version string without `-web-dir`).
- JSON-RPC: `POST /{db}/rpc/` with `{"method":"common.server.version"}`.
- Database: `pg_isready`, then `SELECT count(*) FROM pg_stat_activity`.
- Bus: clients long-poll `POST /{db}/bus`; verify PostgreSQL notifications:
  ```sql
  SELECT pg_notify('tryton_bus', '{"ping":true}');
  ```

## Database operations

- Schema sync runs on startup and is additive (`CREATE TABLE IF NOT EXISTS`,
  `ADD COLUMN IF NOT EXISTS`).
- Never edit `talon`-created tables directly while the service is running;
  use RPC.
- Backups:
  ```bash
  ./cutover/snapshot.sh mydb /var/backups/tryton
  ```
- Restores use plain `pg_restore`; Talon requires no special preparation.

## Filestore

- Content-addressed by SHA-256, so duplicates are stored once.
- Back up `-data-dir` alongside database dumps for a consistent restore.
- Deleted attachments leave files behind until a future garbage collection
  sweep; disk use is bounded by unique content.

## Scheduler (`ir_cron`)

- Due jobs are rows in `ir_cron` with `active` and `next_call <= now()`.
- Talon's cron package exposes `Scheduler.Run`; embed it in the process or run
  `RunDue` from an external timer until the worker pool lands.
- Investigate with:
  ```sql
  SELECT id, name, model, function, next_call, number_calls
  FROM ir_cron ORDER BY next_call;
  ```

## Sessions and security

- Sessions live in memory; restarting Talon logs everyone out.
- Login attempts are rate limited (5 failures / 5 minutes / login).
- Access rules come from `ir_model_access` / `ir_model_field_access`;
  administrators (`res_user.administrator = true`) bypass checks.
- Bootstrap or reset the administrator:
  ```bash
  talon -database mydb -uri "$URI" -admin-password 'new-secret'
  ```

## Troubleshooting

| Symptom | Checks |
|---------|--------|
| `403` on model methods | Session cookie expired; login again. Check access rules. |
| `access denied` | Missing `ir_model_access` row for the user's group. |
| `too many login attempts` | Wait 5 minutes or clear `attempts` by restarting. |
| `no database connection` | `-uri` wrong or PostgreSQL down. |
| Bus never receives | PostgreSQL notifications are per-database; ensure the listener connects to the same DB. |
| Numeric/date values look odd | Ensure clients use tagged codecs (`__class__`). |
