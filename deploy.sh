#!/usr/bin/env bash
set -Eeuo pipefail

APP_NAME="${APP_NAME:-api-stations-and-radio-link}"
PM2_APP_NAME="${PM2_APP_NAME:-${SERVICE_NAME:-$APP_NAME}}"
DEPLOY_DIR="${DEPLOY_DIR:-/root/app/api-stations-and-radio-link}"
BIN_SOURCE="${BIN_SOURCE:-$DEPLOY_DIR/incoming/api}"
BINARY_NAME="${BINARY_NAME:-api}"
HEALTHCHECK_URL="${HEALTHCHECK_URL-http://127.0.0.1:8080/api/v1/healthz}"
STATUS_URL="${STATUS_URL-http://127.0.0.1:8080/api/v1/status}"
HEALTHCHECK_RETRIES="${HEALTHCHECK_RETRIES:-15}"
HEALTHCHECK_INTERVAL="${HEALTHCHECK_INTERVAL:-2}"
KEEP_RELEASES="${KEEP_RELEASES:-5}"
RELEASE_ID="${RELEASE_ID:-$(date -u +%Y%m%d%H%M%S)}"

RELEASES_DIR="$DEPLOY_DIR/releases"
CURRENT_FILE="$DEPLOY_DIR/current_release"
RELEASE_DIR="$RELEASES_DIR/$RELEASE_ID"
BINARY_TARGET="$DEPLOY_DIR/$BINARY_NAME"
ECOSYSTEM_FILE="$DEPLOY_DIR/ecosystem.config.cjs"

log() {
  printf '[deploy] %s\n' "$*"
}

die() {
  printf '[deploy] ERROR: %s\n' "$*" >&2
  exit 1
}

case "$KEEP_RELEASES" in
  '' | *[!0-9]*) die "KEEP_RELEASES must be a positive integer" ;;
esac
[ "$KEEP_RELEASES" -gt 0 ] || die "KEEP_RELEASES must be greater than zero"

case "$HEALTHCHECK_RETRIES" in
  '' | *[!0-9]*) die "HEALTHCHECK_RETRIES must be a positive integer" ;;
esac
[ "$HEALTHCHECK_RETRIES" -gt 0 ] || die "HEALTHCHECK_RETRIES must be greater than zero"

resolve_pm2() {
  if [ -n "${PM2_BIN:-}" ]; then
    [ -x "$PM2_BIN" ] || die "PM2_BIN is not executable: $PM2_BIN"
    printf '%s\n' "$PM2_BIN"
    return 0
  fi

  if command -v pm2 >/dev/null 2>&1; then
    command -v pm2
    return 0
  fi

  local candidate
  for candidate in \
    "$HOME/.nvm/versions/node"/*/bin/pm2 \
    "$HOME/.local/bin/pm2" \
    "$HOME/.npm-global/bin/pm2" \
    "/usr/local/bin/pm2" \
    "/usr/bin/pm2"; do
    if [ -x "$candidate" ]; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done

  return 1
}

PM2_BIN="$(resolve_pm2)" || die "pm2 not found; set PM2_BIN=/absolute/path/to/pm2"
export PATH="$(dirname "$PM2_BIN"):$PATH"
log "using pm2 at $PM2_BIN"

[ -f "$BIN_SOURCE" ] || die "binary not found: $BIN_SOURCE"

if [ -e "$RELEASE_DIR" ]; then
  die "release already exists: $RELEASE_DIR"
fi

log "deploying $APP_NAME release $RELEASE_ID"
mkdir -p "$RELEASES_DIR" "$RELEASE_DIR" "$DEPLOY_DIR/logs"
install -m 0755 "$BIN_SOURCE" "$RELEASE_DIR/$BINARY_NAME"

if [ -f "$DEPLOY_DIR/incoming/ecosystem.config.cjs" ]; then
  install -m 0644 "$DEPLOY_DIR/incoming/ecosystem.config.cjs" "$ECOSYSTEM_FILE"
fi
[ -f "$ECOSYSTEM_FILE" ] || die "PM2 config not found: $ECOSYSTEM_FILE"

set_current() {
  printf '%s\n' "$1" > "$CURRENT_FILE.tmp"
  mv -f "$CURRENT_FILE.tmp" "$CURRENT_FILE"
}

activate_binary() {
  install -m 0755 "$1" "$BINARY_TARGET.new"
  mv -f "$BINARY_TARGET.new" "$BINARY_TARGET"
}

start_or_restart_pm2() {
  DEPLOY_DIR="$DEPLOY_DIR" PM2_APP_NAME="$PM2_APP_NAME" \
    "$PM2_BIN" startOrReload "$ECOSYSTEM_FILE" --update-env
  "$PM2_BIN" save || log "warning: pm2 save failed (process list not persisted)"
}

restart_pm2() {
  "$PM2_BIN" restart "$PM2_APP_NAME" --update-env
  "$PM2_BIN" save || log "warning: pm2 save failed (process list not persisted)"
}

http_status() {
  local url="$1" code="" rest authority path host port

  if command -v curl >/dev/null 2>&1; then
    code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 "$url" 2>/dev/null || true)"
    printf '%s' "${code:-000}"
    return 0
  fi

  if command -v wget >/dev/null 2>&1; then
    code="$(wget -q -S -O /dev/null --timeout=5 "$url" 2>&1 | awk '/HTTP\// {print $2; exit}')"
    if [ -n "$code" ]; then
      printf '%s' "$code"
      return 0
    fi
  fi

  # Last resort for plain HTTP: speak HTTP/1.0 over bash's /dev/tcp.
  if [ "${url#http://}" != "$url" ]; then
    rest="${url#http://}"
    authority="${rest%%/*}"
    if [ "$rest" = "$authority" ]; then
      path="/"
    else
      path="/${rest#*/}"
    fi
    host="${authority%%:*}"
    port="${authority##*:}"
    if [ "$port" = "$authority" ]; then
      port="80"
    fi

    if exec 3<>"/dev/tcp/$host/$port" 2>/dev/null; then
      printf 'GET %s HTTP/1.0\r\nHost: %s\r\nConnection: close\r\n\r\n' "$path" "$host" >&3
      IFS=' ' read -r -t 5 _ code _ <&3 || true
      exec 3<&- 2>/dev/null || true
      printf '%s' "${code:-000}"
      return 0
    fi
  fi

  printf '000'
}

