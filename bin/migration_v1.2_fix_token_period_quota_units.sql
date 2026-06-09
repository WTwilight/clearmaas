-- Migration: v1.2 Fix Token Period Quota Limits Stored as Raw Dollars
-- Fix quota_limit_daily and quota_limit_monthly that were incorrectly stored as raw dollar amounts
-- instead of quota units (quota units = dollars * 500000 where 1 USD = 500000 quota units)
--
-- This migration corrects existing data where admin entered e.g. "10" meaning "$10",
-- but the value was stored as-is (10 quota units ≈ $0.00002) instead of 10 * 500000 = 5,000,000 quota units.
--
-- IMPORTANT: This is a one-time corrective migration.
-- Only affects tokens where quota_limit > 0 and quota_limit < quotaPerUnit (500000).
-- Tokens with values >= quotaPerUnit are assumed to be already correct (entered after bug was known).

DO $$
BEGIN
    -- Correct quota_limit_daily: only fix if the value looks like a raw dollar amount (too small to be quota units)
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tokens' AND column_name = 'quota_limit_daily'
    ) THEN
        UPDATE tokens
        SET quota_limit_daily = quota_limit_daily * 500000
        WHERE quota_limit_daily > 0
          AND quota_limit_daily < 500000;
    END IF;

    -- Correct quota_limit_monthly: same logic
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tokens' AND column_name = 'quota_limit_monthly'
    ) THEN
        UPDATE tokens
        SET quota_limit_monthly = quota_limit_monthly * 500000
        WHERE quota_limit_monthly > 0
          AND quota_limit_monthly < 500000;
    END IF;
END $$;
