// Supplier Stats module type definitions

// ============================================================================
// API 0: Overview
// ============================================================================

export interface SupplierStatsOverviewResult {
  total_charge: number
  total_cost: number
  gross_profit: number
  profit_margin: number
  request_count: number
  supplier_count: number
  channel_count: number
}

// ============================================================================
// API 1: By Supplier
// ============================================================================

export interface SupplierStatsBySupplierItem {
  supplier_id: number
  supplier_name: string
  total_charge: number
  total_cost: number
  gross_profit: number
  profit_margin: number
  request_count: number
}

export interface SupplierStatsBySupplierResult {
  items: SupplierStatsBySupplierItem[]
  total_charge: number
  total_cost: number
  total_profit: number
}

// ============================================================================
// API 2: By Channel
// ============================================================================

export interface SupplierStatsByChannelItem {
  supplier_id: number
  supplier_name: string
  channel_id: number
  channel_name: string
  total_charge: number
  total_cost: number
  gross_profit: number
  profit_margin: number
  request_count: number
}

export interface SupplierStatsByChannelResult {
  items: SupplierStatsByChannelItem[]
  total_charge: number
  total_cost: number
  total_profit: number
}

// ============================================================================
// API 3: By Model
// ============================================================================

export interface SupplierStatsByModelItem {
  supplier_id: number
  supplier_name: string
  channel_id: number
  channel_name: string
  model_name: string
  date: string
  total_charge: number
  total_cost: number
  gross_profit: number
  profit_margin: number
  request_count: number
}

export interface SupplierStatsByModelResult {
  items: SupplierStatsByModelItem[]
  total_charge: number
  total_cost: number
  total_profit: number
}

// ============================================================================
// Shared
// ============================================================================

export interface SupplierStatsParams {
  start_timestamp?: number
  end_timestamp?: number
}

// ============================================================================
// Processed Chart Data Types
// ============================================================================

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type SupplierVChartSpec = Record<string, any>

export interface ProcessedSupplierChartData {
  spec_supplier_rank: SupplierVChartSpec   // 供应商消耗排行
  spec_channel_rank: SupplierVChartSpec   // 渠道消耗排行
  spec_popular_channel: SupplierVChartSpec // 热门渠道
}
