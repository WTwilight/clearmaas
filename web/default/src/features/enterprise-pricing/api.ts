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
  EnterpriseUserBinding,
  EnterpriseUserWithBinding,
} from './types'

// ============================================================================
// Enterprise APIs
// ============================================================================

export async function getEnterprises(
  params: GetEnterprisesParams = {}
): Promise<PaginatedResponse<Enterprise>> {
  const { p = 1, page_size = 10 } = params
  const qs = new URLSearchParams({ p: String(p), page_size: String(page_size) })
  if (params.include_platform !== undefined) {
    qs.set('include_platform', String(params.include_platform))
  }
  const res = await api.get(`/api/enterprise?${qs}`)
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

export async function getAllUserBindings(): Promise<
  ApiResponse<EnterpriseUserBinding[]>
> {
  const res = await api.get('/api/enterprise/bindings/all')
  return res.data
}

// ============================================================================
// Pricing Item APIs
// ============================================================================

export async function getPricingItems(
  sheetId: number,
  params: { p?: number; page_size?: number } = {}
): Promise<{ success: boolean; data?: { items: PricingItem[]; total: number; page: number; page_size: number }; message?: string }> {
  const { p = 1, page_size = 20 } = params
  const res = await api.get(`/api/pricing-sheet/${sheetId}/item`, {
    params: { p, page_size },
  })
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
  const res = await api.get('/api/models/enabled', {
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return res.data?.data || []
}

// Utility: Get models for a specific pricing sheet (scoped to bound channels)
export async function getSheetModels(
  sheetId: number,
): Promise<{ success: boolean; data: string[]; channel_count: number }> {
  const res = await api.get(`/api/pricing-sheet/${sheetId}/models`)
  return res.data
}

// Utility: Get models directly by channel IDs (comma-separated)
export async function getModelsByChannelIds(
  channelIds: number[],
): Promise<{ success: boolean; data: string[]; channel_count: number }> {
  const res = await api.get(`/api/models/by-channels?channel_ids=${channelIds.join(',')}`)
  return res.data
}

// ============================================================================
// Pricing Sheet Channel Binding APIs
// ============================================================================

export async function getSheetChannels(sheetId: number): Promise<number[]> {
  const res = await api.get(`/api/pricing-sheet/${sheetId}/channels`)
  if (!res.data?.success) throw new Error(res.data?.message ?? 'Failed to fetch channels')
  return res.data?.data ?? []
}

export async function bindSheetChannels(
  sheetId: number,
  channelIds: number[]
): Promise<void> {
  const res = await api.post(`/api/pricing-sheet/${sheetId}/channels`, {
    channel_ids: channelIds,
  })
  if (!res.data?.success) throw new Error(res.data?.message ?? 'Failed to bind channels')
}

export async function unbindSheetChannel(
  sheetId: number,
  channelId: number
): Promise<void> {
  const res = await api.delete(`/api/pricing-sheet/${sheetId}/channels/${channelId}`)
  if (!res.data?.success) throw new Error(res.data?.message ?? 'Failed to unbind channel')
}

// ============================================================================
// Pricing Sheet Token Binding APIs
// ============================================================================

export interface SheetTokenBinding {
  id: number
  user_id: number
  username: string
  name: string
  status: number
  key: string
  created_time: number
  accessed_time: number
  token_group: string
  binding_created_at: number
  models: string[]
}

export async function getSheetTokenBindings(
  sheetId: number
): Promise<SheetTokenBinding[]> {
  const res = await api.get(`/api/pricing-sheet/${sheetId}/token-bindings`)
  if (!res.data?.success) throw new Error(res.data?.message ?? 'Failed to fetch token bindings')
  return res.data?.data ?? []
}

export async function unbindTokenPricingBinding(
  tokenId: number
): Promise<void> {
  const res = await api.delete(`/api/token/${tokenId}/pricing-models`)
  if (!res.data?.success) throw new Error(res.data?.message ?? 'Failed to unbind token pricing')
}