wait_for_health() {
  local attempt code

  if [ -z "$HEALTHCHECK_URL" ]; then
    log "healthcheck disabled"
    return 0
  fi

  for attempt in $(seq 1 "$HEALTHCHECK_RETRIES"); do
    code="$(http_status "$HEALTHCHECK_URL")"

    # /healthz returns 200 independently of the database. A 503 from /status is
    # also accepted for compatibility: it means the process is up but the
    # database is unavailable.
    if [ -n "$code" ] && { { [ "$code" -ge 200 ] && [ "$code" -lt 500 ]; } || [ "$code" -eq 503 ]; }; then
      log "healthcheck OK (HTTP $code)"
      return 0
    fi

    log "healthcheck attempt $attempt/$HEALTHCHECK_RETRIES failed (HTTP ${code:-000})"
    sleep "$HEALTHCHECK_INTERVAL"
  done

  return 1
}

PREVIOUS_RELEASE=""
if [ -f "$CURRENT_FILE" ]; then
  PREVIOUS_RELEASE="$(cat "$CURRENT_FILE")"
fi

rollback() {
  if [ -n "$PREVIOUS_RELEASE" ] && [ -d "$PREVIOUS_RELEASE" ] && [ -f "$PREVIOUS_RELEASE/$BINARY_NAME" ]; then
    log "rolling back to $PREVIOUS_RELEASE"
    activate_binary "$PREVIOUS_RELEASE/$BINARY_NAME"
    set_current "$PREVIOUS_RELEASE"
    restart_pm2 || log "rollback restart failed"
  else
    log "no previous release available; stopping PM2 app"
    rm -f "$CURRENT_FILE"
    "$PM2_BIN" delete "$PM2_APP_NAME" >/dev/null 2>&1 || true
    "$PM2_BIN" save >/dev/null 2>&1 || true
  fi

  if [ -d "$RELEASE_DIR" ] && [ "$RELEASE_DIR" != "$PREVIOUS_RELEASE" ]; then
    log "removing failed release $RELEASE_DIR"
    rm -rf -- "$RELEASE_DIR"
  fi
}

log "activating release $RELEASE_ID"
activate_binary "$RELEASE_DIR/$BINARY_NAME"
set_current "$RELEASE_DIR"

if ! start_or_restart_pm2; then
  log "PM2 start/reload failed"
  rollback
  die "deployment failed while starting PM2"
fi

if ! wait_for_health; then
  log "healthcheck failed"
  rollback
  die "deployment failed healthcheck; rolled back"
fi

if [ -n "$STATUS_URL" ]; then
  status_code="$(http_status "$STATUS_URL")"
  if [ "$status_code" != "200" ]; then
    log "warning: $STATUS_URL returned HTTP ${status_code:-000}; check database connectivity (.env and MySQL)"
  fi
fi

log "release $RELEASE_ID is live"

# Keep only the most recent releases, never deleting the active one.
mapfile -t releases < <(ls -1dt "$RELEASES_DIR"/* 2>/dev/null || true)
if [ "${#releases[@]}" -gt "$KEEP_RELEASES" ]; then
  current_target="$(cat "$CURRENT_FILE" 2>/dev/null || true)"

  for old in "${releases[@]:$KEEP_RELEASES}"; do
    if [ "$old" != "$current_target" ]; then
      log "removing old release $old"
      rm -rf -- "$old"
    fi
  done
fi

log "deploy complete"
