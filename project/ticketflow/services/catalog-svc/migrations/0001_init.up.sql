-- catalog-svc schema: the descriptive half of an event.
--
-- Read-heavy and slow-changing, which is exactly what makes it safe to cache
-- aggressively. Contrast with inventory-svc, whose data is never cached.
--
-- Rich, variable-shape content (promoter copy, per-genre metadata) lives in
-- MongoDB instead; only the canonical, queryable fields are here.

CREATE TABLE venue (
    id           TEXT        PRIMARY KEY,
    name         TEXT        NOT NULL,
    city         TEXT        NOT NULL,
    country_code CHAR(2)     NOT NULL,
    address      TEXT        NOT NULL DEFAULT '',
    latitude     DOUBLE PRECISION,
    longitude    DOUBLE PRECISION,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX venue_city_idx ON venue (city);

CREATE TABLE seat_map (
    id             TEXT        PRIMARY KEY,
    venue_id       TEXT        NOT NULL REFERENCES venue (id) ON DELETE RESTRICT,
    viewbox_width  REAL        NOT NULL,
    viewbox_height REAL        NOT NULL,
    version        BIGINT      NOT NULL DEFAULT 1,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX seat_map_venue_idx ON seat_map (venue_id);

-- Fixed geometry only: where a seat physically is. Its availability lives in
-- inventory-svc's database. That split is what makes this table cacheable with
-- a long TTL despite availability changing constantly.
CREATE TABLE seat (
    seat_map_id     TEXT NOT NULL REFERENCES seat_map (id) ON DELETE CASCADE,
    id              TEXT NOT NULL,
    section         TEXT NOT NULL,
    row_label       TEXT NOT NULL,
    seat_number     TEXT NOT NULL,
    pricing_tier_id TEXT NOT NULL,
    x               REAL NOT NULL,
    y               REAL NOT NULL,

    -- Map first: the only access pattern is "give me every seat in this map",
    -- which this PK serves as a single range scan.
    PRIMARY KEY (seat_map_id, id),

    -- A venue cannot have two seats at the same physical position.
    CONSTRAINT seat_position_uniq UNIQUE (seat_map_id, section, row_label, seat_number)
);

CREATE TABLE event (
    id            TEXT        PRIMARY KEY,
    title         TEXT        NOT NULL,
    -- Mirrors ticketflow.catalog.v1.EventKind / EventStatus. Smallint rather
    -- than a Postgres ENUM so adding a value never takes a write-blocking lock.
    kind          SMALLINT    NOT NULL,
    status        SMALLINT    NOT NULL,
    venue_id      TEXT        NOT NULL REFERENCES venue (id) ON DELETE RESTRICT,
    seat_map_id   TEXT        NOT NULL REFERENCES seat_map (id) ON DELETE RESTRICT,
    starts_at     TIMESTAMPTZ NOT NULL,
    ends_at       TIMESTAMPTZ NOT NULL,
    sale_opens_at TIMESTAMPTZ NOT NULL,
    poster_url    TEXT        NOT NULL DEFAULT '',
    tags          TEXT[]      NOT NULL DEFAULT '{}',

    -- Incremented on every mutation. Doubles as the cache-key suffix, so a
    -- stale L1 entry is *detectable* rather than merely expiring on TTL, and
    -- as an optimistic-concurrency token for admin edits.
    version       BIGINT      NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT event_kind_valid   CHECK (kind BETWEEN 1 AND 4),
    CONSTRAINT event_status_valid CHECK (status BETWEEN 1 AND 4),
    CONSTRAINT event_time_order    CHECK (ends_at > starts_at),
    CONSTRAINT event_sale_before_start CHECK (sale_opens_at <= starts_at)
);

-- The storefront's primary browse query is
--   WHERE status = ON_SALE AND starts_at > now() ORDER BY starts_at
-- optionally narrowed by city. This composite is ordered (status, starts_at)
-- so the index supplies the sort as well as the filter -- no sort node in the
-- plan. Phase 7 confirms that with EXPLAIN ANALYZE.
CREATE INDEX event_browse_idx
    ON event (status, starts_at)
    WHERE status IN (1, 2);   -- ANNOUNCED, ON_SALE: cancelled/sold-out are never browsed

CREATE INDEX event_venue_idx ON event (venue_id);

-- Tag filtering ("indie", "premier-league") uses array containment, which needs
-- GIN. Btree cannot answer @> on an array.
CREATE INDEX event_tags_idx ON event USING GIN (tags);

-- Drives the waiting room: which drops open in the next few minutes.
CREATE INDEX event_sale_opens_idx
    ON event (sale_opens_at)
    WHERE status = 1;

CREATE TABLE pricing_tier (
    id            TEXT     PRIMARY KEY,
    event_id      TEXT     NOT NULL REFERENCES event (id) ON DELETE CASCADE,
    name          TEXT     NOT NULL,
    -- Money as minor units, never a float. 1250 + 'INR' means Rs 12.50.
    amount_minor  BIGINT   NOT NULL,
    currency_code CHAR(3)  NOT NULL,

    CONSTRAINT pricing_tier_amount_nonneg CHECK (amount_minor >= 0)
);

CREATE INDEX pricing_tier_event_idx ON pricing_tier (event_id);

COMMENT ON COLUMN event.version IS
'Monotonic mutation counter. Used as the L2 cache key suffix (event:{id}:v{version})
so a schema or content change rolls the keyspace without a manual Redis flush,
and as an optimistic-locking token so two concurrent admin edits cannot silently
overwrite each other.';
