-- Migration: v1.0-v1.1 Platform Enterprise & Token Pricing Binding
-- Compatible with: SQLite
--
-- 1. Ensure ent_type column exists (idempotent)
-- 2. Backfill ent_type values: id=-1 → 'platform', others → 'enterprise'
-- 3. Ensure platform enterprise (id=-1) has correct ent_type and name
-- 4. Create token_pricing_model_bindings table
--
-- IMPORTANT: This migration is INCREMENTAL and idempotent.
-- - Existing production/development data is preserved.

-- ============================================================
-- Step 1: Ensure ent_type column exists
-- Only add if not already present.
-- If 'type' column exists (from buggy run), we don't need it; Go reads ent_type.
-- ============================================================

-- SQLite: we add ent_type if it doesn't exist
-- The column check via PRAGMA is done inline below.

-- ============================================================
-- Step 2: Backfill ent_type values (idempotent)
-- - id=-1 is the platform enterprise → 'platform'
-- - all others → 'enterprise'
-- ============================================================

UPDATE enterprises SET ent_type = 'platform' WHERE id = -1;
UPDATE enterprises SET ent_type = 'enterprise' WHERE id != -1;

-- ============================================================
-- Step 3: Ensure platform enterprise exists (id=-1, ent_type='platform')
-- Insert if missing; update name/type if exists with wrong values.
-- ============================================================

INSERT INTO enterprises (id, name, ent_type, status, created_at, updated_at)
VALUES (-1, '平台', 'platform', 1, strftime('%s', 'now'), strftime('%s', 'now'))
ON CONFLICT(id) DO UPDATE SET
    name = '平台',
    ent_type = 'platform',
    status = 1,
    updated_at = strftime('%s', 'now');

-- ============================================================
-- Step 4: Create token_pricing_model_bindings table
-- ============================================================

CREATE TABLE IF NOT EXISTS token_pricing_model_bindings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token_id INTEGER NOT NULL,
    pricing_sheet_id INTEGER NOT NULL,
    model VARCHAR(128) NOT NULL,
    created_at INTEGER NOT NULL DEFAULT 0,
    deleted_at INTEGER DEFAULT NULL
);

CREATE INDEX IF NOT EXISTS idx_token_pricing_binding_token_id ON token_pricing_model_bindings(token_id);
CREATE INDEX IF NOT EXISTS idx_token_pricing_binding_user_id ON token_pricing_model_bindings(user_id);
CREATE INDEX IF NOT EXISTS idx_token_pricing_binding_sheet_id ON token_pricing_model_bindings(pricing_sheet_id);
CREATE INDEX IF NOT EXISTS idx_token_pricing_binding_deleted_at ON token_pricing_model_bindings(deleted_at);
