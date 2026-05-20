-- Migration: Change supplier_pricing_items.model -> models JSONB array
-- One row = models JSONB array + discount settings (batch config)
-- Aligned with enterprise_pricing_items structure.
-- Compatible with: PostgreSQL

-- Step 1: Add models column as JSONB (stores array of model names)
ALTER TABLE supplier_pricing_items ADD COLUMN models JSONB;

-- Step 2: Backfill models array from existing model name
UPDATE supplier_pricing_items SET models = to_jsonb(ARRAY[model]::TEXT[]);

-- Step 3: Drop the old model column (requires CASCADE to remove dependent objects)
ALTER TABLE supplier_pricing_items DROP COLUMN IF EXISTS model CASCADE;

-- Step 4: Drop the unique index (recreated without model column)
DROP INDEX IF EXISTS idx_supplier_sheet_model;
