-- Migration: v1.1-v1.2 Token Period Quota Limits
-- Add daily/monthly quota limit fields to tokens table for multi-period token限额功能
-- Compatible with: PostgreSQL, MySQL, SQLite
--
-- IMPORTANT: This migration is INCREMENTAL and idempotent.
-- - Existing production/development data is preserved.
-- - Each column is added only if it does not already exist.

-- ============================================================
-- Add quota_limit_daily column (每日配额限额)
-- ============================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tokens' AND column_name = 'quota_limit_daily'
    ) THEN
        ALTER TABLE tokens ADD COLUMN quota_limit_daily INT NOT NULL DEFAULT 0;
    END IF;
END $$;

-- ============================================================
-- Add quota_limit_monthly column (每月配额限额)
-- ============================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tokens' AND column_name = 'quota_limit_monthly'
    ) THEN
        ALTER TABLE tokens ADD COLUMN quota_limit_monthly INT NOT NULL DEFAULT 0;
    END IF;
END $$;

-- ============================================================
-- Add quota_used_daily column (当前日已用配额，系统维护)
-- ============================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tokens' AND column_name = 'quota_used_daily'
    ) THEN
        ALTER TABLE tokens ADD COLUMN quota_used_daily INT NOT NULL DEFAULT 0;
    END IF;
END $$;

-- ============================================================
-- Add quota_used_monthly column (当前月已用配额，系统维护)
-- ============================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tokens' AND column_name = 'quota_used_monthly'
    ) THEN
        ALTER TABLE tokens ADD COLUMN quota_used_monthly INT NOT NULL DEFAULT 0;
    END IF;
END $$;

-- ============================================================
-- Add quota_daily_reset_last column (日配额上次重置时间，Unix seconds UTC)
-- ============================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tokens' AND column_name = 'quota_daily_reset_last'
    ) THEN
        ALTER TABLE tokens ADD COLUMN quota_daily_reset_last BIGINT NOT NULL DEFAULT 0;
    END IF;
END $$;

-- ============================================================
-- Add quota_monthly_reset_last column (月配额上次重置时间，Unix seconds UTC)
-- ============================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tokens' AND column_name = 'quota_monthly_reset_last'
    ) THEN
        ALTER TABLE tokens ADD COLUMN quota_monthly_reset_last BIGINT NOT NULL DEFAULT 0;
    END IF;
END $$;
