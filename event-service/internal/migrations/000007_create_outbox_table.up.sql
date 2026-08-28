CREATE TABLE IF NOT EXISTS event_service.outboxes (
    id UUID PRIMARY KEY,
    payload BYTEA NOT NULL,
    status INT NOT NULL DEFAULT 0,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_event_outboxes_status_timestamp ON event_service.outboxes (status, timestamp);
