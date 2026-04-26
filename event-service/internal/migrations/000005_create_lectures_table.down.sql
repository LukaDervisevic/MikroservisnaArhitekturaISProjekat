ALTER TABLE event_service.lectures DROP CONSTRAINT fk_lectures_event;
ALTER TABLE event_service.lectures DROP CONSTRAINT fk_lectures_lecturers;
DROP TABLE IF EXISTS event_service.lectures;