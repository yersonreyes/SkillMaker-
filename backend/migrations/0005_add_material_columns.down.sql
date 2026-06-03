-- Reverses 0005. Table must be empty (no data migration needed).
ALTER TABLE material DROP COLUMN tamano_bytes;
ALTER TABLE material DROP COLUMN mime_type;
