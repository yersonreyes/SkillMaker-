-- Migration: 0008_add_approvals (C4.1)
-- Adds course.publicado_en (schema-gap fix) + the approval audit table (RF-17b).

ALTER TABLE course ADD COLUMN publicado_en timestamptz;

CREATE TABLE approval (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id   uuid        NOT NULL REFERENCES course(id) ON DELETE CASCADE,
    admin_id    uuid        NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    resultado   text        NOT NULL
        CONSTRAINT ck_approval_resultado CHECK (resultado IN ('aprobado','rechazado')),
    comentario  text        NOT NULL DEFAULT '',
    resuelto_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_approval_course ON approval(course_id);
