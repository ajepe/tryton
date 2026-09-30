#!/usr/bin/env bash
# Take a consistent snapshot of a Tryton/Talon database.
set -euo pipefail

DB="${1:-}"
OUTDIR="${2:-.}"

if [[ -z "$DB" ]]; then
  echo "usage: $0 <database> [output-dir]" >&2
  exit 1
fi

mkdir -p "$OUTDIR"
STAMP="$(date +%Y%m%d%H%M%S)"
FILE="$OUTDIR/${DB}_${STAMP}.dump"

pg_dump --format=custom --compress=9 --file "$FILE" "$DB"
echo "snapshot written to $FILE"
