CREATE TABLE IF NOT EXISTS lecture_service.events (
    id               BIGSERIAL PRIMARY KEY,
    name             VARCHAR(255)   NOT NULL,
    cotisation_price NUMERIC(10, 2) NOT NULL,
    agenda           TEXT,
    type             VARCHAR(100)   NOT NULL,
    date_time        BIGINT         NOT NULL,
    location_id      BIGINT         NOT NULL,

    CONSTRAINT fk_events_location FOREIGN KEY (location_id) REFERENCES lecture_service.locations(id)
);