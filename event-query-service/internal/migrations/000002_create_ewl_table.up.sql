CREATE TABLE IF NOT EXISTS event_query_service.events_with_locations (
    event_id               BIGSERIAL PRIMARY KEY,
    event_name             VARCHAR(255)   NOT NULL,
    event_cotisation_price NUMERIC(10, 2) NOT NULL,
    event_agenda           TEXT,
    event_type             VARCHAR(100)   NOT NULL,
    event_date_time        BIGINT         NOT NULL,
    location_id            BIGINT         NOT NULL,
    location_name          VARCHAR(255)   NOT NULL,
    location_address       VARCHAR(500)   NOT NULL,
    location_capacity      BIGINT         NOT NULL
);