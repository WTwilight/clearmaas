import { axiosClient } from '@/lib/axios';
import type {
  Supplier,
  SupplierPricingSheet,
  SupplierPricingItem,
  SupplierListResponse,
  SupplierPricingSheetListResponse,
  ApiResponse,
} from './types';

// ---------------------------------------------------------------------------
// Supplier APIs
// ---------------------------------------------------------------------------

export async function getSuppliers(params?: {
  p?: number;
  page_size?: number;
}): Promise<ApiResponse<SupplierListResponse>> {
  const { data } = await axiosClient.get<ApiResponse<SupplierListResponse>>('/api/supplier', { params });
  return data;
}

export async function getSupplier(id: number): Promise<Supplier> {
  const { data } = await axiosClient.get<ApiResponse<Supplier>>(`/api/supplier/${id}`);
  if (!data.success) throw new Error(data.message);
  return data.data!;
}

export async function createSupplier(payload: {
  name: string;
  status?: number;
  remark?: string;
}): Promise<Supplier> {
  const { data } = await axiosClient.post<ApiResponse<Supplier>>('/api/supplier', payload);
  if (!data.success) throw new Error(data.message);
  return data.data!;
}

export async function updateSupplier(
  id: number,
  payload: {
    name: string;
    status: number;
    remark?: string;
  }
): Promise<Supplier> {
  const { data } = await axiosClient.put<ApiResponse<Supplier>>(`/api/supplier/${id}`, payload);
  if (!data.success) throw new Error(data.message);
  return data.data!;
}

export async function deleteSupplier(id: number): Promise<void> {
  const { data } = await axiosClient.delete<ApiResponse>(`/api/supplier/${id}`);
  if (!data.success) throw new Error(data.message);
}

// ---------------------------------------------------------------------------
// SupplierPricingSheet APIs
// ---------------------------------------------------------------------------

export async function getSupplierPricingSheets(params?: {
  p?: number;
  page_size?: number;
}): Promise<ApiResponse<SupplierPricingSheetListResponse>> {
  const { data } = await axiosClient.get<ApiResponse<SupplierPricingSheetListResponse>>(
    '/api/supplier-pricing-sheets',
    { params }
  );
  return data;
}

export async function getSupplierPricingSheet(
  supplierId: number,
  sheetId: number
): Promise<SupplierPricingSheet> {
  const { data } = await axiosClient.get<ApiResponse<SupplierPricingSheet>>(
    `/api/supplier/${supplierId}/pricing-sheet/${sheetId}`
  );
  if (!data.success) throw new Error(data.message);
  return data.data!;
}

export async function getSupplierPricingSheetsBySupplier(
  supplierId: number
): Promise<{ items: SupplierPricingSheet[]; total: number }> {
  const { data } = await axiosClient.get<ApiResponse<{ items: SupplierPricingSheet[]; total: number }>>(
    `/api/supplier/${supplierId}/pricing-sheet`
  );
  return data.data ?? { items: [], total: 0 };
}

export async function createSupplierPricingSheet(
  supplierId: number,
  payload: {
    name: string;
    status?: number;
    channel_id?: number;
    start_time?: number;
    end_time?: number;
  }
): Promise<SupplierPricingSheet> {
  const { data } = await axiosClient.post<ApiResponse<SupplierPricingSheet>>(
    `/api/supplier/${supplierId}/pricing-sheet`,
    payload
  );
  if (!data.success) throw new Error(data.message);
  return data.data!;
}

export async function updateSupplierPricingSheet(
  supplierId: number,
  sheetId: number,
  payload: {
    name: string;
    status: number;
    channel_id?: number;
    start_time?: number;
    end_time?: number;
  }
): Promise<SupplierPricingSheet> {
  const { data } = await axiosClient.put<ApiResponse<SupplierPricingSheet>>(
    `/api/supplier/${supplierId}/pricing-sheet/${sheetId}`,
    payload
  );
  if (!data.success) throw new Error(data.message);
  return data.data!;
}

export async function deleteSupplierPricingSheet(
  supplierId: number,
  sheetId: number
): Promise<void> {
  const { data } = await axiosClient.delete<ApiResponse>(
    `/api/supplier/${supplierId}/pricing-sheet/${sheetId}`
  );
  if (!data.success) throw new Error(data.message);
}

// ---------------------------------------------------------------------------
// SupplierPricingItem APIs
// ---------------------------------------------------------------------------

export async function getSupplierPricingItems(
  supplierId: number,
  sheetId: number
): Promise<ApiResponse<SupplierPricingItem[]>> {
  const { data } = await axiosClient.get<ApiResponse<SupplierPricingItem[]>>(
    `/api/supplier/${supplierId}/pricing-sheet/${sheetId}/items`
  );
  return data;
}

export async function createSupplierPricingItem(
  supplierId: number,
  sheetId: number,
  payload: {
    vendor_type: string;
    models: string[];
    discount_type: string;
    discount_value: number;
    remark?: string;
  }
): Promise<SupplierPricingItem> {
  const { data } = await axiosClient.post<ApiResponse<SupplierPricingItem>>(
    `/api/supplier/${supplierId}/pricing-sheet/${sheetId}/items`,
    payload
  );
  if (!data.success) throw new Error(data.message);
  return data.data!;
}

export async function updateSupplierPricingItem(
  supplierId: number,
  sheetId: number,
  itemId: number,
  payload: {
    models: string[];
    discount_type: string;
    discount_value: number;
    remark?: string;
  }
): Promise<SupplierPricingItem> {
  const { data } = await axiosClient.put<ApiResponse<SupplierPricingItem>>(
    `/api/supplier/${supplierId}/pricing-sheet/${sheetId}/items/${itemId}`,
    payload
  );
  if (!data.success) throw new Error(data.message);
  return data.data!;
}

export async function deleteSupplierPricingItem(
  supplierId: number,
  sheetId: number,
  itemId: number
): Promise<void> {
  const { data } = await axiosClient.delete<ApiResponse>(
    `/api/supplier/${supplierId}/pricing-sheet/${sheetId}/items/${itemId}`
  );
  if (!data.success) throw new Error(data.message);
}

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

export async function getEnabledModels(): Promise<string[]> {
  const { data } = await axiosClient.get('/api/models/enabled');
  return data.data || [];
}
