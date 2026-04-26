CREATE TABLE IF NOT EXISTS event_service.lectures (
    lecture_id BIGINT NOT NULL,
    event_id BIGINT NOT NULL,
    lecturer_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    duration BIGINT NOT NULL,

    PRIMARY KEY (lecture_id, event_id, lecturer_id),

    CONSTRAINT fk_lectures_event FOREIGN KEY (event_id) REFERENCES event_service.events(id),
    CONSTRAINT fk_lectures_lecturers FOREIGN KEY (lecturer_id) REFERENCES event_service.lecturers(id)

);