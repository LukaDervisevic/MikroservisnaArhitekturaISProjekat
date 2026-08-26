CREATE TABLE IF NOT EXISTS event_service.saga_instances (
    id            UUID PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    aggregate_id  BIGINT       NOT NULL,
    state         VARCHAR(32)  NOT NULL,
    current_step  INT          NOT NULL DEFAULT 0,
    payload       BYTEA,
    last_error    TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_saga_instances_state ON event_service.saga_instances (state);
CREATE INDEX IF NOT EXISTS idx_saga_instances_aggregate ON event_service.saga_instances (aggregate_id);

CREATE TABLE IF NOT EXISTS event_service.saga_step_logs (
    id           UUID PRIMARY KEY,
    saga_id      UUID         NOT NULL REFERENCES event_service.saga_instances(id) ON DELETE CASCADE,
    step_index   INT          NOT NULL,
    name         VARCHAR(255) NOT NULL,
    state        VARCHAR(32)  NOT NULL,
    compensation BYTEA,
    last_error   TEXT,
    started_at   TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_saga_step_logs_saga ON event_service.saga_step_logs (saga_id, step_index);
