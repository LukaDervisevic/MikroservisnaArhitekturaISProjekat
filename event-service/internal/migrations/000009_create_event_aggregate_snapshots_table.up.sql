CREATE TABLE IF NOT EXISTS event_service.event_aggregate_snapshots (
    aggregate_id UUID NOT NULL,
    version      BIGINT NOT NULL,
    state        JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (aggregate_id, version)
);

CREATE INDEX IF NOT EXISTS idx_event_aggregate_snapshots_latest ON event_service.event_aggregate_snapshots (aggregate_id, version DESC);
