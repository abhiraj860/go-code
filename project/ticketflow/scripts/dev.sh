#!/usr/bin/env bash
#
# Starts, stops and seeds the whole system for local testing.
#
# Process supervision by shell script is not how this runs in production --
# Phase 5 puts every service in a container and Kubernetes supervises them.
# This exists so the system can be driven end to end today, from one command,
# without a developer needing to know which eight processes to launch and in
# what order.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="${ROOT}/.run"
BIN_DIR="${RUN_DIR}/bin"
LOG_DIR="${RUN_DIR}/logs"

PG="postgres://ticketflow:ticketflow@localhost:5432"

# name:package pairs, in start order. Dependencies first, so a service is up
# before anything dials it.
GO_SERVICES=(
  "catalog-svc:./services/catalog-svc/cmd/catalog-svc"
  "inventory-svc:./services/inventory-svc/cmd/inventory-svc"
  "order-svc:./services/order-svc/cmd/order-svc"
  "ticket-svc:./services/ticket-svc/cmd/ticket-svc"
  "search-svc:./services/search-svc/cmd/search-svc"
  "gateway-bff:./services/gateway-bff/cmd/gateway-bff"
  "notification-svc:./services/notification-svc/cmd/notification-svc"
)

log()  { printf '\033[36m>>\033[0m %s\n' "$*"; }
warn() { printf '\033[33m!!\033[0m %s\n' "$*"; }
die()  { printf '\033[31mxx\033[0m %s\n' "$*" >&2; exit 1; }

env_for() {
  case "$1" in
    catalog-svc)   echo "CATALOG_DB_DSN=${PG}/ticketflow_catalog?sslmode=disable" ;;
    inventory-svc) echo "INVENTORY_DB_DSN=${PG}/ticketflow_inventory?sslmode=disable" ;;
    order-svc)     echo "ORDER_DB_DSN=${PG}/ticketflow_order?sslmode=disable" ;;
    *)             echo "" ;;
  esac
}

# ---------------------------------------------------------------- build

build() {
  mkdir -p "$BIN_DIR" "$LOG_DIR"
  log "building services"
  for entry in "${GO_SERVICES[@]}"; do
    local name="${entry%%:*}" pkg="${entry##*:}"
    go build -o "${BIN_DIR}/${name}" "$pkg" || die "building ${name} failed"
  done
  log "built ${#GO_SERVICES[@]} services into .run/bin"
}

# ---------------------------------------------------------------- start

start_one() {
  local name="$1" bin="${BIN_DIR}/$1" extra_env
  extra_env="$(env_for "$name")"

  if pgrep -x "$name" >/dev/null 2>&1; then
    warn "${name} is already running; leaving it alone"
    return 0
  fi

  # LOG_FORMAT=text because a human is about to read these.
  if [[ -n "$extra_env" ]]; then
    env "$extra_env" "$(printf '%s' "${name%%-*}" | tr '[:lower:]' '[:upper:]')_LOG_FORMAT=text" \
      setsid nohup "$bin" >"${LOG_DIR}/${name}.log" 2>&1 </dev/null &
  else
    setsid nohup "$bin" >"${LOG_DIR}/${name}.log" 2>&1 </dev/null &
  fi
  echo $! > "${RUN_DIR}/${name}.pid"
}

wait_healthy() {
  local name="$1" port="$2" tries=40
  while (( tries-- > 0 )); do
    if curl -sf --max-time 1 "http://localhost:${port}/healthz" >/dev/null 2>&1; then
      printf '   \033[32mok\033[0m   %-14s :%s\n' "$name" "$port"
      return 0
    fi
    sleep 0.5
  done
  printf '   \033[31mdown\033[0m %-14s :%s  (see .run/logs/%s.log)\n' "$name" "$port" "$name"
  return 1
}

