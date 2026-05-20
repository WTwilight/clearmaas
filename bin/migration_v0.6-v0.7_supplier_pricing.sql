-- Migration: Create supplier tables for vendor pricing module
-- This enables managing supplier/vendor pricing sheets and pricing items.
-- Compatible with: PostgreSQL

-- Suppliers table
CREATE TABLE IF NOT EXISTS suppliers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 1,
    remark TEXT NOT NULL DEFAULT '',
    created_at BIGINT NOT NULL DEFAULT 0,
    updated_at BIGINT NOT NULL DEFAULT 0
);

-- Supplier pricing sheets table
CREATE TABLE IF NOT EXISTS supplier_pricing_sheets (
    id BIGSERIAL PRIMARY KEY,
    supplier_id BIGINT NOT NULL DEFAULT 0,
    channel_id BIGINT NOT NULL DEFAULT 0,
    name VARCHAR(255) NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 1,
    start_time BIGINT NOT NULL DEFAULT 0,
    end_time BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT 0,
    updated_at BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT fk_supplier_sheet_supplier FOREIGN KEY (supplier_id) REFERENCES suppliers(id) ON DELETE CASCADE
);

-- Supplier pricing items table (unique index on pricing_sheet_id + model)
CREATE TABLE IF NOT EXISTS supplier_pricing_items (
    id BIGSERIAL PRIMARY KEY,
    pricing_sheet_id BIGINT NOT NULL DEFAULT 0,
    model VARCHAR(255) NOT NULL DEFAULT '',
    discount_type VARCHAR(32) NOT NULL DEFAULT '',
    discount_value DOUBLE PRECISION NOT NULL DEFAULT 0,
    remark TEXT NOT NULL DEFAULT '',
    CONSTRAINT fk_supplier_item_sheet FOREIGN KEY (pricing_sheet_id) REFERENCES supplier_pricing_sheets(id) ON DELETE CASCADE
);

-- Create unique index on pricing_sheet_id + model
CREATE UNIQUE INDEX IF NOT EXISTS idx_supplier_sheet_model ON supplier_pricing_items(pricing_sheet_id, model);
