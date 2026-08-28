CREATE TABLE IF NOT EXISTS lecture_query_service.persisted_messages (
      idempotent_key UUID PRIMARY KEY,
      method VARCHAR(255),
      processed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_persisted_messages_timestamp ON lecture_query_service.persisted_messages (processed_at);
