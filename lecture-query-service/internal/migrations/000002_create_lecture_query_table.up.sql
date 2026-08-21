CREATE TABLE IF NOT EXISTS lecture_query_service.lecture_query (
     event_id                                   BIGSERIAL,
     event_name                   VARCHAR(255)   NOT NULL,
     event_cotisation_price       NUMERIC(10, 2) NOT NULL,
     event_agenda                                    TEXT,
     event_type                   VARCHAR(100)   NOT NULL,
     event_date_time              BIGINT         NOT NULL,
     location_id                  BIGINT         NOT NULL,
     location_name                VARCHAR(255)   NOT NULL,
     location_address             VARCHAR(500)   NOT NULL,
     location_capacity            BIGINT         NOT NULL,
     lecturer_id                                BIGSERIAL,
     lecturer_full_name             VARCHAR(255) NOT NULL,
     lecturer_title                          VARCHAR(100),
     lecturer_field_of_expertise             VARCHAR(255),
     lecture_id                        BIGSERIAL NOT NULL,
     lecture_name                   VARCHAR(255) NOT NULL,
     lecture_duration                     BIGINT NOT NULL,

     PRIMARY KEY (lecture_id,event_id,lecturer_id)
);