-- Migration: Add enterprise pricing sheet channels binding
-- Compatible with: PostgreSQL

CREATE TABLE IF NOT EXISTS enterprise_pricing_sheet_channels (
    id BIGSERIAL PRIMARY KEY,
    pricing_sheet_id BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    created_at BIGINT,
    CONSTRAINT uk_sheet_channel UNIQUE (pricing_sheet_id, channel_id)
);

CREATE INDEX IF NOT EXISTS idx_sheet_channel_sheet
    ON enterprise_pricing_sheet_channels(pricing_sheet_id);

CREATE INDEX IF NOT EXISTS idx_sheet_channel_channel
    ON enterprise_pricing_sheet_channels(channel_id);
