-- Migration: v0.9-v1.0 supplier_pricing_sheet_channels (multi-channel binding)
-- Create the intermediate table for many-to-many relationship between supplier pricing sheets and channels.

CREATE TABLE IF NOT EXISTS supplier_pricing_sheet_channels (
    id SERIAL PRIMARY KEY,
    pricing_sheet_id INT NOT NULL REFERENCES supplier_pricing_sheets(id) ON DELETE CASCADE,
    channel_id INT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    created_at BIGINT NOT NULL DEFAULT 0,
    UNIQUE(pricing_sheet_id, channel_id)
);

CREATE INDEX IF NOT EXISTS idx_supplier_pricing_sheet_channels_sheet_id ON supplier_pricing_sheet_channels(pricing_sheet_id);
CREATE INDEX IF NOT EXISTS idx_supplier_pricing_sheet_channels_channel_id ON supplier_pricing_sheet_channels(channel_id);
