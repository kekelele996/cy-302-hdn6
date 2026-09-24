-- Migration 0002: per-exam attempt limits.
-- max_attempts: 最多作答次数（默认 1 次）
-- wait_minutes: 每次交卷后的等待分钟数（默认 0，即不等待）
-- GORM AutoMigrate also adds these columns automatically; this script is for
-- manual upgrades of databases created with migration 0001.

ALTER TABLE exams
    ADD COLUMN max_attempts INT NOT NULL DEFAULT 1 AFTER duration_minutes,
    ADD COLUMN wait_minutes INT NOT NULL DEFAULT 0 AFTER max_attempts;
