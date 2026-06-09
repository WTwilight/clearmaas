-- Migration: v1.0-v1.1 Platform Enterprise & Token Pricing Binding
-- 1. Add "ent_type" column to enterprises table and backfill existing records
-- 2. Ensure platform enterprise exists (id=-1, ent_type='platform') for platform-level default pricing sheet
-- 3. Create token_pricing_model_bindings table for per-token per-model pricing bindings
-- Compatible with: PostgreSQL
--
-- IMPORTANT: This migration is INCREMENTAL and idempotent.
-- - Existing production/development data is preserved.
-- - The "ent_type" column is added only if it does not already exist.
-- - Historical enterprise records are backfilled: id=-1 gets ent_type='platform', others get ent_type='enterprise'.
-- - The platform enterprise (id=-1) is inserted/updated with ent_type='platform'.
--
-- NOTE: The column name is "ent_type" (not "type") because the Go model struct uses
-- gorm tag 'column:ent_type'. All reads/writes must use "ent_type" to match the application.

-- ============================================================
-- Step 1: Add "ent_type" column to enterprises table (incremental)
-- ============================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'enterprises' AND column_name = 'ent_type'
    ) THEN
        ALTER TABLE enterprises ADD COLUMN ent_type TEXT NOT NULL DEFAULT '';
    END IF;
END $$;

-- Backfill ent_type values for existing records (idempotent)
UPDATE enterprises SET ent_type = 'platform' WHERE id = -1 AND (ent_type = '' OR ent_type IS NULL);
UPDATE enterprises SET ent_type = 'enterprise' WHERE id != -1 AND (ent_type = '' OR ent_type IS NULL);

-- Ensure platform enterprise always has the correct type
UPDATE enterprises SET ent_type = 'platform' WHERE id = -1;

-- ============================================================
-- Step 2: Insert/update platform enterprise (id=-1, ent_type='platform')
-- We explicitly set id=-1 to maintain backward compatibility.
-- ============================================================

INSERT INTO enterprises (id, name, ent_type, status, created_at, updated_at)
VALUES (-1, '平台', 'platform', 1, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    ent_type = EXCLUDED.ent_type,
    status = EXCLUDED.status,
    updated_at = EXCLUDED.updated_at;

-- ============================================================
-- Step 3: Create token_pricing_model_bindings table
-- token_id uses ON DELETE SET NULL because tokens are soft-deleted
-- (GORM DeletedAt), not physically deleted. The application-level
-- DeleteTokenById/BatchDeleteTokens handles binding cleanup in the
-- same transaction as the token soft-delete.
-- ============================================================

CREATE TABLE IF NOT EXISTS token_pricing_model_bindings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    token_id BIGINT NOT NULL,
    pricing_sheet_id BIGINT NOT NULL,
    model VARCHAR(128) NOT NULL,
    created_at BIGINT NOT NULL DEFAULT 0,
    deleted_at BIGINT DEFAULT NULL,
    CONSTRAINT fk_token_pricing_binding_token
        FOREIGN KEY (token_id) REFERENCES tokens(id) ON DELETE SET NULL,
    CONSTRAINT fk_token_pricing_binding_sheet
        FOREIGN KEY (pricing_sheet_id) REFERENCES enterprise_pricing_sheets(id) ON DELETE RESTRICT,
    CONSTRAINT uq_token_model UNIQUE (token_id, model)
);

CREATE INDEX IF NOT EXISTS idx_token_pricing_binding_token_id ON token_pricing_model_bindings(token_id);
CREATE INDEX IF NOT EXISTS idx_token_pricing_binding_user_id ON token_pricing_model_bindings(user_id);
CREATE INDEX IF NOT EXISTS idx_token_pricing_binding_sheet_id ON token_pricing_model_bindings(pricing_sheet_id);
CREATE INDEX IF NOT EXISTS idx_token_pricing_binding_deleted_at ON token_pricing_model_bindings(deleted_at);
