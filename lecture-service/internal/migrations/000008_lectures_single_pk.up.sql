ALTER TABLE lecture_service.lectures DROP CONSTRAINT lectures_pkey;

CREATE SEQUENCE IF NOT EXISTS lecture_service.lectures_lecture_id_seq
    OWNED BY lecture_service.lectures.lecture_id;

SELECT setval(
    'lecture_service.lectures_lecture_id_seq',
    COALESCE((SELECT MAX(lecture_id) FROM lecture_service.lectures), 0) + 1,
    false
);

ALTER TABLE lecture_service.lectures
    ALTER COLUMN lecture_id SET DEFAULT nextval('lecture_service.lectures_lecture_id_seq');

ALTER TABLE lecture_service.lectures ADD PRIMARY KEY (lecture_id);
