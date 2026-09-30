# Talon Rollback Procedure

Use this procedure when Talon must be rolled back to Python `trytond`.

## Triggers

- Data corruption or unexpected schema changes.
- Unrecoverable RPC or workflow failures.
- Performance regression beyond the agreed SLO.
- Security incident in the Talon deployment.

## Rollback window

Keep the Python deployment and a pre-cutover database snapshot available for
the agreed rollback window (recommended: 7 days).

## Steps

1. **Stop Talon** to prevent new writes:
   ```bash
   systemctl stop talon
   ```
2. **Freeze the Talon database** (preserve evidence):
   ```bash
   pg_dump -Fc mydb > /var/backups/tryton/mydb_talon_$(date +%Y%m%d%H%M).dump
   ```
3. **Decide the recovery point**:
   - If Talon wrote no data after cutover, restore the pre-cutover snapshot.
   - If Talon wrote data, capture the delta first so it can be replayed:
     ```bash
     ./cutover/verify.sh mydb_precutover mydb > /tmp/delta.txt
     ```
     Rows created by Talon are identified by `create_date >= <cutover time>` on
     each table.
4. **Restore the Python-compatible database**:
   ```bash
   dropdb mydb
   createdb mydb
   pg_restore -d mydb /var/backups/tryton/mydb_precutover.dump
   ```
5. **Start Python `trytond`**:
   ```bash
   systemctl start trytond
   ```
6. **Verify**:
   ```bash
   trytond -d mydb -c /etc/trytond.conf --test  # optional smoke
   curl -fsS http://localhost:8000/ >/dev/null
   ```
7. **Replay Talon-era data** if required, using the delta captured in step 3.
   Prefer re-entering through `trytond` over direct SQL when constraints and
   workflow side effects matter.
8. **Re-enable traffic** and monitor for errors.
9. **Postmortem**: record trigger, timeline, recovery point and follow-ups.

## Notes

- Talon does not alter core table structures, so restoring a pre-cutover dump
  is sufficient.
- If the filestore was populated by Talon, copy `-data-dir/<database>` back to
  the Python filestore root (it uses the same SHA-256 layout).
- Session cookies are not transferable between servers; clients re-login.
