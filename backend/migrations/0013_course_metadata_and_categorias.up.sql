-- Migration 0013: add course metadata fields + curated categoria system.
-- Adds nivel (CHECK), miniatura_key (nullable), horas_practico (stored, not computed).
-- Creates categoria lookup table + course_categoria join table (composite PK mirrors user_role from 0001).
-- Seeds 8 curated categorias with ON CONFLICT DO NOTHING for idempotency.

-- Course metadata columns.
ALTER TABLE course ADD COLUMN nivel text
    CONSTRAINT ck_course_nivel CHECK (nivel IS NULL OR nivel IN ('basico', 'intermedio', 'avanzado'));

ALTER TABLE course ADD COLUMN miniatura_key text;

ALTER TABLE course ADD COLUMN horas_practico numeric NOT NULL DEFAULT 0;

-- Curated categoria lookup table.
CREATE TABLE categoria (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre     text        NOT NULL UNIQUE,
    slug       text        NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Join table: course ↔ categoria (composite PK mirrors user_role pattern from 0001:28).
CREATE TABLE course_categoria (
    course_id    uuid NOT NULL REFERENCES course(id)    ON DELETE CASCADE,
    categoria_id uuid NOT NULL REFERENCES categoria(id) ON DELETE CASCADE,
    PRIMARY KEY (course_id, categoria_id)
);

CREATE INDEX idx_course_categoria_categoria ON course_categoria(categoria_id);

-- Seed 8 curated categorias (idempotent via ON CONFLICT DO NOTHING).
INSERT INTO categoria (nombre, slug) VALUES
    ('Arquitectura de software', 'arquitectura-de-software'),
    ('SQL',                       'sql'),
    ('Frontend',                  'frontend'),
    ('Backend',                   'backend'),
    ('DevOps',                    'devops'),
    ('Testing',                   'testing'),
    ('Seguridad',                 'seguridad'),
    ('Datos e IA',                'datos-e-ia')
ON CONFLICT (nombre) DO NOTHING;
