-- Reverses 0003. Drops in reverse FK-dependency order (children before parents).
-- Indexes drop automatically with their tables.
DROP TABLE IF EXISTS enrollment;
DROP TABLE IF EXISTS material;
DROP TABLE IF EXISTS video;
DROP TABLE IF EXISTS section;
DROP TABLE IF EXISTS course;
