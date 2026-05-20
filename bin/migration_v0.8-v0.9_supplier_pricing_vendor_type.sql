-- Migration: Add vendor_type column to supplier_pricing_items
-- Aligns with enterprise_pricing_items structure which already has vendor_type.
-- Compatible with: PostgreSQL

ALTER TABLE supplier_pricing_items ADD COLUMN vendor_type VARCHAR(32) NOT NULL DEFAULT '';
