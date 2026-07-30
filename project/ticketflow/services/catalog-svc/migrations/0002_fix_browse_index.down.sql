DROP INDEX IF EXISTS event_browse_idx;

-- Restores 0001's (ineffective) definition.
CREATE INDEX event_browse_idx
    ON event (status, starts_at)
    WHERE status IN (1, 2);
