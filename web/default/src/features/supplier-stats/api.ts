import { api } from '@/lib/api'
import type {
  SupplierStatsParams,
  SupplierStatsOverviewResult,
  SupplierStatsBySupplierResult,
  SupplierStatsByChannelResult,
  SupplierStatsByModelResult,
} from './types'

export async function getSupplierStatsOverview(params: SupplierStatsParams = {}) {
  const res = await api.get<{ success: boolean; message?: string; data: SupplierStatsOverviewResult }>(
    '/api/supplier-stats/overview',
    { params }
  )
  return res.data
}

export async function getSupplierStatsBySupplier(params: SupplierStatsParams = {}) {
  const res = await api.get<{ success: boolean; message?: string; data: SupplierStatsBySupplierResult }>(
    '/api/supplier-stats/by-supplier',
    { params }
  )
  return res.data
}

export async function getSupplierStatsByChannel(params: SupplierStatsParams = {}) {
  const res = await api.get<{ success: boolean; message?: string; data: SupplierStatsByChannelResult }>(
    '/api/supplier-stats/by-channel',
    { params }
  )
  return res.data
}

export async function getSupplierStatsByModel(params: SupplierStatsParams = {}) {
  const res = await api.get<{ success: boolean; message?: string; data: SupplierStatsByModelResult }>(
    '/api/supplier-stats/by-model',
    { params }
  )
  return res.data
}
