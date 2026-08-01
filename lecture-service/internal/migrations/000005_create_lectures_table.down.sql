ALTER TABLE lecture_service.lectures DROP CONSTRAINT fk_lectures_event;
ALTER TABLE lecture_service.lectures DROP CONSTRAINT fk_lectures_lecturers;
DROP TABLE IF EXISTS lecture_service.lectures;