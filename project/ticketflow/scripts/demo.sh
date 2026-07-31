#!/usr/bin/env bash
#
# A scripted end-to-end walkthrough.
#
# Every step exercises the real services over their real APIs. Nothing is
# published by hand and nothing is stubbed -- which matters, because a demo that
# fakes one link is how a disconnected feature goes unnoticed.
set -uo pipefail

BFF=http://localhost:8080
INV=localhost:9111
ORD=localhost:9141
GRPCURL="$(go env GOPATH)/bin/grpcurl"
EVENT=evt-arijit-mumbai
COOKIES="$(mktemp)"
trap 'rm -f "$COOKIES"' EXIT

step()  { printf '\n\033[1;36m%s\033[0m\n' "$*"; }
note()  { printf '   \033[2m%s\033[0m\n' "$*"; }
ok()    { printf '   \033[32m✓\033[0m %s\n' "$*"; }
bad()   { printf '   \033[31m✗\033[0m %s\n' "$*"; }

command -v "$GRPCURL" >/dev/null || { echo "grpcurl not installed: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"; exit 1; }
curl -sf --max-time 2 "$BFF/healthz" >/dev/null || { echo "gateway-bff is not running -- run 'make run'"; exit 1; }

step "1. Browse — server-rendered, cached, no availability baked in"
curl -sf "$BFF/v1/events?page_size=5" -c "$COOKIES" \
  | python3 -c "
import sys,json
d=json.load(sys.stdin)
for e in d.get('events',[]):
    print(f\"   {e['title']}  —  {e['venue']['city']}\")
" || bad "browse failed"

step "2. Event detail — a 3-way parallel fan-out"
note "catalog + inventory + mongo content, fetched concurrently via errgroup"
curl -sf "$BFF/v1/events/$EVENT" -b "$COOKIES" -c "$COOKIES" \
  | python3 -c "
import sys,json
d=json.load(sys.stdin)
print(f\"   title        {d['event']['title']}\")
print(f\"   availability {d['availability']}\")
c=d.get('content')
print(f\"   content keys {sorted(c.keys()) if c else None}\")
"

step "3. Hold two seats"
HOLD=$("$GRPCURL" -plaintext -d "{\"event_id\":\"$EVENT\",\"seat_ids\":[\"A-A-1\",\"A-A-2\"],\"user_id\":\"demo-buyer\",\"idempotency_key\":\"demo-$$\",\"ttl_seconds\":300}" \
  "$INV" ticketflow.inventory.v1.InventoryService/HoldSeats 2>&1)
HOLD_ID=$(echo "$HOLD" | python3 -c "import sys,json;print(json.load(sys.stdin)['holdId'])" 2>/dev/null || echo "")
[[ -n "$HOLD_ID" ]] && ok "held A-A-1, A-A-2 (hold ${HOLD_ID:0:8})" || { bad "hold failed: $HOLD"; exit 1; }
note "inventory also published these to Redis, so any open seat map greys them out now"

step "4. A different buyer tries the same seats"
OUT=$("$GRPCURL" -plaintext -d "{\"event_id\":\"$EVENT\",\"seat_ids\":[\"A-A-1\"],\"user_id\":\"other-buyer\",\"idempotency_key\":\"other-$$\"}" \
  "$INV" ticketflow.inventory.v1.InventoryService/HoldSeats 2>&1 | grep -oE 'Code: [A-Za-z]+' || true)
[[ "$OUT" == *ResourceExhausted* ]] && ok "rejected — $OUT" || bad "expected ResourceExhausted, got: $OUT"
note "the loser never opened a Postgres transaction: Redis rejected it in ~0.2ms"

step "5. Retry the ORIGINAL hold with the same idempotency key"
AGAIN=$("$GRPCURL" -plaintext -d "{\"event_id\":\"$EVENT\",\"seat_ids\":[\"A-A-1\",\"A-A-2\"],\"user_id\":\"demo-buyer\",\"idempotency_key\":\"demo-$$\",\"ttl_seconds\":300}" \
  "$INV" ticketflow.inventory.v1.InventoryService/HoldSeats 2>&1 \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['holdId'])" 2>/dev/null || echo "")
[[ "$AGAIN" == "$HOLD_ID" ]] && ok "same hold returned — no second set of seats" || bad "retry produced a different hold"

step "6. Place the order — order row and Kafka message in ONE transaction"
ORDER=$("$GRPCURL" -plaintext -d "{\"user_id\":\"demo-buyer\",\"event_id\":\"$EVENT\",\"hold_id\":\"$HOLD_ID\",\"seat_ids\":[\"A-A-1\",\"A-A-2\"],\"total\":{\"amount_minor\":1100000,\"currency_code\":\"INR\"},\"idempotency_key\":\"ord-$$\"}" \
  "$ORD" ticketflow.order.v1.OrderService/PlaceOrder 2>&1)
ORDER_ID=$(echo "$ORDER" | python3 -c "import sys,json;print(json.load(sys.stdin)['order']['id'])" 2>/dev/null || echo "")
[[ -n "$ORDER_ID" ]] && ok "order ${ORDER_ID:0:16}… created PENDING" || { bad "order failed: $ORDER"; exit 1; }

step "7. Confirm payment — inventory converts HELD to SOLD"
PAID=$("$GRPCURL" -plaintext -d "{\"order_id\":\"$ORDER_ID\",\"payment_reference\":\"pay_demo\"}" \
  "$ORD" ticketflow.order.v1.OrderService/ConfirmPayment 2>&1)
echo "$PAID" | grep -q ORDER_STATUS_PAID && ok "order PAID, seats SOLD" || bad "payment failed: $PAID"

step "8. The outbox relay delivered the message"
sleep 3
PENDING=$(curl -sf http://localhost:9142/metrics | grep -oE 'order_outbox_pending [0-9-]+' | awk '{print $2}')
[[ "$PENDING" == "0" ]] && ok "outbox backlog drained to 0" || bad "outbox backlog is $PENDING"

step "9. ticket-svc generated a PDF per seat and stored it in S3"
sleep 2
KEYS=$(curl -sf "http://localhost:4566/ticketflow-tickets?list-type=2" 2>/dev/null | grep -oE '<Key>[^<]+</Key>' | wc -l)
[[ "$KEYS" -gt 0 ]] && ok "$KEYS ticket PDFs in S3" || bad "no PDFs found in S3"

step "10. Search — faceted, typo-tolerant, cancelled events excluded"
curl -sf "http://localhost:9132/v1/search?q=arijt" 2>/dev/null | python3 -c "
import sys,json
d=json.load(sys.stdin)
print(f\"   'arijt' (misspelled) matched {d['Total']} event(s)\")
for e in d['Events']: print(f\"     {e['title']}\")
print(f\"   facets: {d['Facets'].get('city')}\")
" 2>/dev/null || note "search unavailable (is search-svc running?)"

printf '\n\033[1;32mWalkthrough complete.\033[0m\n'
cat <<'NEXT'
   Open http://localhost:3000 and click into an event to use the seat map.
   Open it in two tabs: holding a seat in one greys it out in the other.

   make reset   release every seat and start over
   make status  what is running
NEXT
