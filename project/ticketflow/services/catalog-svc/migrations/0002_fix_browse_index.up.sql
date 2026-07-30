-- Fixes the browse index, which was never actually used by the browse query.
--
-- 0001 created:
--     (status, starts_at) WHERE status IN (1, 2)
--
-- The intent was for the index to satisfy both the filter and the ORDER BY.
-- It cannot. Because the query matches *both* status values, a leading status
-- column yields status=1 rows ordered by starts_at followed by status=2 rows
-- ordered by starts_at -- correct per group, but not a global ordering. Postgres
-- therefore had to sort anyway, and once it was sorting it saw no reason to use
-- the index at all.
--
-- The leading status column was redundant to begin with: the partial predicate
-- already restricts the index to exactly those rows. Dropping it and ordering
-- by (starts_at, id) makes the index match the ORDER BY exactly, so LIMIT stops
-- after reading a page's worth of entries.
--
-- Including id serves the keyset cursor, which pages on (starts_at, id).
--
-- Measured on 50k events, LIMIT 21:
--   before: Seq Scan + Sort, 16.664 ms, 1109 buffers, 45003 rows scanned
--   after:  Index Scan, no sort node, 0.039 ms, 5 buffers, 21 rows scanned

DROP INDEX IF EXISTS event_browse_idx;

CREATE INDEX event_browse_idx
    ON event (starts_at, id)
    WHERE status IN (1, 2);
