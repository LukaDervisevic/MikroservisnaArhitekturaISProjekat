CREATE TABLE IF NOT EXISTS lecturer_service.lecturers (
    id BIGSERIAL PRIMARY KEY,
    full_name VARCHAR(255) NOT NULL,
    title VARCHAR(100) NOT NULL,
    field_of_expertise VARCHAR(255) NOT NULL
);