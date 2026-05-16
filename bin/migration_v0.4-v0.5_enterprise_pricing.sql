-- Migration: Add enterprise pricing sheet tables
-- This migration creates the TO B enterprise pricing sheet feature tables.
-- Compatible with: SQLite, MySQL, PostgreSQL

-- ============================================================
-- Table 1: enterprise — 企业主体
-- ============================================================
CREATE TABLE IF NOT EXISTS enterprises (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) NOT NULL,
    status INTEGER DEFAULT 1 COMMENT '1=enabled, 0=disabled',
    remark TEXT,
    created_at BIGINT,
    updated_at BIGINT
);

-- ============================================================
-- Table 2: enterprise_pricing_sheet — 报价单
-- ============================================================
CREATE TABLE IF NOT EXISTS enterprise_pricing_sheets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    enterprise_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL COMMENT '报价单名称，如"XX公司2024年报价"',
    status INTEGER DEFAULT 1 COMMENT '1=active, 0=inactive',
    start_time BIGINT COMMENT '生效时间戳',
    end_time BIGINT COMMENT '失效时间戳',
    created_at BIGINT,
    updated_at BIGINT,
    INDEX idx_enterprise_id (enterprise_id),
    INDEX idx_status_time (status, start_time, end_time)
);

-- ============================================================
-- Table 3: enterprise_pricing_item — 报价明细
-- ============================================================
CREATE TABLE IF NOT EXISTS enterprise_pricing_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pricing_sheet_id INTEGER NOT NULL,
    model VARCHAR(128) NOT NULL COMMENT '模型名，如"gpt-4o"',
    discount_type VARCHAR(32) NOT NULL COMMENT '"ratio"=折扣倍率, "fixed_price"=固定价格',
    discount_value DECIMAL(10,4) NOT NULL COMMENT '折扣值: 0.7 或 0.004',
    remark VARCHAR(255),
    INDEX idx_pricing_sheet_id (pricing_sheet_id),
    INDEX idx_model (model),
    UNIQUE KEY uk_sheet_model (pricing_sheet_id, model)
);

-- ============================================================
-- Table 4: enterprise_user_binding — 企业-用户绑定
-- ============================================================
CREATE TABLE IF NOT EXISTS enterprise_user_bindings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    enterprise_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    created_at BIGINT,
    UNIQUE KEY uk_enterprise_user (enterprise_id, user_id),
    INDEX idx_user_id (user_id)
);
