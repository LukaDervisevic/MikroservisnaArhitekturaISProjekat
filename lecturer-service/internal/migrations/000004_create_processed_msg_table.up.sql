CREATE TABLE IF NOT EXISTS lecturer_service.processed_messages (
    idempotent_key UUID PRIMARY KEY,
    method VARCHAR(255),
    processed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

CREATE INDEX IF NOT EXISTS idx_processed_messages_timestamp ON lecturer_service.processed_messages (processed_at);