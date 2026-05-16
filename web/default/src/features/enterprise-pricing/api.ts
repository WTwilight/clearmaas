import { api } from '@/lib/api'
import type {
  Enterprise,
  PricingSheet,
  PricingItem,
  ApiResponse,
  PaginatedResponse,
  GetEnterprisesParams,
  CreateEnterpriseData,
  UpdateEnterpriseData,
  GetSheetsParams,
  CreateSheetData,
  UpdateSheetData,
  CreatePricingItemData,
  UpdatePricingItemData,
  BindUserData,
  EnterpriseUserWithBinding,
} from './types'

// ============================================================================
// Enterprise APIs
// ============================================================================

export async function getEnterprises(
  params: GetEnterprisesParams = {}
): Promise<PaginatedResponse<Enterprise>> {
  const { p = 1, page_size = 10 } = params
  const res = await api.get(`/api/enterprise?p=${p}&page_size=${page_size}`)
  return res.data
}

export async function getEnterprise(
  id: number
): Promise<ApiResponse<Enterprise>> {
  const res = await api.get(`/api/enterprise/${id}`)
  return res.data
}

export async function createEnterprise(
  data: CreateEnterpriseData
): Promise<ApiResponse<Enterprise>> {
  const res = await api.post('/api/enterprise', data)
  return res.data
}

export async function updateEnterprise(
  data: UpdateEnterpriseData
): Promise<ApiResponse<Partial<Enterprise>>> {
  const res = await api.put(`/api/enterprise/${data.id}`, data)
  return res.data
}

export async function deleteEnterprise(
  id: number
): Promise<ApiResponse> {
  const res = await api.delete(`/api/enterprise/${id}`)
  return res.data
}

// ============================================================================
// Pricing Sheet APIs
// ============================================================================

// PricingSheetWithEnterprise represents a pricing sheet with enterprise info.
export interface PricingSheetWithEnterprise extends PricingSheet {
  enterprise_name?: string
}

export async function getAllSheets(
  params: { p?: number; page_size?: number; enterprise_id?: number } = {}
): Promise<{ success: boolean; data?: { items: PricingSheetWithEnterprise[]; total: number; page: number; page_size: number } }> {
  const { p = 1, page_size = 20, enterprise_id } = params
  let url = `/api/pricing-sheet?p=${p}&page_size=${page_size}`
  if (enterprise_id) {
    url += `&enterprise_id=${enterprise_id}`
  }
  const res = await api.get(url)
  return res.data
}

export async function getSheets(
  enterpriseId: number,
  params: GetSheetsParams = {}
): Promise<PaginatedResponse<PricingSheet>> {
  const { p = 1, page_size = 10 } = params
  const res = await api.get(
    `/api/enterprise/${enterpriseId}/pricing-sheet?p=${p}&page_size=${page_size}`
  )
  return res.data
}

export async function getSheet(
  enterpriseId: number,
  sheetId: number
): Promise<ApiResponse<PricingSheet>> {
  const res = await api.get(
    `/api/enterprise/${enterpriseId}/pricing-sheet/${sheetId}`
  )
  return res.data
}

export async function createSheet(
  enterpriseId: number,
  data: CreateSheetData
): Promise<ApiResponse<PricingSheet>> {
  const res = await api.post(`/api/enterprise/${enterpriseId}/pricing-sheet`, data)
  return res.data
}

export async function updateSheet(
  enterpriseId: number,
  data: UpdateSheetData
): Promise<ApiResponse<Partial<PricingSheet>>> {
  const res = await api.put(
    `/api/enterprise/${enterpriseId}/pricing-sheet/${data.id}`,
    data
  )
  return res.data
}

export async function deleteSheet(
  enterpriseId: number,
  sheetId: number
): Promise<ApiResponse> {
  const res = await api.delete(`/api/enterprise/${enterpriseId}/pricing-sheet/${sheetId}`)
  return res.data
}

// ============================================================================
// Enterprise User Binding APIs
// ============================================================================

export async function getEnterpriseUsers(
  enterpriseId: number
): Promise<ApiResponse<EnterpriseUserWithBinding[]>> {
  const res = await api.get(`/api/enterprise/${enterpriseId}/users`)
  return res.data
}

export async function bindUser(
  enterpriseId: number,
  data: BindUserData
): Promise<ApiResponse> {
  const res = await api.post(`/api/enterprise/${enterpriseId}/users`, data)
  return res.data
}

export async function unbindUser(
  enterpriseId: number,
  userId: number
): Promise<ApiResponse> {
  const res = await api.delete(`/api/enterprise/${enterpriseId}/users/${userId}`)
  return res.data
}

// ============================================================================
// Pricing Item APIs
// ============================================================================

export async function getPricingItems(
  sheetId: number
): Promise<ApiResponse<PricingItem[]>> {
  const res = await api.get(`/api/pricing-sheet/${sheetId}/item`)
  return res.data
}

export async function createPricingItem(
  sheetId: number,
  data: CreatePricingItemData
): Promise<ApiResponse<PricingItem>> {
  const res = await api.post(`/api/pricing-sheet/${sheetId}/item`, data)
  return res.data
}

export async function updatePricingItem(
  sheetId: number,
  data: UpdatePricingItemData
): Promise<ApiResponse<Partial<PricingItem>>> {
  const res = await api.put(`/api/pricing-sheet/${sheetId}/item/${data.id}`, data)
  return res.data
}

export async function deletePricingItem(
  sheetId: number,
  itemId: number
): Promise<ApiResponse> {
  const res = await api.delete(`/api/pricing-sheet/${sheetId}/item/${itemId}`)
  return res.data
}

// ============================================================================
// Utility: List all users (for binding dropdown)
// ============================================================================

export async function searchAllUsers(
  keyword = ''
): Promise<Array<{ id: number; username: string; display_name?: string }>> {
  const res = await api.get(
    `/api/user/search?keyword=${keyword}&p=1&page_size=100`,
    { skipErrorHandler: true } as Record<string, unknown>
  )
  if (!res.data?.data?.items) return []
  return res.data.data.items
}

// ============================================================================
// Utility: Get enabled model names (for pricing item model selector)
// ============================================================================

export async function getEnabledModels(): Promise<string[]> {
  const res = await api.get('/api/channel/models_enabled', {
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return res.data?.data || []
}
