-- Migration: Change model to models array + add vendor_type
-- One row = vendor_type + models JSON array + discount settings (batch config)

-- Step 1: Add vendor_type column
ALTER TABLE enterprise_pricing_items ADD COLUMN vendor_type VARCHAR(64) NOT NULL DEFAULT '';

-- Step 2: Add models column as JSON (stores array of model names)
ALTER TABLE enterprise_pricing_items ADD COLUMN models JSON;

-- Step 3: Backfill models array from existing model name
UPDATE enterprise_pricing_items SET models = JSON_ARRAY(model);

-- Step 4: Backfill vendor_type based on model name patterns
UPDATE enterprise_pricing_items SET vendor_type = (
  CASE
    WHEN LOWER(model) LIKE '%gpt%' OR LOWER(model) LIKE '%chatgpt%' OR LOWER(model) LIKE '%dall%' THEN 'openai'
    WHEN LOWER(model) LIKE '%claude%' OR LOWER(model) LIKE '%anthropic%' THEN 'anthropic'
    WHEN LOWER(model) LIKE '%gemini%' OR LOWER(model) LIKE '%learnlm%' THEN 'google'
    WHEN LOWER(model) LIKE '%grok%' OR LOWER(model) LIKE '%xai%' THEN 'xai'
    WHEN LOWER(model) LIKE '%deepseek%' THEN 'deepseek'
    WHEN LOWER(model) LIKE '%qwen%' OR LOWER(model) LIKE '%qwq%' THEN 'qwen'
    WHEN LOWER(model) LIKE '%doubao%' OR LOWER(model) LIKE '%volcengine%' THEN 'doubao'
    WHEN LOWER(model) LIKE '%moonshot%' OR LOWER(model) LIKE '%kimi%' THEN 'moonshot'
    WHEN LOWER(model) LIKE '%mistral%' OR LOWER(model) LIKE '%mixtral%' THEN 'mistral'
    WHEN LOWER(model) LIKE '%llama%' OR LOWER(model) LIKE '%meta%' THEN 'meta'
    WHEN LOWER(model) LIKE '%command%' OR LOWER(model) LIKE '%cohere%' THEN 'cohere'
    ELSE ''
  END
);

-- Step 5: Drop the old model column and its unique index
ALTER TABLE enterprise_pricing_items DROP INDEX idx_sheet_model;
ALTER TABLE enterprise_pricing_items DROP COLUMN model;

-- Step 6: Drop NOT NULL on vendor_type
ALTER TABLE enterprise_pricing_items ALTER COLUMN vendor_type DROP NOT NULL;
