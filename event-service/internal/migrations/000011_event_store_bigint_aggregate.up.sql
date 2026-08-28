TRUNCATE TABLE event_service.event_store_events;
TRUNCATE TABLE event_service.event_aggregate_snapshots;

ALTER TABLE event_service.event_store_events
    ALTER COLUMN aggregate_id TYPE BIGINT USING NULL;

ALTER TABLE event_service.event_aggregate_snapshots
    ALTER COLUMN aggregate_id TYPE BIGINT USING NULL;
