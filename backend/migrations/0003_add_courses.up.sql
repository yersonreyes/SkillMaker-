-- Migration: 0003_add_courses
-- Description: courses aggregate. C2.1 exercises only `course` (full CRUD);
--   section/video/material/enrollment are schema-only (created now so C2.2/C2.3/C2.4
--   add ONLY endpoints, never DDL — proposal D3). estado is a CHECK-constrained text
--   column mirroring role.nombre (proposal D2). creador_id uses ON DELETE RESTRICT to
--   protect course/approval history from user hard-deletes (proposal D1).

CREATE TABLE course (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    creador_id  uuid        NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    titulo      text        NOT NULL,
    descripcion text        NOT NULL DEFAULT '',
    estado      text        NOT NULL DEFAULT 'borrador'
        CONSTRAINT ck_course_estado CHECK (estado IN ('borrador','en_revision','aprobado','rechazado')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_course_creador ON course(creador_id);
CREATE INDEX idx_course_estado  ON course(estado);

CREATE TABLE section (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id  uuid        NOT NULL REFERENCES course(id) ON DELETE CASCADE,
    titulo     text        NOT NULL,
    orden      integer     NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_section_course ON section(course_id);

CREATE TABLE video (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    section_id  uuid        NOT NULL REFERENCES section(id) ON DELETE CASCADE,
    titulo      text        NOT NULL,
    storage_key text        NOT NULL,
    duracion_s  integer     NOT NULL DEFAULT 0,
    orden       integer     NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_video_section ON video(section_id);

CREATE TABLE material (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id   uuid        NOT NULL REFERENCES course(id) ON DELETE CASCADE,
    titulo      text        NOT NULL,
    storage_key text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_material_course ON material(course_id);

CREATE TABLE enrollment (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid        NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    course_id   uuid        NOT NULL REFERENCES course(id) ON DELETE CASCADE,
    inscrito_en timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_enrollment_user_course UNIQUE (user_id, course_id)
);
CREATE INDEX idx_enrollment_user   ON enrollment(user_id);
CREATE INDEX idx_enrollment_course ON enrollment(course_id);
