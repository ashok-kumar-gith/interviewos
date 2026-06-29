#!/usr/bin/env bash
# InterviewOS — sudo-free local dev stack (no Docker required).
#
# Runs Go + Node from ~/.local and PostgreSQL + Redis compiled into ~/.local,
# with data dirs and ports isolated under the user's home. Intended for machines
# where Docker/Homebrew/sudo are unavailable.
#
# Usage:
#   scripts/dev-local.sh start-db     # start postgres + redis
#   scripts/dev-local.sh stop-db      # stop postgres + redis
#   scripts/dev-local.sh backend      # run the Go API (foreground)
#   scripts/dev-local.sh frontend     # run the Next.js dev server (foreground)
#   scripts/dev-local.sh status       # show what's running
#
# Prereqs: ~/.local/interviewos-env.sh exists (created during toolchain setup).

set -euo pipefail

ENV_FILE="$HOME/.local/interviewos-env.sh"
[ -f "$ENV_FILE" ] && source "$ENV_FILE"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PGBIN="$HOME/.local/pgsql/bin"
REDISBIN="$HOME/.local/redis/bin"
PGDATA="${PGDATA:-$HOME/.local/pgdata}"
PGPORT="${PGPORT:-5433}"
REDIS_PORT=6379
LOGDIR="$HOME/.local/run-logs"
mkdir -p "$LOGDIR"

export DATABASE_URL="postgres://interviewos@127.0.0.1:${PGPORT}/interviewos?sslmode=disable"
export REDIS_URL="redis://127.0.0.1:${REDIS_PORT}/0"
export ENV=development PORT=8080 LOG_LEVEL=debug JWT_SECRET="${JWT_SECRET:-dev-only-change-me}"
export CORS_ORIGINS="http://localhost:3000"

start_db() {
  "$PGBIN/pg_ctl" -D "$PGDATA" -o "-p ${PGPORT} -k /tmp" -l "$LOGDIR/postgres.log" start || true
  "$REDISBIN/redis-server" --port "$REDIS_PORT" --daemonize yes --dir /tmp --logfile "$LOGDIR/redis.log"
  sleep 1
  "$PGBIN/createdb" -p "$PGPORT" -h /tmp -U interviewos interviewos 2>/dev/null || true
  echo "postgres: $("$PGBIN/pg_isready" -p "$PGPORT" -h /tmp)"
  echo "redis: $("$REDISBIN/redis-cli" -p "$REDIS_PORT" ping)"
}

stop_db() {
  "$PGBIN/pg_ctl" -D "$PGDATA" stop || true
  "$REDISBIN/redis-cli" -p "$REDIS_PORT" shutdown nosave 2>/dev/null || true
  echo "stopped postgres + redis"
}

case "${1:-}" in
  start-db) start_db ;;
  stop-db)  stop_db ;;
  backend)  cd "$ROOT/backend" && exec go run ./cmd/api ;;
  migrate)  cd "$ROOT/backend" && exec go run ./cmd/migrate ;;
  seed)     cd "$ROOT/backend" && exec go run ./cmd/seed ;;
  frontend) cd "$ROOT/frontend" && rm -rf .next && exec npm run dev ;;
  status)
    echo "postgres: $("$PGBIN/pg_isready" -p "$PGPORT" -h /tmp 2>&1 || true)"
    echo "redis: $("$REDISBIN/redis-cli" -p "$REDIS_PORT" ping 2>&1 || true)"
    echo "backend: $(curl -sf -m 2 http://127.0.0.1:8080/readyz 2>/dev/null || echo down)"
    ;;
  *) echo "usage: $0 {start-db|stop-db|backend|frontend|migrate|seed|status}"; exit 1 ;;
esac
