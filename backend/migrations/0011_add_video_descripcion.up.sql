-- Migration 0011: add descripcion column to video table.
-- This column stores an optional long-form description for each video (max 5000 chars enforced at service layer).
-- Default '' ensures existing rows are valid without a data fix.
ALTER TABLE video ADD COLUMN descripcion text NOT NULL DEFAULT '';
