ALTER TABLE lecturer_service.lecturers
    ADD COLUMN IF NOT EXISTS email VARCHAR(255) NOT NULL DEFAULT '';