start() {
  docker ps --filter "name=tf-postgres" --format '{{.Names}}' | grep -q tf-postgres \
    || die "the stack is not running -- run 'make up-all' first"

  build

  log "starting services"
  for entry in "${GO_SERVICES[@]}"; do
    start_one "${entry%%:*}"
  done

  # Node apps. Built artifacts only; no watch mode, so this matches what CI
  # builds rather than a dev-server approximation.
  if [[ -d "${ROOT}/node/realtime-gateway/dist" ]]; then
    (cd "${ROOT}/node/realtime-gateway" && setsid nohup node dist/server.js \
      >"${LOG_DIR}/realtime-gateway.log" 2>&1 </dev/null & echo $! > "${RUN_DIR}/realtime-gateway.pid")
  else
    warn "realtime-gateway is not built -- run 'make test-node' first"
  fi

  # Build the storefront if it has not been built. A first-time user should not
  # have to know that `make run` silently skips the UI without it.
  if [[ ! -d "${ROOT}/node/web/.next" ]]; then
    log "building the storefront (first run only)"
    (cd "${ROOT}/node/web" && npm run build >"${LOG_DIR}/web-build.log" 2>&1) \
      || warn "storefront build failed -- see .run/logs/web-build.log"
  fi
  if [[ -d "${ROOT}/node/web/.next" ]]; then
    (cd "${ROOT}/node/web" && setsid nohup npx next start -p 3000 \
      >"${LOG_DIR}/web.log" 2>&1 </dev/null & echo $! > "${RUN_DIR}/web.pid")
  fi

  echo
  local failed=0
  wait_healthy catalog-svc      9102 || failed=1
  wait_healthy inventory-svc    9112 || failed=1
  wait_healthy order-svc        9142 || failed=1
  wait_healthy ticket-svc       9122 || failed=1
  wait_healthy search-svc       9132 || failed=1
  wait_healthy gateway-bff      8080 || failed=1
  wait_healthy notification-svc 9162 || failed=1
  wait_healthy realtime-gateway 9150 || failed=1
  # The storefront has no /healthz; a 200 on the home page is the check.
  if curl -sf --max-time 10 http://localhost:3000/ >/dev/null 2>&1; then
    printf '   \033[32mok\033[0m   %-14s :%s\n' "web" "3000"
  else
    printf '   \033[31mdown\033[0m %-14s :%s  (see .run/logs/web.log)\n' "web" "3000"
    failed=1
  fi

  echo
  if (( failed )); then
    warn "some services did not come up; check .run/logs/"
  fi
  cat <<'BANNER'
   storefront   http://localhost:3000
   REST API     http://localhost:8080/v1/events
   search       http://localhost:9132/v1/search
   logs         .run/logs/

   next: make seed   (if you have not already)
         make demo   (scripted end-to-end walkthrough)
BANNER
}

# ---------------------------------------------------------------- stop

