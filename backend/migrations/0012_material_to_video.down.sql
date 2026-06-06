-- Migration 0012 DOWN: best-effort reverse — NOT lossless.
-- WARNING: this down migration is intended for development rollback ONLY.
-- Data loss is expected: the per-video granularity is collapsed back to course level,
-- and any materials deleted as orphans in the up migration are NOT restored.
-- Do NOT run this in production.

-- Step 1: add course_id as nullable (reverse of step 6 in up).
ALTER TABLE material ADD COLUMN course_id uuid REFERENCES course(id) ON DELETE CASCADE;

-- Step 2: backfill course_id from the video → section → course chain.
UPDATE material m SET course_id = (
    SELECT s.course_id FROM video v
    JOIN section s ON s.id = v.section_id
    WHERE v.id = m.video_id
);

-- Step 3: drop idx_material_video (reverse of step 7).
DROP INDEX IF EXISTS idx_material_video;

-- Step 4: drop video_id column (reverse of step 1).
ALTER TABLE material DROP COLUMN video_id;

-- Step 5: set course_id NOT NULL (reverse of step 4 in up).
ALTER TABLE material ALTER COLUMN course_id SET NOT NULL;

-- Step 6: recreate the original course index.
CREATE INDEX idx_material_course ON material(course_id);
