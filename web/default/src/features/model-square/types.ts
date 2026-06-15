/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export type ModelSquareVendor = {
  id: number
  name: string
  icon?: string
}

export type ModelSquarePriceInfo = {
  input_ratio: number
  output_ratio: number
  quota_type: number
  model_price?: number
  discount_ratio?: number
  input_original_price?: number
  output_original_price?: number
  input_discounted_price?: number
  output_discounted_price?: number
  ratio_source?: string
}

export type ModelSquareVersion = {
  model_name: string
  upstream_key: string
  channel_id: number
  channel_name: string
  channel_tags?: string[]
  channel_type: number
  status: 'enabled' | 'disabled' | string
  priority: number
  mode_description?: string
  discount_percent?: number
  discount_ratio?: number
  ratio_source?: string
  tps?: number
  latency_seconds?: number
  success_rate?: number
  request_count?: number
  updated_time?: number
  price_info?: ModelSquarePriceInfo
}

export type ModelSquareItem = {
  id: number
  name: string
  display_name: string
  description?: string
  icon?: string
  vendor_id?: number
  vendor_name?: string
  vendor_icon?: string
  tags: string[]
  input_types?: string[]
  output_types?: string[]
  parameters?: string[]
  protocols?: string[]
  reasoning?: string
  context_tokens?: number
  max_output?: number
  discount_percent?: number
  request_count?: number
  updated_time?: number
  versions: ModelSquareVersion[]
  version_count: number
  price_info?: ModelSquarePriceInfo
  capabilities?: {
    supported_endpoints?: string[]
    streaming?: boolean
    vision?: boolean
  }
}

export type ModelSquareData = {
  models: ModelSquareItem[]
  vendors: ModelSquareVendor[]
  tags: string[]
  total: number
}

export type ModelSquareResponse = {
  success: boolean
  message?: string
  data: ModelSquareData
}

export type ModelSquareParams = {
  keyword?: string
  vendor_id?: string
  tag?: string
}
