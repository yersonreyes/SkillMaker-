-- Migration: 0009_add_enrollment_completado
-- Description: adds the completado flag to enrollment (C2.4). Progress = this boolean ONLY (D7).
-- The EnrollmentCompleter seam (evaluations) flips it to true on a passed attempt.
ALTER TABLE enrollment ADD COLUMN completado boolean NOT NULL DEFAULT false;
