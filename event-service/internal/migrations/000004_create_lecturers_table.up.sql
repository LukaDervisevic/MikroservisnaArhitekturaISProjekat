CREATE TABLE IF NOT EXISTS event_service.lecturers (
    id                  BIGSERIAL PRIMARY KEY,
    full_name           VARCHAR(255) NOT NULL,
    title               VARCHAR(100),
    field_of_expertise  VARCHAR(255)
);