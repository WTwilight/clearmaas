// Supplier Pricing module type definitions

export interface Supplier {
  id: number;
  name: string;
  status: number;
  remark: string;
  created_at: number;
  updated_at: number;
}

export interface SupplierPricingSheet {
  id: number;
  supplier_id: number;
  supplier_name?: string;
  channel_id: number;
  channel_name?: string;
  channel_ids?: number[];
  channel_names?: string[];
  name: string;
  status: number;
  start_time: number;
  end_time: number;
  created_at: number;
  updated_at: number;
}

export interface SupplierPricingItem {
  id: number;
  pricing_sheet_id: number;
  vendor_type: string;
  models: string[];
  discount_type: string;
  discount_value: number;
  remark: string;
}

export interface SupplierListResponse {
  items: Supplier[];
  total: number;
  page: number;
  page_size: number;
}

export interface SupplierPricingSheetListResponse {
  items: SupplierPricingSheet[];
  total: number;
  page: number;
  page_size: number;
}

export interface ApiResponse<T = unknown> {
  success: boolean;
  message?: string;
  data?: T;
}

export type DiscountType = 'ratio' | 'fixed_price' | 'per_call';

export type SupplierStatus = 0 | 1;
export type SheetStatus = 0 | 1;

// Channel type (matches channels feature types)
export interface Channel {
  id: number;
  name: string;
  type: number;
  status: number;
  base_url?: string;
  created_at?: number;
  updated_at?: number;
}
