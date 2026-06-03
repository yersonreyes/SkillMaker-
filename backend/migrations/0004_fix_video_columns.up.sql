-- Migration: 0004_fix_video_columns
-- Description: replace storage_key (belongs to material/C2.3, RT-24) with
--   url + proveedor for embedded external videos (youtube, vimeo). KEEP duracion_s.
-- The video table is EMPTY (schema-only since 0003) — NOT NULL adds are safe.

ALTER TABLE video ADD COLUMN url       text NOT NULL;
ALTER TABLE video ADD COLUMN proveedor text NOT NULL;
ALTER TABLE video ADD CONSTRAINT ck_video_proveedor CHECK (proveedor IN ('youtube','vimeo'));
ALTER TABLE video DROP COLUMN storage_key;
