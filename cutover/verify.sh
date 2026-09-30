#!/usr/bin/env bash
# Compare row counts and sequences between two databases.
set -euo pipefail

SOURCE="${1:-}"
TARGET="${2:-}"

if [[ -z "$SOURCE" || -z "$TARGET" ]]; then
  echo "usage: $0 <source-db> <target-db>" >&2
  exit 1
fi

query() {
  psql -tA -c "$1" "$2"
}

tables_query="
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
ORDER BY table_name;"

mapfile -t TABLES < <(query "$tables_query" "$SOURCE")

mismatches=0
checked=0

for table in "${TABLES[@]}"; do
  src_count=$(query "SELECT count(*) FROM \"$table\";" "$SOURCE")
  dst_count=$(query "SELECT count(*) FROM \"$table\";" "$TARGET")
  checked=$((checked + 1))
  if [[ "$src_count" != "$dst_count" ]]; then
    echo "MISMATCH $table: source=$src_count target=$dst_count"
    mismatches=$((mismatches + 1))
  fi
done

echo "checked $checked tables, $mismatches mismatches"

if [[ "$mismatches" -gt 0 ]]; then
  exit 1
fi
