CREATE TABLE IF NOT EXISTS lecturer_service.outboxes (
    id UUID PRIMARY KEY,
    payload BYTEA NOT NULL,
    status INT NOT NULL DEFAULT 0,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_outboxes_status_timestamp ON lecturer_service.outboxes (status, timestamp);

