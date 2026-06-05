-- Migration: 0008_add_approvals DOWN (C4.1)
-- Removes approval table and publicado_en column from course.

DROP TABLE IF EXISTS approval;
ALTER TABLE course DROP COLUMN IF EXISTS publicado_en;
