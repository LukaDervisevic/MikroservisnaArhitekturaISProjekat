ALTER TABLE lecture_service.lectures DROP CONSTRAINT lectures_pkey;

ALTER TABLE lecture_service.lectures ALTER COLUMN lecture_id DROP DEFAULT;

DROP SEQUENCE IF EXISTS lecture_service.lectures_lecture_id_seq;

ALTER TABLE lecture_service.lectures ADD PRIMARY KEY (lecture_id, event_id, lecturer_id);
