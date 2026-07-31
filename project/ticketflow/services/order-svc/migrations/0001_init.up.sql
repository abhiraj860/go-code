-- order-svc schema: orders, payments, and the transactional outbox.
--
-- The outbox exists to solve one problem: an order and the Kafka message
-- announcing it must either both happen or neither. Publishing to Kafka inside
-- the transaction is impossible (Kafka is not transactional with Postgres), and
-- publishing after the commit means a crash in between loses the message
-- forever -- an order that exists but that no ticket is ever issued for.
--
-- Writing the message to a table in the SAME transaction makes it atomic with
-- the order. A separate relay then reads unpublished rows and sends them. That
-- is at-least-once delivery, so every consumer must be idempotent.

CREATE TABLE customer_order (
    id              TEXT        PRIMARY KEY,
    user_id         TEXT        NOT NULL,
    event_id        TEXT        NOT NULL,
    -- The inventory hold being converted into a sale. Not a foreign key:
    -- inventory owns its own database and services do not reach across.
    hold_id         UUID        NOT NULL,
    -- 1 = PENDING, 2 = PAID, 3 = FAILED, 4 = CANCELLED. Smallint rather than a
    -- Postgres ENUM so adding a state never takes a write-blocking lock.
    status          SMALLINT    NOT NULL DEFAULT 1,
    seat_ids        TEXT[]      NOT NULL,
    -- Money in minor units. Never a float: order totals get summed.
    total_minor     BIGINT      NOT NULL,
    currency_code   CHAR(3)     NOT NULL,
    -- Same contract as inventory: the database enforces idempotency, because a
    -- check-then-act in application code has a race window.
    idempotency_key TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT customer_order_idempotency_uniq UNIQUE (user_id, idempotency_key),
    CONSTRAINT customer_order_status_valid CHECK (status BETWEEN 1 AND 4),
    CONSTRAINT customer_order_total_nonneg CHECK (total_minor >= 0),
    CONSTRAINT customer_order_has_seats CHECK (cardinality(seat_ids) > 0)
);

CREATE INDEX customer_order_user_idx ON customer_order (user_id, created_at DESC);
CREATE INDEX customer_order_event_idx ON customer_order (event_id);

-- A held hold may only ever back one order.
CREATE UNIQUE INDEX customer_order_hold_uniq ON customer_order (hold_id);

CREATE TABLE outbox (
    -- BIGSERIAL gives a total order for replay and debugging. It is NOT used as
    -- a relay cursor -- see the note at the bottom of this file, which is the
    -- subtlest thing about the whole pattern.
    id             BIGSERIAL   PRIMARY KEY,

    -- Aggregate this event is about, so a consumer can partition by it and
    -- preserve per-order ordering.
    aggregate_type TEXT        NOT NULL,
    aggregate_id   TEXT        NOT NULL,

    -- Kafka topic, e.g. 'order.created'.
    topic          TEXT        NOT NULL,
    -- Message key. Same key means same partition means preserved ordering.
    message_key    TEXT        NOT NULL,

    payload        JSONB       NOT NULL,
    -- Trace id and schema version travel here so a consumer can correlate a
    -- message with the request that produced it.
    headers        JSONB       NOT NULL DEFAULT '{}'::jsonb,

    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at   TIMESTAMPTZ,

    -- Retry bookkeeping. A row stuck with a high attempt count and a
    -- last_error is the signal that something needs a human.
    attempts       INT         NOT NULL DEFAULT 0,
    last_error     TEXT,

    CONSTRAINT outbox_topic_not_empty CHECK (topic <> '')
);

-- The relay's only query. Partial, so it indexes just the backlog: on a healthy
-- system that is a handful of rows even after millions have been published,
-- and the index does not grow without bound.
CREATE INDEX outbox_unpublished_idx
    ON outbox (id)
    WHERE published_at IS NULL;

-- Supports pruning published rows on a retention schedule.
CREATE INDEX outbox_published_idx
    ON outbox (published_at)
    WHERE published_at IS NOT NULL;

COMMENT ON TABLE outbox IS
'Transactional outbox. Rows are written in the same transaction as the state
change they describe, so a message can never exist without its order, nor an
order without its message.

THE SUBTLE PART -- why the relay tracks published_at rather than a cursor.

The obvious relay design is "remember the last id I sent, then poll for
id > last_seen". That design silently loses messages. BIGSERIAL hands out ids
at INSERT time, but rows become visible at COMMIT time, and those orders can
differ: transaction A can take id 100, transaction B take id 101 and commit
first. A relay that reads 101 and advances its cursor will never see 100 when
it commits a moment later. The message is lost with no error anywhere.

Tracking published_at instead removes the problem entirely -- a row is either
marked sent or it is not, and late-committing rows are simply picked up on the
next sweep regardless of id. The relay claims work with

    SELECT ... WHERE published_at IS NULL
     ORDER BY id LIMIT $1
       FOR UPDATE SKIP LOCKED

so several relay replicas can run concurrently without processing the same row
twice, and a slow publish blocks nobody else.

Delivery is at-least-once: a crash after the Kafka publish but before the
UPDATE will resend. Every consumer must therefore be idempotent, which is why
messages carry a stable event id.';
