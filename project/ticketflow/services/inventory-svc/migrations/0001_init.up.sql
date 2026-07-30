-- inventory-svc schema: the authoritative record of who may have which seat.
--
-- Everything here is correctness-critical. Nothing in this database may be
-- cached at any tier -- a stale read here means selling one seat twice.

-- Seat status as a smallint rather than a Postgres ENUM: adding a value to an
-- ENUM takes a lock that blocks writes, which is unacceptable during a drop.
-- Values mirror ticketflow.inventory.v1.SeatStatus.
--   1 = AVAILABLE, 2 = HELD, 3 = SOLD, 4 = BLOCKED

CREATE TABLE seat_hold (
    id              UUID        PRIMARY KEY,
    event_id        TEXT        NOT NULL,
    user_id         TEXT        NOT NULL,
    -- Caller-supplied. The UNIQUE constraint below is what makes a retried
    -- HoldSeats call return the original hold instead of grabbing a second
    -- set of seats -- the database enforces idempotency, not application code.
    idempotency_key TEXT        NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    released_at     TIMESTAMPTZ,

    CONSTRAINT seat_hold_idempotency_uniq UNIQUE (user_id, idempotency_key)
);

-- The reaper sweeps expired holds back to AVAILABLE. A partial index keeps it
-- tiny: only live holds are ever scanned, so the index stays small even after
-- millions of historical holds accumulate.
CREATE INDEX seat_hold_expiry_idx
    ON seat_hold (expires_at)
    WHERE released_at IS NULL;

CREATE TABLE seat_allocation (
    event_id        TEXT        NOT NULL,
    seat_id         TEXT        NOT NULL,
    status          SMALLINT    NOT NULL DEFAULT 1,
    -- Set when status = HELD. FK is intentionally omitted: the hot path updates
    -- this table thousands of times a second and an FK check on every write is
    -- overhead we don't need, since hold_id is only ever written by this service.
    hold_id         UUID,
    hold_expires_at TIMESTAMPTZ,
    -- Set when status = SOLD.
    order_id        TEXT,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Composite PK, event first. Both the point lookup (one seat) and the range
    -- scan (all seats for an event) are served by this one index, so the seat
    -- map query needs no secondary index at all.
    PRIMARY KEY (event_id, seat_id),

    CONSTRAINT seat_allocation_status_valid CHECK (status BETWEEN 1 AND 4),

    -- Structural guarantee that a SOLD seat always names its order and a HELD
    -- seat always names its hold. Prevents a half-written state from a buggy
    -- deploy silently orphaning a seat.
    CONSTRAINT seat_allocation_sold_has_order CHECK (
        status <> 3 OR order_id IS NOT NULL
    ),
    CONSTRAINT seat_allocation_held_has_hold CHECK (
        status <> 2 OR (hold_id IS NOT NULL AND hold_expires_at IS NOT NULL)
    )
);

-- Releasing a hold updates every seat carrying that hold_id. Partial, because
-- only currently-held rows are ever targeted.
CREATE INDEX seat_allocation_hold_idx
    ON seat_allocation (hold_id)
    WHERE hold_id IS NOT NULL;

-- Drives the expiry reaper: find held seats whose hold has lapsed.
CREATE INDEX seat_allocation_expiry_idx
    ON seat_allocation (hold_expires_at)
    WHERE status = 2;

-- Availability counts per event, used by the storefront's "N seats left" badge.
-- Partial index on AVAILABLE only, so it stays small on a sold-out event.
CREATE INDEX seat_allocation_available_idx
    ON seat_allocation (event_id)
    WHERE status = 1;

COMMENT ON TABLE seat_allocation IS
'Source of truth for seat ownership. The no-double-sell guarantee comes from a
single conditional UPDATE rather than SELECT ... FOR UPDATE:

  UPDATE seat_allocation
     SET status = 2, hold_id = $3, hold_expires_at = $4, updated_at = now()
   WHERE event_id = $1
     AND seat_id = ANY($2)
     AND (status = 1 OR (status = 2 AND hold_expires_at < now()))
  RETURNING seat_id;

Postgres evaluates the WHERE clause under a row lock it takes itself, so two
concurrent holds on the same seat cannot both match -- the loser sees the
winner''s committed row and its predicate fails. Returned rows are exactly the
seats won; anything absent from RETURNING was lost to another caller and is
reported as rejected_seat_ids. One statement, no explicit locking, no deadlock
ordering to reason about.

The expired-hold clause lets a lapsed hold be stolen in the same statement,
so the reaper is an optimization for freeing seats promptly rather than a
correctness requirement.';
