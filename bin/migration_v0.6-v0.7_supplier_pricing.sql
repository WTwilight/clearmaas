-- Migration: Create supplier tables for vendor pricing module
-- This enables managing supplier/vendor pricing sheets and pricing items.

-- Suppliers table
CREATE TABLE IF NOT EXISTS suppliers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 1,
    remark TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL DEFAULT 0
);

-- Supplier pricing sheets table
CREATE TABLE IF NOT EXISTS supplier_pricing_sheets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    supplier_id INTEGER NOT NULL DEFAULT 0,
    channel_id INTEGER NOT NULL DEFAULT 0,
    name VARCHAR(255) NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 1,
    start_time INTEGER NOT NULL DEFAULT 0,
    end_time INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (supplier_id) REFERENCES suppliers(id) ON DELETE CASCADE
);

-- Supplier pricing items table (unique index on pricing_sheet_id + model)
CREATE TABLE IF NOT EXISTS supplier_pricing_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pricing_sheet_id INTEGER NOT NULL DEFAULT 0,
    model VARCHAR(255) NOT NULL DEFAULT '',
    discount_type VARCHAR(32) NOT NULL DEFAULT '',
    discount_value REAL NOT NULL DEFAULT 0,
    remark TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (pricing_sheet_id) REFERENCES supplier_pricing_sheets(id) ON DELETE CASCADE
);

-- Create unique index on pricing_sheet_id + model
CREATE UNIQUE INDEX IF NOT EXISTS idx_supplier_sheet_model ON supplier_pricing_items(pricing_sheet_id, model);
