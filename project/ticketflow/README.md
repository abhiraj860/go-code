# TicketFlow

A live-event ticketing and resale marketplace, built as a distributed system.

The whole design turns on one hard problem:

> 50,000 users hit "buy" the instant tickets drop for a single venue, and the system must never
> sell the same seat twice — while staying online through a deploy.

That constraint is what forces ACID inventory, distributed locks, cache-stampede protection,
idempotency at the edge, asynchronous fan-out and blue/green deploys. Nothing here is included
because it looked good on a list.

## Quick start

```bash
make tools     # install pinned buf, protoc plugins, golangci-lint
make up        # start Postgres, Redis, Kafka, ElasticSearch, Mongo, LocalStack
make proto     # generate Go code from .proto
make test      # run the test suite
make help      # every available target
```

`make up` waits for health checks and creates the Kafka topics, so a green `make up` means the
stack is genuinely ready — not just that containers were created.

## Architecture

```
                        Browser (Next.js SSR/SSG/ISR)
                               |
                  +------------+-------------+
                  |                          |
          API Gateway (AWS)          realtime-gateway (Node/TS, WebSocket)
                  |                          ^
        Lambda edge layer                    | live seat + queue updates
   (authorizer, idempotency,                 |
    waiting-room token)                      |
                  |                          |
                  v                          |
          gateway-bff (Go, Gin, REST) -------+
              | L1 in-process LRU
              | L2 Redis cache-aside
       gRPC (protobuf) fan-out via errgroup
    +-------------+-------------+-------------+
    |             |             |             |
catalog-svc  inventory-svc  order-svc    search-svc
 (Mongo +     (Postgres +    (Postgres     (ElasticSearch)
  Postgres)     Redis locks)  + outbox)         ^
    ^                             |             |
    | cache invalidation          v             |
    +---------------------- Kafka topics -------+
                                  |
                    +-------------+--------------+
                    |                            |
              ticket-svc                  notification-svc
        (worker pool, QR/PDF -> S3)      (Echo admin API, SQS -> SNS)
```

### Why five datastores

Polyglot persistence is usually a smell. Here each store is doing something the others are bad at:

| Store | Role |
|---|---|
| **PostgreSQL** | Seats, orders, money. Needs real transactions and the outbox pattern. |
| **Redis** | Cache (L2), sessions, seat holds with TTL, rate limiting, pub/sub to the WS gateway. |
| **MongoDB** | Event content documents — a concert, a football match and a theatre run have genuinely different metadata shapes. |
| **ElasticSearch** | Faceted search with custom analyzers and typo tolerance. |
| **DynamoDB** | Idempotency keys and purchase limits, written from Lambda where a Postgres connection pool isn't available. |

### Caching, and what is deliberately *not* cached

| Data | Volatility | Decision |
|---|---|---|
| Seat-map layout | Changes ~never | Cache aggressively, long TTL |
| Event metadata, pricing tiers | Changes rarely | Cache-aside, invalidated over Kafka |
| Search results & facets | Changes on reindex | Short TTL, best-effort |
| **Seat availability** | Changes 1000×/sec during a drop | **Never cached** — always reads through to the source of truth |

Caching the seat *map* but never the seat *state* is the central judgment call. The system is
fast because the static 95% is cached; it is correct because the volatile 5% never is.

Supporting mechanics: a generic in-process `cache.LRU[K,V]` in front of Redis, `singleflight`
to collapse concurrent misses, jittered early expiration so keys don't expire in lockstep, and
Kafka-driven invalidation so in-process caches on N replicas can't drift independently.

## Layout

```
proto/      protobuf contracts (own Go module) + committed generated code
pkg/        shared Go libraries (own Go module): cache, kafka, logging, errors, config
services/   seven Go services, one binary each
node/       realtime-gateway (WebSocket) and web (Next.js storefront)
lambdas/    edge auth, idempotency, waiting room, image resize, gate scan
deploy/     docker-compose, Kubernetes manifests, Terraform
loadtest/   k6 scenarios
```

Three Go modules tied together by `go.work`. `proto/` is separate so consumers can import the
contracts without pulling in service code.

## Conventions

- **Money is `int64` minor units**, never a float.
- **Every enum has an `_UNSPECIFIED = 0` member**, so "unset" is distinguishable from a real value.
- **Generated protobuf code is committed**; CI regenerates and fails on drift.
- **`buf breaking` gates every PR** — a wire-incompatible change to a released `v1` package must
  ship as a new version instead.
- **Hand-written SQL, no ORM.** Query plans are part of the point.

## Status

| Phase | Scope | State |
|---|---|---|
| 0 | Foundations: workspace, compose stack, buf codegen, CI | ✅ done |
| 1 | catalog-svc, inventory-svc, gateway-bff, L1+L2 cache | ✅ done |
| 2 | Transactional outbox, Kafka, ticket-svc, search-svc | ✅ done |
| 3 | Next.js storefront, realtime WebSocket gateway | |
| 4 | AWS on LocalStack: Lambda, API GW, DynamoDB, SQS/SNS | |
| 5 | Docker, Kubernetes, blue/green | |
| 6 | Real AWS: ECS Fargate, CodePipeline, CloudWatch | |
| 7 | Load testing, cache tuning, query optimization | |
