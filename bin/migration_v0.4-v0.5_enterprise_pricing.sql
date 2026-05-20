-- Migration: Add enterprise pricing sheet tables
-- This migration creates the TO B enterprise pricing sheet feature tables.
-- Compatible with: PostgreSQL

-- ============================================================
-- Table 1: enterprise — 企业主体
-- ============================================================
CREATE TABLE IF NOT EXISTS enterprises (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status INTEGER DEFAULT 1,
    remark TEXT,
    created_at BIGINT,
    updated_at BIGINT
);

COMMENT ON COLUMN enterprises.status IS '1=enabled, 0=disabled';

-- ============================================================
-- Table 2: enterprise_pricing_sheet — 报价单
-- ============================================================
CREATE TABLE IF NOT EXISTS enterprise_pricing_sheets (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    status INTEGER DEFAULT 1,
    start_time BIGINT,
    end_time BIGINT,
    created_at BIGINT,
    updated_at BIGINT
);

COMMENT ON COLUMN enterprise_pricing_sheets.name IS '报价单名称，如"XX公司2024年报价"';
COMMENT ON COLUMN enterprise_pricing_sheets.status IS '1=active, 0=inactive';
COMMENT ON COLUMN enterprise_pricing_sheets.start_time IS '生效时间戳';
COMMENT ON COLUMN enterprise_pricing_sheets.end_time IS '失效时间戳';

CREATE INDEX IF NOT EXISTS idx_enterprise_sheet_enterprise_id ON enterprise_pricing_sheets(enterprise_id);
CREATE INDEX IF NOT EXISTS idx_enterprise_sheet_status_time ON enterprise_pricing_sheets(status, start_time, end_time);

-- ============================================================
-- Table 3: enterprise_pricing_item — 报价明细
-- ============================================================
CREATE TABLE IF NOT EXISTS enterprise_pricing_items (
    id BIGSERIAL PRIMARY KEY,
    pricing_sheet_id BIGINT NOT NULL,
    model VARCHAR(128) NOT NULL,
    discount_type VARCHAR(32) NOT NULL,
    discount_value DECIMAL(10,4) NOT NULL,
    remark VARCHAR(255)
);

COMMENT ON COLUMN enterprise_pricing_items.model IS '模型名，如"gpt-4o"';
COMMENT ON COLUMN enterprise_pricing_items.discount_type IS '"ratio"=折扣倍率, "fixed_price"=固定价格';
COMMENT ON COLUMN enterprise_pricing_items.discount_value IS '折扣值: 0.7 或 0.004';

CREATE INDEX IF NOT EXISTS idx_pricing_item_sheet_id ON enterprise_pricing_items(pricing_sheet_id);
CREATE INDEX IF NOT EXISTS idx_pricing_item_model ON enterprise_pricing_items(model);
CREATE UNIQUE INDEX IF NOT EXISTS uk_sheet_model ON enterprise_pricing_items(pricing_sheet_id, model);

-- ============================================================
-- Table 4: enterprise_user_binding — 企业-用户绑定
-- ============================================================
CREATE TABLE IF NOT EXISTS enterprise_user_bindings (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    created_at BIGINT,
    CONSTRAINT uk_enterprise_user UNIQUE (enterprise_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_binding_user_id ON enterprise_user_bindings(user_id);
