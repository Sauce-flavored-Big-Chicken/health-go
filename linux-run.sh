#!/usr/bin/env bash
set -Eeuo pipefail

# One-file Linux bootstrap script for health-go.
# Usage:
#   curl -fsSL <RAW_SCRIPT_URL> | bash
# Optional env vars:
#   REPO_URL, BRANCH, APP_DIR

REPO_URL="${REPO_URL:-https://github.com/Sauce-flavored-Big-Chicken/health-go.git}"
BRANCH="${BRANCH:-main}"
APP_DIR="${APP_DIR:-$HOME/health-go}"
BIN_PATH="$APP_DIR/server"
LOG_DIR="$APP_DIR/logs"
LOG_FILE="$LOG_DIR/server.log"
PID_FILE="$APP_DIR/.server.pid"

log() {
  printf '[health-go] %s\n' "$*"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log "missing required command: $1"
    exit 1
  fi
}

stop_old_process() {
  if [[ -f "$PID_FILE" ]]; then
    old_pid="$(cat "$PID_FILE" 2>/dev/null || true)"
    if [[ -n "${old_pid:-}" ]] && kill -0 "$old_pid" 2>/dev/null; then
      log "stopping old process (pid=$old_pid)"
      kill "$old_pid" 2>/dev/null || true
      sleep 1
      if kill -0 "$old_pid" 2>/dev/null; then
        kill -9 "$old_pid" 2>/dev/null || true
      fi
    fi
    rm -f "$PID_FILE"
  fi

  # Extra safety: stop any previous server from the same APP_DIR.
  pkill -f "^$BIN_PATH$" 2>/dev/null || true
}

backup_local_file() {
  local src="$1"
  local dst_var="$2"
  if [[ -f "$src" ]]; then
    local tmp
    tmp="$(mktemp)"
    cp "$src" "$tmp"
    printf -v "$dst_var" '%s' "$tmp"
  fi
}

restore_local_file() {
  local backup="$1"
  local target="$2"
  if [[ -n "${backup:-}" ]] && [[ -f "$backup" ]]; then
    cp "$backup" "$target"
    rm -f "$backup"
  fi
}

main() {
  require_cmd git
  require_cmd go
  require_cmd nohup

  mkdir -p "$APP_DIR"

  if [[ ! -d "$APP_DIR/.git" ]]; then
    log "cloning $REPO_URL ($BRANCH) into $APP_DIR"
    git clone --branch "$BRANCH" "$REPO_URL" "$APP_DIR"
  fi

  cd "$APP_DIR"

  log "syncing latest code from origin/$BRANCH"
  git remote set-url origin "$REPO_URL"

  local env_backup=""
  local db_backup=""
  backup_local_file ".env" env_backup
  backup_local_file "health.db" db_backup

  git fetch origin "$BRANCH" --prune
  git checkout "$BRANCH"
  git reset --hard "origin/$BRANCH"

  restore_local_file "$env_backup" ".env"
  restore_local_file "$db_backup" "health.db"

  log "downloading dependencies"
  go mod download

  log "building server binary"
  go build -o "$BIN_PATH" ./cmd/server

  mkdir -p "$LOG_DIR"
  stop_old_process

  log "starting server in background"
  nohup "$BIN_PATH" >>"$LOG_FILE" 2>&1 &
  new_pid=$!
  echo "$new_pid" > "$PID_FILE"

  sleep 1
  if kill -0 "$new_pid" 2>/dev/null; then
    log "server started (pid=$new_pid)"
    log "log file: $LOG_FILE"
    log "tail logs: tail -f $LOG_FILE"
  else
    log "server failed to start, recent logs:"
    tail -n 80 "$LOG_FILE" || true
    exit 1
  fi
}

main "$@"
