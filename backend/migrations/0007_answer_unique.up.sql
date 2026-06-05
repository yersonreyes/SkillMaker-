-- Migration: 0007_answer_unique (C3.2)
-- Enforces one answer per (attempt, question) so SaveAnswer can upsert via ON CONFLICT.
-- The answer table is schema-only/empty in C3.1 → adding the constraint is safe.
ALTER TABLE answer
    ADD CONSTRAINT uq_answer_attempt_question UNIQUE (attempt_id, question_id);
