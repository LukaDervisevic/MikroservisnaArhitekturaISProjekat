CREATE TABLE IF NOT EXISTS event_service.locations (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    address    VARCHAR(500) NOT NULL,
    capacity   BIGINT       NOT NULL
);