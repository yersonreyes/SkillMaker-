-- Migration 0013 DOWN: remove course metadata + categoria tables.
-- CHECK constraint drops automatically with the nivel column.

DROP TABLE IF EXISTS course_categoria;
DROP TABLE IF EXISTS categoria;

ALTER TABLE course DROP COLUMN IF EXISTS horas_practico;
ALTER TABLE course DROP COLUMN IF EXISTS miniatura_key;
ALTER TABLE course DROP COLUMN IF EXISTS nivel;
