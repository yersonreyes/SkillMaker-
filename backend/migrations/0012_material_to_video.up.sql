-- Migration 0012: relocate material from course to video.
-- BACKFILL strategy: add video_id nullable, backfill to first video by orden/created_at,
-- delete orphans (courses with no video), then set NOT NULL.
-- ISOLATION: this is intentionally a separate file from 0013 because the backfill+orphan-delete
-- is the only risky/irreversible step; isolating it prevents a failed backfill from blocking
-- the unrelated metadata changes in 0013.

-- Step 1: add video_id as nullable FK (so existing rows don't violate NOT NULL yet).
ALTER TABLE material ADD COLUMN video_id uuid REFERENCES video(id) ON DELETE CASCADE;

-- Step 2: backfill — point each material to the first video (by orden ASC, created_at ASC) of its course.
-- Materials on courses that have at least one video get a valid video_id.
-- Materials on courses with NO video remain NULL and are deleted in step 3.
UPDATE material m SET video_id = (
    SELECT v.id FROM video v
    JOIN section s ON s.id = v.section_id
    WHERE s.course_id = m.course_id
    ORDER BY v.orden ASC, v.created_at ASC
    LIMIT 1
);

-- Step 3: delete orphans — materials whose course had no video (video_id is still NULL).
DELETE FROM material WHERE video_id IS NULL;

-- Step 4: set NOT NULL — all remaining rows have a valid video_id.
ALTER TABLE material ALTER COLUMN video_id SET NOT NULL;

-- Step 5: drop the old course_id index explicitly before dropping the column (intent clarity).
DROP INDEX IF EXISTS idx_material_course;

-- Step 6: drop the course_id column (was nullable after step 5 drop, now remove entirely).
ALTER TABLE material DROP COLUMN course_id;

-- Step 7: add new index on video_id for efficient per-video material lookups.
CREATE INDEX idx_material_video ON material(video_id);
