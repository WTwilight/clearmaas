-- Migration: Add enterprise pricing sheet channels binding
-- Compatible with: SQLite, MySQL

CREATE TABLE IF NOT EXISTS enterprise_pricing_sheet_channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pricing_sheet_id INTEGER NOT NULL,
    channel_id INTEGER NOT NULL,
    created_at BIGINT,
    UNIQUE(pricing_sheet_id, channel_id)
);

CREATE INDEX IF NOT EXISTS idx_sheet_channel_sheet
    ON enterprise_pricing_sheet_channels(pricing_sheet_id);

CREATE INDEX IF NOT EXISTS idx_sheet_channel_channel
    ON enterprise_pricing_sheet_channels(channel_id);
