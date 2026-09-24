-- Migration 0002: per-exam retake limits.
-- max_attempts caps how many times a student may submit the exam (default 1);
-- retake_wait_minutes is the cooldown after each submission before starting again (default 0).
-- The Go server also runs GORM AutoMigrate at startup, which applies these columns automatically.

ALTER TABLE exams
    ADD COLUMN max_attempts INT NOT NULL DEFAULT 1 AFTER duration_minutes,
    ADD COLUMN retake_wait_minutes INT NOT NULL DEFAULT 0 AFTER max_attempts;
