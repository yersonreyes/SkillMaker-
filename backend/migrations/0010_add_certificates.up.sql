CREATE TABLE certificate (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    course_id   uuid NOT NULL REFERENCES course(id) ON DELETE CASCADE,
    codigo      text NOT NULL,
    storage_key text NOT NULL,
    emitido_en  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_certificate_user_course UNIQUE (user_id, course_id),
    CONSTRAINT uq_certificate_codigo UNIQUE (codigo)
);
CREATE INDEX idx_certificate_user ON certificate(user_id);

CREATE TABLE badge (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre      text NOT NULL UNIQUE,
    descripcion text NOT NULL DEFAULT '',
    umbral      integer NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE user_badge (
    user_id     uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    badge_id    uuid NOT NULL REFERENCES badge(id) ON DELETE CASCADE,
    otorgado_en timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, badge_id)
);

INSERT INTO badge (nombre, descripcion, umbral) VALUES
 ('Primer curso completado', 'Completaste tu primer curso', 1),
 ('5 cursos completados', 'Completaste 5 cursos', 5),
 ('10 cursos completados', 'Completaste 10 cursos', 10)
ON CONFLICT (nombre) DO NOTHING;
