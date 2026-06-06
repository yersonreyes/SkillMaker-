-- Migration 0011 DOWN: remove descripcion column from video table.
ALTER TABLE video DROP COLUMN descripcion;
