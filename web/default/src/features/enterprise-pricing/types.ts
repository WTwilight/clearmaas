import { z } from 'zod'

// ============================================================================
// Enterprise
// ============================================================================

export const enterpriseSchema = z.object({
  id: z.number(),
  name: z.string(),
  status: z.number(),
  remark: z.string().optional(),
  created_at: z.number().optional(),
  updated_at: z.number().optional(),
})
export type Enterprise = z.infer<typeof enterpriseSchema>

// ============================================================================
// Pricing Sheet
// ============================================================================

export const pricingSheetSchema = z.object({
  id: z.number(),
  enterprise_id: z.number(),
  name: z.string(),
  status: z.number(),
  start_time: z.number(),
  end_time: z.number(),
  created_at: z.number().optional(),
  updated_at: z.number().optional(),
})
export type PricingSheet = z.infer<typeof pricingSheetSchema>

// ============================================================================
// Pricing Item
// ============================================================================

export const pricingItemSchema = z.object({
  id: z.number(),
  pricing_sheet_id: z.number(),
  model: z.string(),
  discount_type: z.enum(['ratio', 'fixed_price']),
  discount_value: z.number(),
  remark: z.string().optional(),
})
export type PricingItem = z.infer<typeof pricingItemSchema>

// ============================================================================
// Enterprise User Binding
// ============================================================================

export const enterpriseUserBindingSchema = z.object({
  id: z.number(),
  enterprise_id: z.number(),
  user_id: z.number(),
  created_at: z.number().optional(),
})
export type EnterpriseUserBinding = z.infer<typeof enterpriseUserBindingSchema>

// Extended binding with user info
export const enterpriseUserBindingWithUserSchema =
  enterpriseUserBindingSchema.extend({
    username: z.string(),
    display_name: z.string().optional(),
  })
export type EnterpriseUserBindingWithUser = z.infer<
  typeof enterpriseUserBindingWithUserSchema
>

// User with binding info returned from enterprise users API
export interface EnterpriseUserWithBinding {
  user_id: number
  username: string
  display_name?: string
  created_at?: number
}

// ============================================================================
// API Response Types
// ============================================================================

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export interface PaginatedResponse<T> {
  success: boolean
  message?: string
  data?: {
    items: T[]
    total: number
    page: number
    page_size: number
  }
}

// ============================================================================
// Enterprise CRUD
// ============================================================================

export interface GetEnterprisesParams {
  p?: number
  page_size?: number
}

export interface CreateEnterpriseData {
  name: string
  status?: number
  remark?: string
}

export interface UpdateEnterpriseData extends CreateEnterpriseData {
  id: number
}

// ============================================================================
// Pricing Sheet CRUD
// ============================================================================

export interface GetSheetsParams {
  p?: number
  page_size?: number
}

export interface CreateSheetData {
  name: string
  status?: number
  start_time?: number
  end_time?: number
}

export interface UpdateSheetData extends CreateSheetData {
  id: number
}

// ============================================================================
// Pricing Item CRUD
// ============================================================================

export interface CreatePricingItemData {
  model: string
  discount_type: 'ratio' | 'fixed_price'
  discount_value: number
  remark?: string
}

export interface UpdatePricingItemData extends CreatePricingItemData {
  id: number
}

// ============================================================================
// Pricing Sheet With Enterprise
// ============================================================================

export interface PricingSheetWithEnterprise extends PricingSheet {
  enterprise_name?: string
}

// ============================================================================
// User Binding
// ============================================================================

export interface BindUserData {
  user_id: number
}
