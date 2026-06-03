-- Reverses 0004_fix_video_columns.
-- Restores storage_key, drops url/proveedor + constraint.
-- Table must be empty before running down (no data migration needed).

ALTER TABLE video ADD COLUMN storage_key text NOT NULL;
ALTER TABLE video DROP CONSTRAINT ck_video_proveedor;
ALTER TABLE video DROP COLUMN proveedor;
ALTER TABLE video DROP COLUMN url;
