#!/usr/bin/env bash
# Smoke-test a running Talon server over JSON-RPC.
set -euo pipefail

BASE_URL="${1:-http://localhost:8000}"
DB="${2:-}"
ADMIN_LOGIN="${3:-admin}"
ADMIN_PASSWORD="${4:-}"

if [[ -z "$DB" ]]; then
  echo "usage: $0 <base-url> <database> [login] [password]" >&2
  exit 1
fi

COOKIE_JAR="$(mktemp)"
trap 'rm -f "$COOKIE_JAR"' EXIT

rpc() {
  curl -fsS -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
    -X POST "$BASE_URL/$DB/rpc/" \
    -H 'Content-Type: application/json' \
    -d "$1"
}

echo "== version"
rpc '{"id":1,"method":"common.server.version","params":[]}'
echo

echo "== db list"
rpc '{"id":2,"method":"common.db.list","params":[]}'
echo

if [[ -n "$ADMIN_PASSWORD" ]]; then
  echo "== login"
  curl -fsS -c "$COOKIE_JAR" \
    -X POST "$BASE_URL/$DB/session/login" \
    -H 'Content-Type: application/json' \
    -d "{\"id\":0,\"method\":\"common.db.login\",\"params\":[\"$ADMIN_LOGIN\",{\"password\":\"$ADMIN_PASSWORD\"},\"en\"]}"
  echo

  echo "== fields_get party.party"
  rpc '{"id":3,"method":"model.party.party.fields_get","params":[[],{}]}' >/dev/null
  echo "ok"

  echo "== search party.party"
  rpc '{"id":4,"method":"model.party.party.search","params":[[],0,10,"name ASC",{}]}'
  echo

  echo "== logout"
  curl -fsS -b "$COOKIE_JAR" -X POST "$BASE_URL/$DB/session/logout" \
    -H 'Content-Type: application/json' -d '{}'
  echo
fi

echo "smoke test passed"
