-- Migration: 0005_add_material_columns
-- Description: add mime_type + tamano_bytes to material (C2.3, RF-07).
-- The material table is EMPTY (schema-only since 0003) — NOT NULL adds are safe.
ALTER TABLE material ADD COLUMN mime_type    text   NOT NULL;
ALTER TABLE material ADD COLUMN tamano_bytes bigint NOT NULL;
