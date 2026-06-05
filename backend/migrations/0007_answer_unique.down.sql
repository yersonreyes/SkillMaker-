-- Migration: 0007_answer_unique (C3.2) — rollback
-- Drops the UNIQUE constraint added in the up migration.
ALTER TABLE answer
    DROP CONSTRAINT IF EXISTS uq_answer_attempt_question;