stop() {
  log "stopping services"
  for pidfile in "${RUN_DIR}"/*.pid; do
    [[ -e "$pidfile" ]] || continue
    local pid
    pid="$(cat "$pidfile")"
    kill "$pid" 2>/dev/null
    rm -f "$pidfile"
  done

  # Also stop anything started outside this script. Without it, a service left
  # running from an earlier session survives `stop`, and the next `start` sees
  # the port taken and silently keeps the OLD BINARY running -- so a rebuild
  # appears to do nothing and the bug you just fixed is still there.
  #
  # pgrep -x matches the executable name exactly, so it cannot match this
  # script's own command line the way `pkill -f` would.
  for entry in "${GO_SERVICES[@]}"; do
    local name="${entry%%:*}"
    if pgrep -x "$name" >/dev/null 2>&1; then
      pgrep -x "$name" | xargs -r kill 2>/dev/null
      printf '   stopped %s\n' "$name"
    fi
  done

  pgrep -f "node dist/server.js" | xargs -r kill 2>/dev/null
  pgrep -f "next start -p 3000" | xargs -r kill 2>/dev/null
  pgrep -x "next-server" | xargs -r kill 2>/dev/null

  sleep 1
  log "done"
}

# ---------------------------------------------------------------- seed

seed() {
  log "seeding catalog (postgres)"
  docker exec -i tf-postgres psql -U ticketflow -d ticketflow_catalog -q -v ON_ERROR_STOP=1 \
    < "${ROOT}/deploy/seed/catalog_seed.sql" || die "catalog seed failed"

  log "seeding content (mongo)"
  docker exec -i tf-mongo mongosh --quiet -u ticketflow -p ticketflow \
    --authenticationDatabase admin < "${ROOT}/deploy/seed/content_seed.js" >/dev/null \
    || warn "mongo seed failed (event pages will render without content)"

  # Inventory seats are derived from the catalog seat map rather than written
  # out, so the two cannot drift apart.
  log "seeding inventory seats"
  docker exec tf-postgres psql -U ticketflow -d ticketflow_inventory -q -c "
    INSERT INTO seat_allocation (event_id, seat_id, status)
    SELECT e, section||'-'||row_label||'-'||n, 1
    FROM (VALUES ('evt-arijit-mumbai'),('evt-coldplay-mumbai')) ev(e),
         (VALUES ('A'),('B'),('C')) s(section),
         (VALUES ('A'),('B'),('C'),('D')) r(row_label),
         generate_series(1,10) n
    ON CONFLICT DO NOTHING;" || die "inventory seed failed"

  log "indexing events into elasticsearch"
  for e in evt-arijit-mumbai evt-coldplay-mumbai evt-mi-vs-rcb evt-cancelled-demo; do
    printf '{"id":"evt_seed_%s","type":"catalog.event.updated","aggregate_id":"%s","occurred_at":"2026-01-01T00:00:00Z","schema_version":1,"payload":{"event_id":"%s","version":1}}\n' "$e" "$e" "$e"
  done | docker exec -i tf-kafka /opt/kafka/bin/kafka-console-producer.sh \
      --bootstrap-server localhost:9092 --topic catalog.event.updated 2>/dev/null

  sleep 3
  curl -sf -X POST http://localhost:9200/events/_refresh >/dev/null 2>&1 || true

  echo
  log "seeded. counts:"
  printf '   events    %s\n' "$(docker exec tf-postgres psql -U ticketflow -d ticketflow_catalog -tAc 'SELECT count(*) FROM event;')"
  printf '   seats     %s\n' "$(docker exec tf-postgres psql -U ticketflow -d ticketflow_inventory -tAc 'SELECT count(*) FROM seat_allocation;')"
  printf '   indexed   %s\n' "$(curl -sf 'http://localhost:9200/events/_count' 2>/dev/null | grep -oE '"count":[0-9]+' | cut -d: -f2 || echo '?')"
}

# ---------------------------------------------------------------- reset

reset() {
  log "resetting seat state and orders (keeps the catalogue)"
  docker exec tf-postgres psql -U ticketflow -d ticketflow_inventory -q \
    -c "UPDATE seat_allocation SET status=1, hold_id=NULL, hold_expires_at=NULL, order_id=NULL;" \
    -c "DELETE FROM seat_hold;"
  docker exec tf-postgres psql -U ticketflow -d ticketflow_order -q \
    -c "DELETE FROM outbox;" -c "DELETE FROM customer_order;" 2>/dev/null || true
  docker exec tf-redis redis-cli -n 1 FLUSHDB >/dev/null
  log "every seat is available again"
}

# probe reports whether something is actually answering on a port, which is the
# only claim that matters -- a process can exist and not be serving.
probe() {
  curl -sf --max-time 2 "http://localhost:$1$2" >/dev/null 2>&1 && echo running || echo stopped
}

status() {
  echo "containers:"
  docker ps --filter "name=tf-" --format '   {{.Names}}\t{{.Status}}' | sort
  echo
  echo "services:"
  for entry in "${GO_SERVICES[@]}"; do
    local name="${entry%%:*}"
    printf '   %-16s %s\n' "$name" "$(pgrep -x "$name" >/dev/null && echo running || echo stopped)"
  done
  printf '   %-16s %s\n' "realtime-gateway" "$(probe 9150 /healthz)"
  printf '   %-16s %s\n' "web" "$(probe 3000 /)"
}

case "${1:-}" in
  start)  start ;;
  stop)   stop ;;
  seed)   seed ;;
  reset)  reset ;;
  status) status ;;
  *) die "usage: dev.sh {start|stop|seed|reset|status}" ;;
esac
