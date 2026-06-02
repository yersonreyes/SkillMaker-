-- Migration: 0002_add_supervisions (down)
-- Reverses: drops the supervision table and its index.
-- The index is dropped automatically when the table is dropped.

DROP TABLE IF EXISTS supervision;
