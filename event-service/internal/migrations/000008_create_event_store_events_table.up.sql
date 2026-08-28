CREATE TABLE IF NOT EXISTS event_service.event_store_events (
    event_id       UUID PRIMARY KEY,
    aggregate_id   UUID NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    version        BIGINT NOT NULL,
    event_type     VARCHAR(100) NOT NULL,
    payload        JSONB NOT NULL,
    occurred_at    TIMESTAMPTZ NOT NULL,

    CONSTRAINT uq_event_store_events_aggregate_version UNIQUE (aggregate_id, version)
);

CREATE INDEX IF NOT EXISTS idx_event_store_events_aggregate_id ON event_service.event_store_events (aggregate_id, version);
