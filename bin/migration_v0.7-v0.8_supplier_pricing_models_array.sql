-- Migration: Change supplier_pricing_items.model -> models JSON array
-- One row = models JSON array + discount settings (batch config)
-- Aligned with enterprise_pricing_items structure.

-- Step 1: Add models column as JSON (stores array of model names)
ALTER TABLE supplier_pricing_items ADD COLUMN models JSON;

-- Step 2: Backfill models array from existing model name
UPDATE supplier_pricing_items SET models = JSON_ARRAY(model);

-- Step 3: Drop the old model column and its unique index
ALTER TABLE supplier_pricing_items DROP INDEX IF EXISTS idx_supplier_sheet_model;
ALTER TABLE supplier_pricing_items DROP COLUMN IF EXISTS model;
